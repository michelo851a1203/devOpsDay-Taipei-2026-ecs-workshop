// Package ai for building ai decision
package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"michleo851a1203/ecs-aiopsworkshop/pkg/actuator"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
)

type Decision struct {
	Action     actuator.Action `json:"action"`
	Confidence int             `json:"confidence"`
	Reasoning  string          `json:"reasoning"`
}

// nova request ==============================

type NovaRequestMessageContent struct {
	Text string `json:"text"`
}

type NovaRequestMessage struct {
	Role    string                      `json:"role"`
	Content []NovaRequestMessageContent `json:"content"`
}

type NovaRequestInferenceConfig struct {
	MaxNewTokens int `json:"max_new_tokens"`
}

type NovaRequest struct {
	Messages        []NovaRequestMessage       `json:"messages"`
	InferenceConfig NovaRequestInferenceConfig `json:"inferenceConfig"`
}

// ==============================

// response ==============================

type NovaResponseOutputMessageContent struct {
	Text string `json:"text"`
}

type NovaResponseOutputMessage struct {
	Role    string                             `json:"role"`
	Content []NovaResponseOutputMessageContent `json:"content"`
}

type NovaResponseOutput struct {
	Message NovaResponseOutputMessage `json:"message"`
}

type NovaResponseUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type NovaResponse struct {
	Output     NovaResponseOutput `json:"output"`
	StopReason string             `json:"stopReason"`
	Usage      NovaResponseUsage  `json:"usage"`
}

// ==============================

type Engine struct {
	bedrock *bedrockruntime.Client
	modelID string
}

func NewEngine(cfg aws.Config) *Engine {
	return &Engine{
		bedrock: bedrockruntime.NewFromConfig(cfg),
		modelID: "apac.amazon.nova-lite-v1:0",
	}
}

var mdFenceRe = regexp.MustCompile("(?s)```[a-zA-Z]*\\n?(.*?)```")

func stripMarkdownFence(s string) string {
	s = strings.TrimSpace(s)
	if m := mdFenceRe.FindStringSubmatch(s); m != nil {
		return strings.TrimSpace(m[1])
	}
	return s
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

	var resultString strings.Builder
	for _, block := range outputMemberMessage.Value.Content {
		textBlock, ok := block.(*types.ContentBlockMemberText)
		if !ok {
			continue
		}
		resultString.WriteString(textBlock.Value)
	}
	// 這裡轉換一下 json 結果
	var result Decision
	jsonString := stripMarkdownFence(resultString.String())
	err = json.Unmarshal([]byte(jsonString), &result)
	if err != nil {
		return nil, fmt.Errorf("json unmarshal error : %v", err)
	}
	return &result, nil
}
