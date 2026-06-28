package llm

import (
	"context"
	"fmt"
	"net/http"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

type Provider interface {
	Complete(ctx context.Context, prompt string) (string, error)
}

func New(cfg analysis.LLMConfig) (Provider, error) {
	switch cfg.Provider {
	case "cli":
		return &CLIProvider{command: cfg.CLICommand}, nil
	case "api":
		if cfg.APIKeyEnv == "" {
			return nil, fmt.Errorf("llm: api_key_env must be set for provider %q", cfg.Provider)
		}
		return &APIProvider{
			baseURL:   cfg.APIBaseURL,
			apiKeyEnv: cfg.APIKeyEnv,
			model:     cfg.Model,
			client:    http.DefaultClient,
		}, nil
	case "oauth":
		return nil, fmt.Errorf("llm: oauth mode is not yet supported — use provider=\"cli\" with an OAuth-backed agent such as 'claude -p {prompt}'")
	default:
		return nil, fmt.Errorf("llm: unknown provider %q", cfg.Provider)
	}
}
