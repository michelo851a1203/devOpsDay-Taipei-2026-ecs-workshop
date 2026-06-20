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

func ParseResponse(output []byte) (*Decision, error) {
	var res NovaResponse
	err := json.Unmarshal(output, &res)
	if err != nil {
		return nil, fmt.Errorf("[parseResponse] failed to unmarshal response: %w", err)
	}
	contents := res.Output.Message.Content
	if len(contents) == 0 {
		return nil, fmt.Errorf("no content in response")
	}
	resultContent := stripMarkdownFence(contents[0].Text)

	var result Decision
	err = json.Unmarshal([]byte(resultContent), &result)
	if err != nil {
		return nil, fmt.Errorf("[parseResponse] failed to unmarshal response: %w", err)
	}

	return &result, nil
}

func (e *Engine) Evaluate(ctx context.Context, prompt string) (*Decision, error) {
	request := NovaRequest{
		Messages: []NovaRequestMessage{
			{Role: "user", Content: []NovaRequestMessageContent{{Text: prompt}}},
		},
		InferenceConfig: NovaRequestInferenceConfig{
			MaxNewTokens: 1024,
		},
	}

	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("json marshal error : %w", err)
	}

	res, err := e.bedrock.InvokeModel(ctx, &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(e.modelID),
		ContentType: aws.String("application/json"),
		Body:        body,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to invoke model : %w", err)
	}

	return ParseResponse(res.Body)
}
