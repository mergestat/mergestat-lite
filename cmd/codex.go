package cmd

import (
	"context"
	_ "embed"
	"fmt"
	"os"

	"github.com/PullRequestInc/go-gpt3"
)

//go:embed codex-prompt-context.sql
var promptPrefix string

// codexToSQL generates SQL from a natural language prompt
func codexToSQL(ctx context.Context, prompt string) (string, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("missing OPENAI_API_KEY environment variable")
	}

	client := gpt3.NewClient(apiKey)
	var temp float32 = 0
	var topP float32 = 1
	var maxTokens = 512
	res, err := client.CompletionWithEngine(ctx, "code-davinci-002", gpt3.CompletionRequest{
		Prompt:           []string{promptPrefix + prompt + "SELECT"},
		Temperature:      &temp,
		TopP:             &topP,
		FrequencyPenalty: 1,
		Stop:             []string{";"},
		MaxTokens:        &maxTokens,
	})
	if err != nil {
		return "", err
	}

	for _, choice := range res.Choices {
		return "SELECT" + choice.Text, nil
	}

	return "", nil
}
