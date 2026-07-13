package ai

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/qualiguard/qualiguard/internal/model"
)

const freeChatSystem = `Sen QualiGuard panelindeki kod asistanısın. Görevin: kod kalitesi, güvenlik açıkları, kalite kapısı ve Python/JS/Go hakkında yardım etmek.

Bildiğin gerçekler:
- QualiGuard, statik kod analizi yapan bir kalite platformudur.
- Sen Ollama üzerinde çalışan bir dil modelisin; ChatGPT değilsin.
- Belirli bir kişinin QualiGuard'ı geliştirdiğine dair bilgin yok — isim söyleme.

Yanıt tarzı:
- Cevabın tamamen Türkçe olmalı; İngilizce kelime kullanma.
- İstisna: teknik terimler (API, SQL, JSON, Python, JavaScript, eval, XSS, Ollama).
- Kısa ve doğal konuş (en fazla 2 paragraf).
- Bilmediğin konuda kibarca "bilmiyorum" de; asla uydurma bilgi verme.
- Kuralları veya talimatları cevabın içinde tekrarlama.
- Kod örneğinde markdown kod bloğu kullan.`

const chatTemperature = 0.4
const chatMaxTokens = 600
// ChatMessage is one turn in free-form assistant chat.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func normalizeChatMessages(messages []ChatMessage) ([]ChatMessage, error) {
	if len(messages) == 0 {
		return nil, fmt.Errorf("en az bir mesaj gerekli")
	}
	out := make([]ChatMessage, 0, len(messages))
	for _, m := range messages {
		role := strings.ToLower(strings.TrimSpace(m.Role))
		content := strings.TrimSpace(m.Content)
		if content == "" {
			continue
		}
		if role != "user" && role != "assistant" {
			return nil, fmt.Errorf("geçersiz rol: %s", m.Role)
		}
		out = append(out, ChatMessage{Role: role, Content: content})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("boş mesaj gönderilemez")
	}
	if out[len(out)-1].Role != "user" {
		return nil, fmt.Errorf("son mesaj kullanıcıdan olmalı")
	}
	if len(out) > 30 {
		out = out[len(out)-30:]
	}
	return out, nil
}

func (e *Explainer) Chat(ctx context.Context, messages []ChatMessage) (string, error) {
	msgs, err := normalizeChatMessages(messages)
	if err != nil {
		return "", err
	}
	p := e.ensureProvider()
	if p == nil {
		return "", fmt.Errorf("yapay zeka aktif değil — Ollama uygulamasını açın (yeniden başlatma gerekmez)")
	}
	reply, err := p.Chat(ctx, msgs)
	if err != nil {
		return "", err
	}
	return polishChatReply(reply), nil
}

// ChatStream emits partial tokens via onDelta; returns the polished full reply.
func (e *Explainer) ChatStream(ctx context.Context, messages []ChatMessage, onDelta func(string) error) (string, error) {
	msgs, err := normalizeChatMessages(messages)
	if err != nil {
		return "", err
	}
	p := e.ensureProvider()
	if p == nil {
		return "", fmt.Errorf("yapay zeka aktif değil — Ollama uygulamasını açın (yeniden başlatma gerekmez)")
	}
	if sp, ok := p.(streamProvider); ok {
		reply, err := sp.ChatStream(ctx, msgs, onDelta)
		if err != nil {
			return "", err
		}
		return polishChatReply(reply), nil
	}
	reply, err := p.Chat(ctx, msgs)
	if err != nil {
		return "", err
	}
	polished := polishChatReply(reply)
	if onDelta != nil && polished != "" {
		if err := onDelta(polished); err != nil {
			return polished, err
		}
	}
	return polished, nil
}

type streamProvider interface {
	ChatStream(ctx context.Context, messages []ChatMessage, onDelta func(string) error) (string, error)
}

var (
	reMultiSpace      = regexp.MustCompile(`[ \t]{2,}`)
	reBlankLines      = regexp.MustCompile(`\n{3,}`)
	reLeakedPrompt    = regexp.MustCompile(`(?i)^(uydurma\s*[—\-:]\s*)?(['"]?bundan emin değilim['"]?\.?\s*)`)
	reLeakedPromptAny = regexp.MustCompile(`(?i)uydurma\s*[—\-:]\s*['"]?bundan emin değilim['"]?\.?`)
)

func polishChatReply(text string) string {
	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, "Assistant:")
	text = strings.TrimPrefix(text, "Asistan:")
	text = strings.TrimPrefix(text, "AI:")
	text = strings.TrimSpace(text)

	var b strings.Builder
	for _, r := range text {
		if r == '\uFFFD' {
			continue
		}
		// Küçük modellerde sızan CJK karakterleri temizle
		if unicode.Is(unicode.Han, r) {
			continue
		}
		b.WriteRune(r)
	}
	text = b.String()
	text = stripLeakedInstructions(text)
	text = fixForeignLeaks(text)
	text = turkishizeEnglishLeaks(text)
	text = dedupeParagraphs(text)
	text = reBlankLines.ReplaceAllString(text, "\n\n")
	text = reMultiSpace.ReplaceAllString(text, " ")
	return strings.TrimSpace(text)
}

