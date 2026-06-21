package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
)

type Decision struct {
	Signal    string
	Reasoning string
}

type Engine struct {
	bedrock *bedrockruntime.Client
	modelID string
}

func NewEngine(cfg aws.Config) *Engine {
	// 如果不知道自己可用的模型有哪些可以用以下指令達到
	// # region 可以換成自己的
	// # profile 請用自己的
	// aws bedrock list-inference-profiles \
	//   --region ap-east-2 \
	//   --profile AdministratorAccess-xxxx \
	//   --query 'inferenceProfileSummaries[*].{id:inferenceProfileId,name:inferenceProfileName}' \
	//   --output table
	return &Engine{
		bedrock: bedrockruntime.NewFromConfig(cfg),
		modelID: "apac.amazon.nova-lite-v1:0",
	}
}

func (e *Engine) Evaluate(ctx context.Context, prompt string) (*Decision, error) {
	output, err := e.bedrock.Converse(ctx, &bedrockruntime.ConverseInput{
		ModelId: aws.String(e.modelID),
		Messages: []types.Message{
			{
				Role: types.ConversationRoleUser,
				Content: []types.ContentBlock{
					&types.ContentBlockMemberText{
						Value: prompt,
					},
				},
			},
		},
		InferenceConfig: &types.InferenceConfiguration{
			MaxTokens: aws.Int32(1024),
		},
	})
	if err != nil {
		return nil, err
	}
	outputMemberMessage, ok := output.Output.(*types.ConverseOutputMemberMessage)
	if !ok {
		return nil, fmt.Errorf("unexpected output type of converse api")
	}
	// 這裡處理處理 :
	var resultString strings.Builder
	for _, block := range outputMemberMessage.Value.Content {
		textBlock, ok := block.(*types.ContentBlockMemberText)
		if !ok {
			continue
		}
		resultString.WriteString(textBlock.Value)
	}
	// 這裡看看結果如何 :
	fmt.Println("==============================")
	fmt.Printf("\033[032m%s\033[0m\n", resultString.String())
	fmt.Println("==============================")
	// 這裡轉換一下 json 結果
	var result Decision
	err = json.Unmarshal([]byte(resultString.String()), &result)
	if err != nil {
		return nil, fmt.Errorf("json unmarshal error : %v", err)
	}

	return &result, nil
}

func main() {
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
	prmopt := fmt.Sprintf(promptTemplate, 25) // 可以改動這裡的數值，看看結果如何

	ctx := context.Background()
	config, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("ap-east-2"), // 這裡選用自己的 region
		config.WithSharedConfigProfile("<這裡輸入自己的 profile>"),
	)
	if err != nil {
		log.Fatalf("failed to load config : %v\n", err)
		return
	}
	engine := NewEngine(config)
	result, err := engine.Evaluate(ctx, prmopt)
	if err != nil {
		log.Fatalf("converse api error : %v\n", err)
		return
	}

	fmt.Printf("「危險程度」: %s\n", result.Signal)
	fmt.Printf("「推理過程」: %s\n", result.Reasoning)
}
