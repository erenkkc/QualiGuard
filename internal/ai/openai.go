package ai

import (
	"context"
	"strings"

	"github.com/qualiguard/qualiguard/internal/model"
)

type OpenAIProvider struct {
	apiKey string
	model  string
}

func NewOpenAIProvider() *OpenAIProvider {
	apiKey := envFirst("QUALIGUARD_OPENAI_API_KEY", "OPENAI_API_KEY")
	model := envFirst("QUALIGUARD_OPENAI_MODEL", "OPENAI_MODEL")
	if model == "" {
		model = "gpt-4o-mini"
	}
	return &OpenAIProvider{apiKey: apiKey, model: model}
}

func (p *OpenAIProvider) Name() string      { return "openai" }
func (p *OpenAIProvider) ModelName() string { return p.model }
func (p *OpenAIProvider) Available() bool {
	return strings.TrimSpace(p.apiKey) != ""
}

func (p *OpenAIProvider) ExplainIssue(ctx context.Context, issue model.Issue) (model.AIExplanation, error) {
	return p.ExplainPrompt(ctx, issuePrompt(issue))
}

func (p *OpenAIProvider) ExplainPrompt(ctx context.Context, prompt string) (model.AIExplanation, error) {
	text, err := chatCompletion(ctx, "https://api.openai.com/v1/chat/completions", p.apiKey, p.model, prompt)
	if err != nil {
		return model.AIExplanation{}, err
	}
	out, err := parseExplainJSON(text)
	if err != nil {
		return model.AIExplanation{}, err
	}
	out.Source = "llm"
	out.Provider = "openai:" + p.model
	return out, nil
}

func (p *OpenAIProvider) Chat(ctx context.Context, messages []ChatMessage) (string, error) {
	return chatCompletionAdvanced(ctx, "https://api.openai.com/v1/chat/completions", p.apiKey, p.model, freeChatSystem, messages, 0.7, chatMaxTokens)
}

func (p *OpenAIProvider) ChatStream(ctx context.Context, messages []ChatMessage, onDelta func(string) error) (string, error) {
	return chatCompletionStream(ctx, "https://api.openai.com/v1/chat/completions", p.apiKey, p.model, freeChatSystem, messages, 0.7, chatMaxTokens, onDelta)
}