package main

import (
	"context"
	"fmt"
	"log"
	"michleo851a1203/ecs-aiopsworkshop/pkg/actuator"
	"michleo851a1203/ecs-aiopsworkshop/pkg/ai"
	"michleo851a1203/ecs-aiopsworkshop/pkg/metric"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
)

func main() {
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("ap-east-2"),
		config.WithSharedConfigProfile("AdministratorAccess-687126124212"),
	)

	if err != nil {
		log.Fatalf("load config error : %v\n", err)
		return
	}

	// ==============================
	cluster := "<你的 cluster>"
	blueService := "api-blue"
	greenService := "api-green"
	namespace := "<你的 namespace>"

	serviceNames := map[string]string{
		"api-blue":  "<api-blue-service-name>",
		"api-green": "<api-green-service-name>",
	}

	// ALB (AWS/ApplicationELB) 維度值 —— 主要判斷依據
	// 這個值: 進到 loadbalancer arn : 後面會長這樣 arn:aws:elasticloadbalancing:ap-east-2:687126124212:loadbalancer/app/demo-alb/0df4f07d47294b76
	// 取後面的 : app/demo-alb/0df4f07d47294b76
	albLoadBalancer := "<alb 維度值>" // app/demo-alb/0df4f07d47294b76
	// 這個值: 進到 target group arn : 後面會長這樣 arn:aws:elasticloadbalancing:ap-east-2:687126124212:targetgroup/dmeo-tg/b1b55f2d05892872
	// 取後面的 : targetgroup/dmeo-tg/b1b55f2d05892872
	albTargetGroup := "<target group維度值>" // targetgroup/dmeo-tg/b1b55f2d05892872

	errorRateThreshold := 80.0
	minSampleSize := 5
	p99LatencyThreshold := 1.0 // 秒,ALB P99 超過此值視為延遲異常
	// ==============================

	promptTemplate := `
	你是一個 AIOps 金絲雀部署決策引擎。本架構為:ALB → 單一共用 target group → blue/green 兩個 ECS 服務。
	因為共用 target group,ALB 的錯誤/延遲是「整體(blue+green 混合)」數字,無法直接分版;
	Service Connect 則能以 discovery name 分別量到 blue / green 各自的請求數,用來把整體錯誤歸因到正確版本。

  ## ALB 整體指標 (主要判斷依據) - 過去 %d 分鐘 [資料可用: %v]
	- 整體 Request Count: %.0f
	- Target 5XX Count: %.0f
	- ELB 5XX Count (ALB 自身錯誤): %.0f
	- 整體 Error Rate: %.2f%%
	- P50 延遲: %.1f ms
	- P99 延遲: %.1f ms

  ## Service Connect 分版流量 (輔助,用於歸因) - 過去 %d 分鐘
	- Blue 請求數: %.0f (活躍連線 %.0f / 新建連線 %.0f)
	- Green 請求數: %.0f (活躍連線 %.0f / 新建連線 %.0f)

  ## 歸因推算 (關鍵)
	- Green 推算錯誤率 = ALB Target 5XX ÷ SC Green 請求數 = %.2f%%
	- 前提假設:canary 情境下 blue 為穩定基準 (錯誤≈0),故將 ALB 的 5XX 全部歸因給 green。

  ## Container Insights (資源面)
	- Blue : CPU %.3f vCPU / Memory %.0f MB / Running Tasks %.0f
	- Green: CPU %.3f vCPU / Memory %.0f MB / Running Tasks %.0f

  ## 決策規則(依優先順序)
  1. [立即回滾] 若「Green 推算錯誤率」> %.1f%% → rollback
  2. [立即回滾] 若 ALB 整體 Error Rate 明顯升高,且 SC 顯示 Green 正在接收流量 → rollback
  3. [延遲異常] 若 ALB P99 延遲 > %.0f ms → 降低 confidence,傾向 rollback
  4. [資源異常] 若 Green CPU 或 Memory 比 Blue 高出 2 倍以上 → 降低 confidence,傾向 rollback
  5. [樣本不足] 若 SC Green 請求數 < %.0f → hold
  6. [允許推進] 若 ALB 整體健康、Green 推算錯誤率低、延遲正常 → proceed
  7. [無法判定] 以上都不符合 → hold

	注意:若 ALB 資料不可用([資料可用: false]),只能參考 Service Connect / Container Insights,此時應降低 confidence 並傾向 hold。

	## 輸出規則（非常重要）
		- 只能輸出一個 JSON 物件，不得有任何其他文字
		- 不得使用 markdown 或程式碼區塊（禁止使用 ` + "```" + `）
		- 不得自我修正或重複輸出
		- 格式如下：

	{"action": "proceed|rollback|hold", "confidence": 0-100, "reasoning": "你的逐步推理過程"}
	`

	// ==============================

	collector := metric.NewCollector(cfg, cluster, namespace, serviceNames, albLoadBalancer, albTargetGroup)
	engine := ai.NewEngine(cfg)
	act := actuator.NewActuator(cfg, cluster, serviceNames[blueService], serviceNames[greenService])

	windowMin := 5
	ticker := time.NewTicker(15 * time.Second) // 因為這裡為了 workshop 所以設 15 秒，正常可能用個 1-3 分鐘數值會比較準確
	defer ticker.Stop()

	for range ticker.C {
		blueServiceMetric, err := collector.Collect(ctx, blueService, windowMin)
		if err != nil {
			log.Printf("blue-service metric error : %v\n", err)
			continue
		}
		greenServiceMetric, err := collector.Collect(ctx, greenService, windowMin)
		if err != nil {
			log.Printf("green-service metric error : %v\n", err)
			continue
		}
		albMetric, err := collector.CollectALB(ctx, windowMin)
		if err != nil {
			log.Printf("alb metric error : %v\n", err)
			continue
		}

		// 歸因:共用 TG 下 ALB 的 5XX 無法分版,canary 假設 blue 穩定,
		// 故把 ALB Target 5XX 全部歸因給 green,再除以 SC 量到的 green 請求數。
		greenAttrErrRate := 0.0
		if greenServiceMetric.RequestCount > 0 {
			greenAttrErrRate = albMetric.Error5XXCount / greenServiceMetric.RequestCount * 100
		}

		log.Printf("\n\033[095m%s\033[0m\n\033[094mblue SC: reqs=%.0f\033[0m  \033[032mgreen SC: reqs=%.0f → 歸因 errRate=%.2f%%\033[0m\n",
			albMetric.String(), blueServiceMetric.RequestCount, greenServiceMetric.RequestCount, greenAttrErrRate)

		prompt := fmt.Sprintf(
			promptTemplate,
			windowMin,
			albMetric.Available,
			albMetric.RequestCount,
			albMetric.Error5XXCount,
			albMetric.ELB5XXCount,
			albMetric.ErrorRate,
			albMetric.P50Latency*1000,
			albMetric.P99Latency*1000,
			windowMin,
			blueServiceMetric.RequestCount,
			blueServiceMetric.ActiveConnections,
			blueServiceMetric.NewConnections,
			greenServiceMetric.RequestCount,
			greenServiceMetric.ActiveConnections,
			greenServiceMetric.NewConnections,
			greenAttrErrRate,
			blueServiceMetric.CPUUtilized,
			blueServiceMetric.MemoryUtilized,
			blueServiceMetric.RunningTaskCount,
			greenServiceMetric.CPUUtilized,
			greenServiceMetric.MemoryUtilized,
			greenServiceMetric.RunningTaskCount,
			errorRateThreshold,
			p99LatencyThreshold*1000,
			float64(minSampleSize),
		)
		decision, err := engine.Evaluate(ctx, prompt)
		if err != nil {
			log.Printf("[Error] AI evaluation failed : %v\n", err)
			continue
		}
		log.Printf("\033[033m[AI] -> [決策:%s]<信心: %d%%>\033[0m\n", decision.Action, decision.Confidence)
		log.Printf("[AI推理過程]: %s\n", decision.Reasoning)

		if decision.Confidence > 80 {
			// fmt.Printf("\033[032m假裝執行為了測試 %s\033[0m\n", decision.Action)
			if err = act.Execute(ctx, decision.Action); err != nil {
				log.Printf("[Error]Execute failed : %v\n", err)
			}
		} else {
			log.Println("[AI表示]沒把握，等下一個週期在決策!!😅")
		}
	}
}
