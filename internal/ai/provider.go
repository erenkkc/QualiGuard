package ai

import (
	"context"
	"strings"

	"github.com/qualiguard/qualiguard/internal/model"
)

type Provider interface {
	Available() bool
	Name() string
	ModelName() string
	ExplainIssue(ctx context.Context, issue model.Issue) (model.AIExplanation, error)
	ExplainPrompt(ctx context.Context, prompt string) (model.AIExplanation, error)
	Chat(ctx context.Context, messages []ChatMessage) (string, error)
}

func BuildProvider(cfg Config) Provider {
	if !cfg.Enabled {
		return nil
	}
	choice := strings.ToLower(strings.TrimSpace(cfg.Provider))
	switch choice {
	case "openai":
		return NewOpenAIProvider()
	case "gemini":
		return NewGeminiProvider()
	case "ollama":
		return NewOllamaProvider()
	default:
		for _, p := range []Provider{
			NewOpenAIProvider(),
			NewGeminiProvider(),
			NewOllamaProvider(),
		} {
			if p != nil && p.Available() {
				return p
			}
		}
		// Ollama sonra açılabilir — yeniden başlatma zorunlu olmasın
		return NewOllamaProvider()
	}
}

func issuePrompt(issue model.Issue) string {
	return buildExplainPrompt(issue)
}

func AssistantStatus() map[string]any {
	cfg := LoadConfig()
	provider := BuildProvider(cfg)
	active := cfg.Enabled && provider != nil && provider.Available()
	out := map[string]any{
		"enabled": cfg.Enabled,
		"active":  active,
		"mode":    modeLabel(active),
		"choice":  cfg.Provider,
	}
	if provider != nil {
		out["provider"] = provider.Name()
		out["model"] = provider.ModelName()
	}
	return out
}

func modeLabel(llmActive bool) string {
	if llmActive {
		return "detailed"
	}
	return "template"
}
