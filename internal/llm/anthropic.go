package llm

import (
	"context"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/emirozbir/micro-sre/internal/config"
)

type AnthropicClient struct {
	client      *anthropic.Client
	model       string
	maxTokens   int
	temperature float32
}

func NewAnthropicClient(cfg *config.Config) (*AnthropicClient, error) {
	if cfg.LLM.APIKey == "" {
		return nil, fmt.Errorf("anthropic API key not configured")
	}

	client := anthropic.NewClient(
		option.WithAPIKey(cfg.LLM.APIKey),
	)

	return &AnthropicClient{
		client:      client,
		model:       cfg.LLM.Model,
		maxTokens:   cfg.LLM.MaxTokens,
		temperature: cfg.LLM.Temperature,
	}, nil
}

func (a *AnthropicClient) Analyze(ctx context.Context, prompt string) (string, error) {
	message, err := a.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.F(a.model),
		MaxTokens: anthropic.Int(int64(a.maxTokens)),
		Messages: anthropic.F([]anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		}),
		Temperature: anthropic.Float(float64(a.temperature)),
	})

	if err != nil {
		return "", fmt.Errorf("anthropic API call failed: %w", err)
	}

	if len(message.Content) == 0 {
		return "", fmt.Errorf("empty response from Anthropic")
	}

	// Extract text from the first content block
	if textBlock, ok := message.Content[0].AsUnion().(anthropic.TextBlock); ok {
		return textBlock.Text, nil
	}

	return "", fmt.Errorf("unexpected response format from Anthropic")
}
