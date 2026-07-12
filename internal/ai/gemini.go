package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/qualiguard/qualiguard/internal/model"
)

type GeminiProvider struct {
	apiKey string
	model  string
}

func NewGeminiProvider() *GeminiProvider {
	apiKey := envFirst("QUALIGUARD_GEMINI_API_KEY", "GEMINI_API_KEY", "GOOGLE_API_KEY")
	model := envFirst("QUALIGUARD_GEMINI_MODEL", "GEMINI_MODEL")
	if model == "" {
		model = "gemini-2.0-flash"
	}
	return &GeminiProvider{apiKey: apiKey, model: model}
}

func (p *GeminiProvider) Name() string      { return "gemini" }
func (p *GeminiProvider) ModelName() string { return p.model }
func (p *GeminiProvider) Available() bool {
	return strings.TrimSpace(p.apiKey) != ""
}

func (p *GeminiProvider) ExplainIssue(ctx context.Context, issue model.Issue) (model.AIExplanation, error) {
	return p.ExplainPrompt(ctx, issuePrompt(issue))
}

func (p *GeminiProvider) ExplainPrompt(ctx context.Context, prompt string) (model.AIExplanation, error) {
	text, err := geminiCompletion(ctx, p.apiKey, p.model, prompt)
	if err != nil {
		return model.AIExplanation{}, err
	}
	out, err := parseExplainJSON(text)
	if err != nil {
		return model.AIExplanation{}, err
	}
	out.Source = "llm"
	out.Provider = "gemini:" + p.model
	return out, nil
}

func geminiCompletion(ctx context.Context, apiKey, modelName, prompt string) (string, error) {
	url := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
		modelName,
		apiKey,
	)
	body, _ := json.Marshal(map[string]any{
		"contents": []map[string]any{
			{"parts": []map[string]string{{"text": prompt}}},
		},
		"generationConfig": map[string]any{
			"temperature": 0.2,
		},
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 45 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("gemini status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var parsed struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil || len(parsed.Candidates) == 0 {
		return "", fmt.Errorf("invalid gemini response")
	}
	parts := parsed.Candidates[0].Content.Parts
	if len(parts) == 0 || strings.TrimSpace(parts[0].Text) == "" {
		return "", fmt.Errorf("empty gemini response")
	}
	return strings.TrimSpace(parts[0].Text), nil
}

func (p *GeminiProvider) Chat(ctx context.Context, messages []ChatMessage) (string, error) {
	return geminiChat(ctx, p.apiKey, p.model, messages)
}

func geminiChat(ctx context.Context, apiKey, modelName string, messages []ChatMessage) (string, error) {
	url := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
		modelName,
		apiKey,
	)
	contents := make([]map[string]any, 0, len(messages))
	for _, m := range messages {
		role := "user"
		if m.Role == "assistant" {
			role = "model"
		}
		contents = append(contents, map[string]any{
			"role":  role,
			"parts": []map[string]string{{"text": m.Content}},
		})
	}
	body, _ := json.Marshal(map[string]any{
		"systemInstruction": map[string]any{
			"parts": []map[string]string{{"text": freeChatSystem}},
		},
		"contents": contents,
		"generationConfig": map[string]any{
			"temperature":     chatTemperature,
			"maxOutputTokens": chatMaxTokens,
		},	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("gemini status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var parsed struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil || len(parsed.Candidates) == 0 {
		return "", fmt.Errorf("invalid gemini response")
	}
	parts := parsed.Candidates[0].Content.Parts
	if len(parts) == 0 || strings.TrimSpace(parts[0].Text) == "" {
		return "", fmt.Errorf("empty gemini response")
	}
	return strings.TrimSpace(parts[0].Text), nil
}
