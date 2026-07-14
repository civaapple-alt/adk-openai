package main

import (
	"context"
	"fmt"
	"os"

	adkopenai "github.com/civaapple-alt/adk-openai"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"google.golang.org/genai"

	"google.golang.org/adk/v2/model"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	baseURL := os.Getenv("OPENAI_BASE_URL")
	modelName := os.Getenv("OPENAI_MODEL")
	if modelName == "" {
		modelName = "grok-4.5"
	}
	mode, err := adkopenai.ParseAPIMode(os.Getenv("OPENAI_API_MODE"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if apiKey == "" || baseURL == "" {
		fmt.Fprintln(os.Stderr, "set OPENAI_API_KEY and OPENAI_BASE_URL")
		os.Exit(1)
	}

	client := openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL(baseURL),
	)
	llm := adkopenai.New(client, modelName, adkopenai.WithAPIMode(mode))

	req := &model.LLMRequest{
		Contents: []*genai.Content{
			genai.NewContentFromText("Say hello in one short sentence.", genai.RoleUser),
		},
	}

	fmt.Printf("model=%s apiMode=%s\n", llm.Name(), llm.APIMode())
	for resp, err := range llm.GenerateContent(context.Background(), req, false) {
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if resp != nil && resp.Content != nil {
			for _, p := range resp.Content.Parts {
				if p != nil && p.Text != "" {
					fmt.Println(p.Text)
				}
			}
		}
	}
}