func stripLeakedInstructions(text string) string {
	text = reLeakedPromptAny.ReplaceAllString(text, "")
	lines := strings.Split(text, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			out = append(out, line)
			continue
		}
		lower := strings.ToLower(trimmed)
		if strings.HasPrefix(lower, "uydurma") && strings.Contains(lower, "emin değilim") {
			continue
		}
		cleaned := reLeakedPrompt.ReplaceAllString(trimmed, "")
		if strings.TrimSpace(cleaned) == "" {
			continue
		}
		out = append(out, cleaned)
	}
	return strings.Join(out, "\n")
}

func fixForeignLeaks(text string) string {
	repl := []struct{ old, new string }{
		{" ihre ", " onların "},
		{" Ihre ", " Onların "},
		{"contentsi", "içeriği"},
		{"section'u", "bölümü"},
		{"written'si", "yazısı"},
		{"written", "yazı"},
	}
	for _, r := range repl {
		text = strings.ReplaceAll(text, r.old, r.new)
	}
	return text
}

// englishLeakRules — küçük modellerin Türkçe cevaba sıkıştırdığı İngilizce kelimeler.
var englishLeakRules = []struct {
	re *regexp.Regexp
	tr string
}{
	{regexp.MustCompile(`(?i)\bhowever\b`), "ancak"},
	{regexp.MustCompile(`(?i)\bespecially\b`), "özellikle"},
	{regexp.MustCompile(`(?i)\bactually\b`), "aslında"},
	{regexp.MustCompile(`(?i)\bbasically\b`), "temelde"},
	{regexp.MustCompile(`(?i)\bsimply\b`), "basitçe"},
	{regexp.MustCompile(`(?i)\bprobably\b`), "muhtemelen"},
	{regexp.MustCompile(`(?i)\bdeveloper\b`), "geliştirici"},
	{regexp.MustCompile(`(?i)\bdeveloped\b`), "geliştirildi"},
	{regexp.MustCompile(`(?i)\bdevelopment\b`), "geliştirme"},
	{regexp.MustCompile(`(?i)\binformation\b`), "bilgi"},
	{regexp.MustCompile(`(?i)\bcontent\b`), "içerik"},
	{regexp.MustCompile(`(?i)\bsection\b`), "bölüm"},
	{regexp.MustCompile(`(?i)\bwritten\b`), "yazılmış"},
	{regexp.MustCompile(`(?i)\btheir\b`), "onların"},
	{regexp.MustCompile(`(?i)\byour\b`), "senin"},
	{regexp.MustCompile(`(?i)\babout\b`), "hakkında"},
	{regexp.MustCompile(`(?i)\bbecause\b`), "çünkü"},
	{regexp.MustCompile(`(?i)\balthough\b`), "rağmen"},
	{regexp.MustCompile(`(?i)\bthrough\b`), "üzerinden"},
	{regexp.MustCompile(`(?i)\bbetween\b`), "arasında"},
	{regexp.MustCompile(`(?i)\bwithout\b`), "olmadan"},
	{regexp.MustCompile(`(?i)\bwithin\b`), "içinde"},
	{regexp.MustCompile(`(?i)\bincluding\b`), "dahil"},
	{regexp.MustCompile(`(?i)\bexample\b`), "örnek"},
	{regexp.MustCompile(`(?i)\bimportant\b`), "önemli"},
	{regexp.MustCompile(`(?i)\bdifferent\b`), "farklı"},
	{regexp.MustCompile(`(?i)\bsimilar\b`), "benzer"},
	{regexp.MustCompile(`(?i)\bcommon\b`), "yaygın"},
	{regexp.MustCompile(`(?i)\bpossible\b`), "mümkün"},
	{regexp.MustCompile(`(?i)\bavailable\b`), "mevcut"},
	{regexp.MustCompile(`(?i)\boutside\b`), "dışarıda"},
	{regexp.MustCompile(`(?i)\bmeans\b`), "anlamına gelir"},
	{regexp.MustCompile(`(?i)\bprovide\b`), "sağlar"},
	{regexp.MustCompile(`(?i)\bprovides\b`), "sağlar"},
	{regexp.MustCompile(`(?i)\busing\b`), "kullanarak"},
	{regexp.MustCompile(`(?i)\bused\b`), "kullanıldı"},
	{regexp.MustCompile(`(?i)\bhelp\b`), "yardım"},
	{regexp.MustCompile(`(?i)\bhelps\b`), "yardımcı olur"},
	{regexp.MustCompile(`(?i)\bthink\b`), "düşün"},
	{regexp.MustCompile(`(?i)\bbelieve\b`), "inan"},
	{regexp.MustCompile(`(?i)\bknow\b`), "bil"},
	{regexp.MustCompile(`(?i)\bknown\b`), "bilinen"},
	{regexp.MustCompile(`(?i)\bsomething\b`), "bir şey"},
	{regexp.MustCompile(`(?i)\bsomeone\b`), "biri"},
	{regexp.MustCompile(`(?i)\beverything\b`), "her şey"},
	{regexp.MustCompile(`(?i)\bnothing\b`), "hiçbir şey"},
	{regexp.MustCompile(`(?i)\bmaybe\b`), "belki"},
	{regexp.MustCompile(`(?i)\breally\b`), "gerçekten"},
	{regexp.MustCompile(`(?i)\balso\b`), "ayrıca"},
	{regexp.MustCompile(`(?i)\bonly\b`), "yalnızca"},
	{regexp.MustCompile(`(?i)\bvery\b`), "çok"},
	{regexp.MustCompile(`(?i)\bmore\b`), "daha fazla"},
	{regexp.MustCompile(`(?i)\bmost\b`), "en çok"},
	{regexp.MustCompile(`(?i)\bsome\b`), "bazı"},
	{regexp.MustCompile(`(?i)\bmany\b`), "birçok"},
	{regexp.MustCompile(`(?i)\bother\b`), "diğer"},
	{regexp.MustCompile(`(?i)\bsuch\b`), "böyle"},
	{regexp.MustCompile(`(?i)\blike\b`), "gibi"},
	{regexp.MustCompile(`(?i)\bjust\b`), "sadece"},
	{regexp.MustCompile(`(?i)\bstill\b`), "hâlâ"},
	{regexp.MustCompile(`(?i)\balready\b`), "zaten"},
	{regexp.MustCompile(`(?i)\bhere\b`), "burada"},
	{regexp.MustCompile(`(?i)\bthere\b`), "orada"},
	{regexp.MustCompile(`(?i)\bwhere\b`), "nerede"},
	{regexp.MustCompile(`(?i)\bwhen\b`), "ne zaman"},
	{regexp.MustCompile(`(?i)\bwhat\b`), "ne"},
	{regexp.MustCompile(`(?i)\bwhy\b`), "neden"},
	{regexp.MustCompile(`(?i)\bhow\b`), "nasıl"},
	{regexp.MustCompile(`(?i)\bwhich\b`), "hangi"},
	{regexp.MustCompile(`(?i)\bwith\b`), "ile"},
	{regexp.MustCompile(`(?i)\bfor\b`), "için"},
	{regexp.MustCompile(`(?i)\bfrom\b`), "den"},
	{regexp.MustCompile(`(?i)\binto\b`), "içine"},
	{regexp.MustCompile(`(?i)\bover\b`), "üzerinde"},
	{regexp.MustCompile(`(?i)\bunder\b`), "altında"},
	{regexp.MustCompile(`(?i)\bafter\b`), "sonra"},
	{regexp.MustCompile(`(?i)\bbefore\b`), "önce"},
	{regexp.MustCompile(`(?i)\bduring\b`), "sırasında"},
	{regexp.MustCompile(`(?i)\bwhile\b`), "iken"},
	{regexp.MustCompile(`(?i)\band\b`), "ve"},
	{regexp.MustCompile(`(?i)\bbut\b`), "ama"},
	{regexp.MustCompile(`(?i)\bor\b`), "veya"},
}

