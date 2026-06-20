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

type Descision struct {
	Signal    string `json:"signal"`
	Reasoning string `json:"reasoning"`
}

type InvokeRequestMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type InvokedRequest struct {
	AnthropicVersion string                 `json:"anthropic_version"`
	MaxTokens        uint64                 `json:"max_tokens"`
	MesageList       []InvokeRequestMessage `json:"messages"`
}

type InvokeResponseContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type InvokedResponse struct {
	ID          string                  `json:"id"`
	Model       string                  `json:"model"`
	Type        string                  `json:"type"`
	Role        string                  `json:"role"`
	StopReason  string                  `json:"stop_reason"`
	ContentList []InvokeResponseContent `json:"content"`
}

type Engine struct {
	bedrock *bedrockruntime.Client
	modelID string
}

func NewEngine(cfg aws.Config) *Engine {
	return &Engine{
		bedrock: bedrockruntime.NewFromConfig(cfg),
		modelID: "global.anthropic.claude-haiku-4-5-20251001-v1:0",
	}
}

func (e Engine) Evaluate(ctx context.Context, prompt string) (*Descision, error) {
	request := InvokedRequest{
		AnthropicVersion: "bedrock-2023-05-31",
		MaxTokens:        1024,
		MesageList: []InvokeRequestMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}
	body, err := json.Marshal(request)
	if err != nil {
		log.Fatalf("start server error :%v\n", err)
	}

	response, err := e.bedrock.InvokeModel(ctx, &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(e.modelID),
		ContentType: aws.String("application/json"),
		Body:        body,
	})

	if err != nil {
		return nil, fmt.Errorf("bedrock invoke error : %w", err)
	}
	return ParseResponse(response.Body)
}

func ParseResponse(output []byte) (*Descision, error) {
	var resp InvokedResponse
	err := json.Unmarshal(output, &resp)
	if err != nil {
		return nil, err
	}

	fmt.Printf("\033[032m%s\033[0m\n", string(output))
	contentList := resp.ContentList
	if len(contentList) == 0 {
		return nil, fmt.Errorf("no content response")
	}

	content := contentList[0]
	resultText := content.Text

	var descisionResult Descision
	err = json.Unmarshal([]byte(resultText), &descisionResult)
	if err != nil {
		return nil, err
	}

	return &descisionResult, nil
}

func main() {
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(
		ctx,
		config.WithRegion("ap-east-2"), // 如果不是用這個 region 的可以替換
		config.WithSharedConfigProfile("<請輸入個人的 profile>"),
	)
	if err != nil {
		log.Fatalf("load default config error : %v\n", err)
		return
	}

	aiEngine := NewEngine(cfg)

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
	descision, err := aiEngine.Evaluate(ctx, prompt)
	if err != nil {
		log.Fatalf("[Ai] Evaluatoin error : %v\n", err)
		return
	}

	fmt.Printf("[目前危險程度]: %s\n", descision.Signal)
	fmt.Printf("[推理過程]: %s\n", descision.Reasoning)
}
