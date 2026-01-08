package llm

import (
	"context"
	"fmt"

	"github.com/emirozbir/micro-sre/internal/config"
)

type Client interface {
	Analyze(ctx context.Context, prompt string) (string, error)
}

func NewClient(cfg *config.Config) (Client, error) {
	switch cfg.LLM.Provider {
	case "anthropic":
		return NewAnthropicClient(cfg)
	case "openai":
		return NewOpenAIClient(cfg)
	default:
		return nil, fmt.Errorf("unknown LLM provider: %s", cfg.LLM.Provider)
	}
}
