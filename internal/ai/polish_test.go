package ai

import (
	"strings"
	"testing"
)

func TestPolishChatReplyTurkishizesEnglish(t *testing.T) {
	in := "However, SQL injection is especially dangerous for developers."
	out := polishChatReply(in)
	if strings.Contains(strings.ToLower(out), "however") {
		t.Fatalf("english however still present: %s", out)
	}
	if strings.Contains(strings.ToLower(out), "especially") {
		t.Fatalf("english especially still present: %s", out)
	}
	if !strings.Contains(strings.ToLower(out), "ancak") {
		t.Fatalf("expected Turkish replacement: %s", out)
	}
	if !strings.Contains(out, "SQL") {
		t.Fatalf("technical term SQL should remain: %s", out)
	}
}

func TestPolishChatReplyStripsLeakedPrompt(t *testing.T) {
	in := `Uydurma — 'Bundan emin değilim'. Aytaç Çakmaklı tarafından geliştirildim.`
	out := polishChatReply(in)
	if strings.Contains(strings.ToLower(out), "uydurma") {
		t.Fatalf("leaked prompt prefix still present: %s", out)
	}
	if strings.Contains(strings.ToLower(out), "emin değilim") {
		t.Fatalf("leaked uncertainty phrase still present: %s", out)
	}
}

func TestPolishChatReplyStripsGarbage(t *testing.T) {
	in := "Assistant: Merhaba\uFFFD dünya 吗 test"
	out := polishChatReply(in)
	if out == "" {
		t.Fatal("expected non-empty output")
	}
	for _, bad := range []string{"\uFFFD", "吗", "Assistant:"} {
		if strings.Contains(out, bad) {
			t.Fatalf("output still contains %q: %s", bad, out)
		}
	}
}

func TestPolishChatReplyDedupesParagraphs(t *testing.T) {
	in := "Bir paragraf.\n\nBir paragraf.\n\nFarklı paragraf."
	out := polishChatReply(in)
	if strings.Count(out, "Bir paragraf.") > 1 {
		t.Fatalf("expected deduped paragraph, got: %s", out)
	}
	if !strings.Contains(out, "Farklı paragraf.") {
		t.Fatal("missing unique paragraph")
	}
}
