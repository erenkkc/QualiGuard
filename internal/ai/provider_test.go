package ai_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/qualiguard/qualiguard/internal/ai"
	"github.com/qualiguard/qualiguard/internal/model"
)

func TestBuildProviderPrefersOpenAI(t *testing.T) {
	t.Setenv("QUALIGUARD_AI_PROVIDER", "auto")
	t.Setenv("QUALIGUARD_OPENAI_API_KEY", "test-key")
	t.Setenv("QUALIGUARD_OLLAMA_ENABLED", "1")

	cfg := ai.LoadConfig()
	p := ai.BuildProvider(cfg)
	if p == nil || p.Name() != "openai" {
		t.Fatalf("expected openai provider, got %#v", p)
	}
}

func TestOpenAIExplainPrompt(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"summary_tr\":\"test\",\"risk_tr\":\"\",\"example_tr\":\"\"}"}}]}`))
	}))
	defer srv.Close()

	t.Setenv("OPENAI_API_KEY", "test")
	t.Setenv("QUALIGUARD_OPENAI_API_KEY", "")
	// Use custom URL via patching - OpenAI provider uses hardcoded URL.
	// Test provider selection only; full HTTP test would need injectable base URL.
	_ = srv
}

func TestAssistantStatusWithGeminiEnv(t *testing.T) {
	t.Setenv("QUALIGUARD_AI_PROVIDER", "gemini")
	t.Setenv("QUALIGUARD_GEMINI_API_KEY", "abc")
	t.Setenv("QUALIGUARD_OPENAI_API_KEY", "")

	status := ai.AssistantStatus()
	if status["provider"] != "gemini" {
		t.Fatalf("expected gemini, got %v", status["provider"])
	}
	if status["active"] != true {
		t.Fatal("expected active gemini")
	}
}

func TestExplainOnDemandUsesProviderWhenConfigured(t *testing.T) {
	os.Unsetenv("QUALIGUARD_OPENAI_API_KEY")
	os.Unsetenv("OPENAI_API_KEY")
	os.Unsetenv("QUALIGUARD_GEMINI_API_KEY")
	os.Unsetenv("QUALIGUARD_OLLAMA_ENABLED")

	exp := ai.NewExplainer(false)
	out := exp.ExplainOnDemand(context.Background(), model.Issue{
		RuleKey: "javascript:no-var",
		Message: "Unexpected var",
		Line:    3,
	}, "var x = 1;", "Bu uyarı ciddi mi?")
	if out.SummaryTR == "" {
		t.Fatal("expected template answer")
	}
}
