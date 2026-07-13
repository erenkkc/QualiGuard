package ai

import (
	"context"
	"encoding/json"
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
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
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
	if resp.StatusCode >= 300 {
		return false
	}
	var payload struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return true // tags ok, model list optional
	}
	want := strings.TrimSpace(p.model)
	if want == "" {
		return len(payload.Models) > 0
	}
	wantBase := strings.Split(want, ":")[0]
	for _, m := range payload.Models {
		name := strings.TrimSpace(m.Name)
		if name == want || strings.HasPrefix(name, wantBase+":") || name == wantBase {
			return true
		}
	}
	return false
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