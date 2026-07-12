package ai

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/qualiguard/qualiguard/internal/model"
)

func buildAskPrompt(issue model.Issue, question string) string {
	code := strings.TrimSpace(issue.Snippet)
	if code == "" {
		code = "(kod satırı yok)"
	}
	q := strings.TrimSpace(question)
	if q == "" {
		q = "Bu uyarı ne anlama geliyor ve nasıl düzeltilir?"
	}
	casualNote := ""
	if isCasualQuestion(q) {
		casualNote = `
NOT: Öğrencinin sorusu selamlaşma veya sohbet (kod ile ilgili değil). Kısa ve nazik yanıt ver.
risk_tr ve example_tr alanlarını boş string "" bırak; sadece summary_tr doldur.`
	}
	return fmt.Sprintf(`Sen QualiGuard kod asistanısın. Öğrenciye sıcak ve anlaşılır Türkçe ile yardım et.

Kural: %s | Satır: %d
Uyarı: %s
Kod:
%s

Öğrencinin sorusu: %s
%s

Yanıt kuralları:
- Önce sorunun cevabını net ver; öğretmen gibi açıkla.
- Soru uyarı/kod ile ilgiliyse risk ve düzeltme önerisi ekle.
- Soru konu dışıysa: kısa nazik yanıt; risk_tr ve example_tr boş kalsın.
- Sadece JSON döndür.

JSON:
{"summary_tr":"doğrudan ve samimi cevap","risk_tr":"neden önemli veya boş","example_tr":"kısa düzeltme örneği veya boş"}`,
		issue.RuleKey, issue.Line, issue.Message, code, q, casualNote)
}

func buildExplainPrompt(issue model.Issue) string {
	return buildAskPrompt(issue, "")
}

func isCasualQuestion(question string) bool {
	q := strings.ToLower(strings.TrimSpace(question))
	if q == "" {
		return false
	}
	casual := []string{
		"nasılsın", "nasilsin", "naber", "merhaba", "selam", "iyi misin",
		"kimsin", "ne yapıyorsun", "hello", "how are you", "hi",
	}
	for _, word := range casual {
		if strings.Contains(q, word) {
			return true
		}
	}
	return false
}

func casualFallbackAnswer() model.AIExplanation {
	return model.AIExplanation{
		SummaryTR: "Yapay zeka şu an yanıt veremiyor. Ollama'nın açık olduğundan emin ol.",
		Source:    "template",
	}
}

func parseExplainJSON(content string) (model.AIExplanation, error) {
	content = trimJSONFence(content)
	var out struct {
		SummaryTR string `json:"summary_tr"`
		RiskTR    string `json:"risk_tr"`
		ExampleTR string `json:"example_tr"`
	}
	if err := json.Unmarshal([]byte(content), &out); err != nil {
		return model.AIExplanation{}, err
	}
	return model.AIExplanation{
		SummaryTR: out.SummaryTR,
		RiskTR:    out.RiskTR,
		ExampleTR: out.ExampleTR,
	}, nil
}

func trimJSONFence(content string) string {
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	return strings.TrimSpace(content)
}
