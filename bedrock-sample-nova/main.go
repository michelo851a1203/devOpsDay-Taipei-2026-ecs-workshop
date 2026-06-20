package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
)

type Decision struct {
	Signal    string `json:"signal"`
	Reasoning string `json:"reasoning"`
}

// request ::==============================

type NovaRequestMessageContent struct {
	Text string `json:"text"`
}

type NovaRequestMessage struct {
	Role        string                      `json:"role"`
	ContentList []NovaRequestMessageContent `json:"content"`
}

type NovaInferenceConfig struct {
	MaxNewtokens int `json:"max_new_tokens"`
}

type NovaRequest struct {
	MessageList     []NovaRequestMessage `json:"messages"`
	InferenceConfig NovaInferenceConfig  `json:"inferenceConfig"`
}

// request:: end ==============================

// response:: ==============================

type NovaOutputMessageContent struct {
	Text string `json:"text"`
}

type NovaOutputMessage struct {
	Role        string                     `json:"role"`
	ContentList []NovaOutputMessageContent `json:"content"`
}

type NovaOuput struct {
	Message NovaOutputMessage `json:"message"`
}

type NovaUsage struct {
	InputTokens  int `json:"inputTokens"`
	OutputTokens int `json:"outputTokens"`
}

type NovaResponse struct {
	Output     NovaOuput `json:"output"`
	StopReason string    `json:"stopReason"`
	Usage      NovaUsage `json:"usage"`
}

// response:: end ==============================

// Engine:: ==============================

type Engine struct {
	bedrock *bedrockruntime.Client
	modelID string
}

func NewEngine(cfg aws.Config) *Engine {
	return &Engine{
		bedrock: bedrockruntime.NewFromConfig(cfg),
		modelID: "global.amazon.nova-2-lite-v1:0",
	}
}

func ParseResponse(output []byte) (*Decision, error) {
	fmt.Println("=============")
	fmt.Printf("\033[032m%s\033[0m\n", string(output))
	fmt.Println("=============")

	var resp NovaResponse
	err := json.Unmarshal(output, &resp)
	if err != nil {
		return nil, err
	}

	contentList := resp.Output.Message.ContentList
	if len(contentList) == 0 {
		return nil, fmt.Errorf("no content in response")
	}

	resultText := contentList[0].Text

	var decision Decision
	if err := json.Unmarshal([]byte(resultText), &decision); err != nil {
		return nil, fmt.Errorf("unmarshal decision error : %w", err)
	}

	return &decision, nil
}

func (e *Engine) Evaluate(ctx context.Context, prompt string) (*Decision, error) {
	request := NovaRequest{
		MessageList: []NovaRequestMessage{
			{
				Role:        "user",
				ContentList: []NovaRequestMessageContent{{Text: prompt}},
			},
		},
		InferenceConfig: NovaInferenceConfig{
			MaxNewtokens: 1024,
		},
	}
	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("json marshal error : %w", err)
	}

	response, err := e.bedrock.InvokeModel(ctx, &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(e.modelID),
		ContentType: aws.String("application/json"),
		Body:        body,
	})

	if err != nil {
		return nil, fmt.Errorf("bedrock Invoke error : %w", err)
	}

	return ParseResponse(response.Body)
}

// Engine:: end ==============================

func main() {
	ctx := context.Background()

	config, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("ap-east-2"), // 如果不是用這個	region 的可以替換
		config.WithSharedConfigProfile("<請輸入個人的 profile>"),
	)
	if err != nil {
		log.Fatalf("Load default config error :%v\n", err)
		return
	}

	engine := NewEngine(config)
	promptTemplate := `
你是一個危險程度決策者，根據以下情境給出決策

## 目前危險指數指標
	- danger_rate: %.0f

## 決策規則（依序檢查，第一個符合的即為結果）
	1. danger_rate > 80  → 超危險
	2. danger_rate > 50  → 危險
	3. danger_rate > 25  → 普通
	4. danger_rate <= 25 → 安全

## 輸出規則（非常重要）
	- 只能輸出一個 JSON 物件，不得有任何其他文字
	- 不得使用 markdown 或程式碼區塊（禁止使用 ` + "```" + `）
	- 不得自我修正或重複輸出
	- 格式如下：

{"signal":"超危險|危險|普通|安全","reasoning":"逐步推理過程"}
	`
	prompt := fmt.Sprintf(
		promptTemplate,
		75,
	)

	result, err := engine.Evaluate(ctx, prompt)
	if err != nil {
		log.Fatalf("evaluate reasoning error : %v\n", err)
		return
	}

	fmt.Printf("「危險程度」: %s\n", result.Signal)
	fmt.Printf("「推理過程」: %s\n", result.Reasoning)
}
