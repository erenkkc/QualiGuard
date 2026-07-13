package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)
func chatCompletion(ctx context.Context, url, apiKey, modelName, prompt string) (string, error) {
	return chatCompletionMessages(ctx, url, apiKey, modelName,
		"Sen QualiGuard kod asistanısın. Türkçe yanıt ver. İstenen JSON formatına uy.",
		[]ChatMessage{{Role: "user", Content: prompt}},
		0.2,
	)
}

func chatCompletionMessages(ctx context.Context, url, apiKey, modelName, system string, messages []ChatMessage, temperature float64) (string, error) {
	return chatCompletionAdvanced(ctx, url, apiKey, modelName, system, messages, temperature, 0)
}

func chatCompletionAdvanced(ctx context.Context, url, apiKey, modelName, system string, messages []ChatMessage, temperature float64, maxTokens int) (string, error) {
	llmMsgs := make([]map[string]string, 0, len(messages)+1)
	if strings.TrimSpace(system) != "" {
		llmMsgs = append(llmMsgs, map[string]string{"role": "system", "content": system})
	}
	for _, m := range messages {
		llmMsgs = append(llmMsgs, map[string]string{"role": m.Role, "content": m.Content})
	}
	payload := map[string]any{
		"model":       modelName,
		"messages":    llmMsgs,
		"temperature": temperature,
	}
	if maxTokens > 0 {
		payload["max_tokens"] = maxTokens
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	client := &http.Client{Timeout: 180 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("chat completion status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil || len(parsed.Choices) == 0 {
		return "", fmt.Errorf("invalid chat completion response")
	}
	return strings.TrimSpace(parsed.Choices[0].Message.Content), nil
}

func chatCompletionStream(ctx context.Context, url, apiKey, modelName, system string, messages []ChatMessage, temperature float64, maxTokens int, onDelta func(string) error) (string, error) {
	llmMsgs := make([]map[string]string, 0, len(messages)+1)
	if strings.TrimSpace(system) != "" {
		llmMsgs = append(llmMsgs, map[string]string{"role": "system", "content": system})
	}
	for _, m := range messages {
		llmMsgs = append(llmMsgs, map[string]string{"role": m.Role, "content": m.Content})
	}
	payload := map[string]any{
		"model":       modelName,
		"messages":    llmMsgs,
		"temperature": temperature,
		"stream":      true,
	}
	if maxTokens > 0 {
		payload["max_tokens"] = maxTokens
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	client := &http.Client{Timeout: 180 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("chat completion status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var full strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			break
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil || len(chunk.Choices) == 0 {
			continue
		}
		delta := chunk.Choices[0].Delta.Content
		if delta == "" {
			delta = chunk.Choices[0].Message.Content
		}
		if delta == "" {
			continue
		}
		full.WriteString(delta)
		if onDelta != nil {
			if err := onDelta(delta); err != nil {
				return full.String(), err
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return full.String(), err
	}
	return strings.TrimSpace(full.String()), nil
}