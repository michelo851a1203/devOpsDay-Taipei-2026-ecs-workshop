package main

import (
	"context"
	"fmt"
	"log"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/ollama"
)

func main() {
	fmt.Println("你執行了!!等一下下不急")
	// ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	// defer cancel()
	ctx := context.Background()
	// use genkit
	genkitClient := genkit.Init(ctx, genkit.WithPlugins(
		&ollama.Ollama{
			ServerAddress: "http://localhost:11434",
			Timeout:       600, // 10 mins 如果是跑本地的模型會比較久，可以考慮設 timoue 時間比較久，像是我這跑 10 分鐘
		},
	))
	// model := ollama.Model(genkitClient, "gemma4:cloud")
	// model := ollama.Model(genkitClient, "gemma4:31b-mlx") // 這個挺久的，電腦好的人比較適合 -> 這個 timeout 可能要設5分鐘以上比較保險
	model := ollama.Model(genkitClient, "gemma4:e2b-mlx")
	resp, err := genkit.Generate(ctx, genkitClient,
		ai.WithModel(model),
		ai.WithPrompt("你好可以可以說幾句吉祥話嗎?"),
	)
	if err != nil {
		log.Fatalf("start repsonse error : %v\n", err)
	}
	fmt.Println(resp.Text())
}
