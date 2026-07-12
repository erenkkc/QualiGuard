package ai

import (
	"context"
	"strings"

	"github.com/qualiguard/qualiguard/internal/model"
)

type Explainer struct {
	provider Provider
}

func NewExplainer(enableLLM bool) *Explainer {
	if !enableLLM {
		return &Explainer{}
	}
	cfg := LoadConfig()
	return &Explainer{provider: BuildProvider(cfg)}
}

func (e *Explainer) Explain(ctx context.Context, issue model.Issue) model.AIExplanation {
	if e.provider != nil && e.provider.Available() {
		if llm, err := e.provider.ExplainIssue(ctx, issue); err == nil && llm.SummaryTR != "" {
			llm.Source = "llm"
			return llm
		}
	}
	return templateExplain(issue)
}

// ExplainOnDemand answers only the user's question — no auto template fill.
func (e *Explainer) ExplainOnDemand(ctx context.Context, issue model.Issue, codeLine, question string) model.AIExplanation {
	if strings.TrimSpace(codeLine) != "" {
		issue.Snippet = codeLine
	}
	q := strings.TrimSpace(question)
	if q == "" {
		return model.AIExplanation{Source: "hint"}
	}

	if e.provider != nil && e.provider.Available() {
		reply, err := e.provider.Chat(ctx, []ChatMessage{
			{Role: "user", Content: buildIssueQuestion(issue, q)},
		})
		if err == nil && strings.TrimSpace(reply) != "" {
			return model.AIExplanation{
				SummaryTR: polishChatReply(reply),
				Source:    "llm",
			}
		}
		return model.AIExplanation{
			SummaryTR: "Yanıt alınamadı. Biraz bekleyip tekrar deneyin.",
			Source:    "error",
		}
	}

	return model.AIExplanation{
		SummaryTR: "Yapay zeka kapalı. Ollama'yı başlatıp sunucuyu yeniden çalıştırın.",
		Source:    "template",
	}
}

func EnrichIssues(ctx context.Context, explainer *Explainer, issues []model.Issue) {
	if explainer == nil {
		explainer = NewExplainer(false)
	}
	for i := range issues {
		exp := explainer.Explain(ctx, issues[i])
		issues[i].AIExplanation = &exp
	}
}

func templateExplain(issue model.Issue) model.AIExplanation {
	if tpl, ok := templates[issue.RuleKey]; ok {
		return model.AIExplanation{
			SummaryTR: tpl.summary,
			RiskTR:    tpl.risk,
			ExampleTR: tpl.example,
			Source:    "template",
		}
	}
	return model.AIExplanation{
		SummaryTR: issue.Message,
		RiskTR:    "Bu sorun kod kalitesini veya güvenliği etkileyebilir.",
		Source:    "template",
	}
}
