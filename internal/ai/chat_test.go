package ai_test

import (
	"testing"

	"github.com/qualiguard/qualiguard/internal/ai"
)

func TestNormalizeChatMessages(t *testing.T) {
	exp := ai.NewExplainer(false)
	_, err := exp.Chat(t.Context(), nil)
	if err == nil {
		t.Fatal("expected error for empty messages")
	}

	_, err = exp.Chat(t.Context(), []ai.ChatMessage{
		{Role: "assistant", Content: "merhaba"},
	})
	if err == nil {
		t.Fatal("expected error when last message is not user")
	}
}
