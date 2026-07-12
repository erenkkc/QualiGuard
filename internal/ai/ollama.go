package ai

import (
	"context"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/qualiguard/qualiguard/internal/model"
)

type OllamaProvider struct {
	baseURL string
	model   string
}

func NewOllamaProvider() *OllamaProvider {
	baseURL := envFirst("QUALIGUARD_OLLAMA_BASE_URL", "OLLAMA_BASE_URL")
	if baseURL == "" {
		baseURL = "http://127.0.0.1:11434/v1"
	}
	model := envFirst("QUALIGUARD_OLLAMA_MODEL", "OLLAMA_MODEL")
	if model == "" {
		model = "llama3.2:latest"
	}
	return &OllamaProvider{baseURL: strings.TrimRight(baseURL, "/"), model: model}
}

func (p *OllamaProvider) Name() string      { return "ollama" }
func (p *OllamaProvider) ModelName() string { return p.model }
func (p *OllamaProvider) Available() bool {
	if strings.ToLower(envFirst("QUALIGUARD_OLLAMA_ENABLED")) == "0" {
		return false
	}
	enabled := envFirst("QUALIGUARD_OLLAMA_ENABLED") == "1" ||
		envFirst("QUALIGUARD_OLLAMA_MODEL") != "" ||
		os.Getenv("QUALIGUARD_OLLAMA_BASE_URL") != "" ||
		os.Getenv("OLLAMA_BASE_URL") != ""
	if !enabled {
		return false
	}
	return p.ping()
}

func (p *OllamaProvider) ping() bool {
	base := strings.TrimSuffix(p.baseURL, "/v1")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/api/tags", nil)
	if err != nil {
		return false
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode < 300
}

func (p *OllamaProvider) ExplainIssue(ctx context.Context, issue model.Issue) (model.AIExplanation, error) {
	return p.ExplainPrompt(ctx, issuePrompt(issue))
}

func (p *OllamaProvider) ExplainPrompt(ctx context.Context, prompt string) (model.AIExplanation, error) {
	text, err := chatCompletion(ctx, p.baseURL+"/chat/completions", "ollama", p.model, prompt)
	if err != nil {
		return model.AIExplanation{}, err
	}
	out, err := parseExplainJSON(text)
	if err != nil {
		return model.AIExplanation{}, err
	}
	out.Source = "llm"
	out.Provider = p.model
	return out, nil
}

func (p *OllamaProvider) Chat(ctx context.Context, messages []ChatMessage) (string, error) {
	return chatCompletionAdvanced(ctx, p.baseURL+"/chat/completions", "ollama", p.model, freeChatSystem, messages, chatTemperature, chatMaxTokens)
}

func (p *OllamaProvider) ChatStream(ctx context.Context, messages []ChatMessage, onDelta func(string) error) (string, error) {
	return chatCompletionStream(ctx, p.baseURL+"/chat/completions", "ollama", p.model, freeChatSystem, messages, chatTemperature, chatMaxTokens, onDelta)
}