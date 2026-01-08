package llm

import (
	"context"
	"fmt"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"

	"github.com/emirozbir/micro-sre/internal/config"
)

type OpenAIClient struct {
	client      *openai.Client
	model       string
	maxTokens   int
	temperature float32
}

func NewOpenAIClient(cfg *config.Config) (*OpenAIClient, error) {
	if cfg.LLM.APIKey == "" {
		return nil, fmt.Errorf("openai API key not configured")
	}

	client := openai.NewClient(
		option.WithAPIKey(cfg.LLM.APIKey),
	)

	return &OpenAIClient{
		client:      &client,
		model:       cfg.LLM.Model,
		maxTokens:   cfg.LLM.MaxTokens,
		temperature: cfg.LLM.Temperature,
	}, nil
}

func (o *OpenAIClient) Analyze(ctx context.Context, prompt string) (string, error) {
	completion, err := o.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: openai.ChatModel(o.model),
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(prompt),
		},
		MaxTokens:   openai.Int(int64(o.maxTokens)),
		Temperature: openai.Float(float64(o.temperature)),
	})

	if err != nil {
		return "", fmt.Errorf("openai API call failed: %w", err)
	}

	if len(completion.Choices) == 0 {
		return "", fmt.Errorf("empty response from OpenAI")
	}

	return completion.Choices[0].Message.Content, nil
}