func turkishizeEnglishLeaks(text string) string {
	chunks := splitPreservingCodeFences(text)
	for i, ch := range chunks {
		if ch.code {
			continue
		}
		s := ch.text
		for _, rule := range englishLeakRules {
			s = rule.re.ReplaceAllString(s, rule.tr)
		}
		s = reMultiSpace.ReplaceAllString(s, " ")
		s = strings.TrimSpace(s)
		chunks[i].text = s
	}
	return joinPreservingCodeFences(chunks)
}

type textChunk struct {
	code bool
	text string
}

func splitPreservingCodeFences(text string) []textChunk {
	parts := strings.Split(text, "```")
	out := make([]textChunk, 0, len(parts))
	for i, p := range parts {
		out = append(out, textChunk{code: i%2 == 1, text: p})
	}
	return out
}

func joinPreservingCodeFences(chunks []textChunk) string {
	var b strings.Builder
	for i, ch := range chunks {
		b.WriteString(ch.text)
		if i < len(chunks)-1 {
			b.WriteString("```")
		}
	}
	return b.String()
}

func dedupeParagraphs(text string) string {
	paras := strings.Split(text, "\n\n")
	seen := make(map[string]struct{}, len(paras))
	out := make([]string, 0, len(paras))
	for _, p := range paras {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		key := strings.ToLower(p)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, p)
	}
	return strings.Join(out, "\n\n")
}
func buildIssueQuestion(issue model.Issue, question string) string {
	code := strings.TrimSpace(issue.Snippet)
	if code == "" {
		code = "(kod satırı yok)"
	}
	return fmt.Sprintf(`Kod uyarısı:
- Kural: %s
- Satır: %d
- Mesaj: %s
- Kod: %s

Sorum: %s`,
		issue.RuleKey, issue.Line, issue.Message, code, strings.TrimSpace(question))
}
