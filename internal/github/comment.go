package github

import (
	"fmt"
	"sort"
	"strings"

	"github.com/qualiguard/qualiguard/internal/model"
)

const CommentMarker = "<!-- qualiguard-pr-comment -->"

type Breakdown struct {
	Total    int
	Blocker  int
	Critical int
	Bugs     int
	Vuln     int
	Smells   int
}

func IssueBreakdown(issues []model.Issue) Breakdown {
	var b Breakdown
	b.Total = len(issues)
	for _, issue := range issues {
		switch issue.Severity {
		case model.SeverityBlocker:
			b.Blocker++
		case model.SeverityCritical:
			b.Critical++
		}
		switch issue.Type {
		case model.TypeBug:
			b.Bugs++
		case model.TypeVulnerability:
			b.Vuln++
		case model.TypeCodeSmell:
			b.Smells++
		}
	}
	return b
}

func BuildPRComment(report *model.Report) string {
	if report == nil {
		return CommentMarker + "\n## QualiGuard\n\nRapor oluşturulamadı.\n"
	}

	b := IssueBreakdown(report.Issues)
	gate := report.Gate
	statusEmoji, statusText := gateStatus(gate)
	critical := b.Blocker + b.Critical + b.Vuln

	var sb strings.Builder
	sb.WriteString(CommentMarker)
	sb.WriteString("\n## QualiGuard — Kod Kalitesi Raporu\n\n")
	sb.WriteString(fmt.Sprintf("**Kalite Kapısı:** %s **%s**\n\n", statusEmoji, statusText))

	sb.WriteString("| Metrik | Değer |\n")
	sb.WriteString("| --- | ---: |\n")
	sb.WriteString(fmt.Sprintf("| Toplam uyarı | **%d** |\n", b.Total))
	sb.WriteString(fmt.Sprintf("| Kritik (engelleyici + kritik + güvenlik) | %d |\n", critical))
	sb.WriteString(fmt.Sprintf("| Stil (kod kokusu) | %d |\n", b.Smells))
	sb.WriteString(fmt.Sprintf("| Taranan dosya | %d |\n", report.Measures.Files))
	if report.Analysis.Incremental {
		base := report.Analysis.BaseRef
		if base == "" {
			base = "main"
		}
		sb.WriteString(fmt.Sprintf("| Tarama modu | Artımlı (`%s`) |\n", base))
	}

	if gate != nil && len(gate.Conditions) > 0 {
		sb.WriteString("\n### Kapı kriterleri\n\n")
		for _, cond := range gate.Conditions {
			mark := "✓"
			if !cond.Passed {
				mark = "✗"
			}
			sb.WriteString(fmt.Sprintf("- %s **%s:** %.0f (limit %.0f)\n", mark, cond.LabelTR, cond.Actual, cond.Threshold))
		}
		if b.Total > 0 {
			sb.WriteString(fmt.Sprintf("- → **Stil uyarısı:** %d — kapıya dahil değil\n", b.Smells))
		}
	}

	if gate != nil && gate.Status == "FAIL" {
		sb.WriteString("\n> **Bu PR kalite kapısından geçemedi.** Kritik sorunları düzeltin.\n")
	} else if b.Total > 0 {
		sb.WriteString("\n> Kapı **geçti** — bulgular stil uyarısı; merge engellenmez.\n")
	}

	top := topIssues(report.Issues, 10)
	if len(top) > 0 {
		sb.WriteString("\n### Öne çıkan bulgular\n\n")
		sb.WriteString("| Dosya | Satır | Kural | Mesaj |\n")
		sb.WriteString("| --- | ---: | --- | --- |\n")
		for _, issue := range top {
			sb.WriteString(fmt.Sprintf("| `%s` | %d | `%s` | %s |\n",
				escapeTable(issue.File),
				issue.Line,
				escapeTable(issue.RuleKey),
				escapeTable(issue.Message),
			))
		}
		if len(report.Issues) > len(top) {
			sb.WriteString(fmt.Sprintf("\n_…ve %d bulgu daha._\n", len(report.Issues)-len(top)))
		}
	}

	sb.WriteString("\n---\n")
	sb.WriteString(fmt.Sprintf("<sub>QualiGuard %s", report.ScannerVersion))
	if report.Project.Key != "" {
		sb.WriteString(fmt.Sprintf(" · proje: %s", report.Project.Key))
	}
	sb.WriteString("</sub>\n")

	return sb.String()
}

func gateStatus(gate *model.GateResult) (string, string) {
	if gate == nil {
		return "⚪", "Bilinmiyor"
	}
	switch gate.Status {
	case "PASS":
		return "✅", gate.StatusTR
	case "WARN":
		return "⚠️", gate.StatusTR
	case "FAIL":
		return "❌", gate.StatusTR
	default:
		return "⚪", gate.StatusTR
	}
}

func topIssues(issues []model.Issue, limit int) []model.Issue {
	if len(issues) == 0 || limit <= 0 {
		return nil
	}
	sorted := append([]model.Issue(nil), issues...)
	sort.Slice(sorted, func(i, j int) bool {
		si := severityRank(sorted[i].Severity)
		sj := severityRank(sorted[j].Severity)
		if si != sj {
			return si < sj
		}
		if sorted[i].File != sorted[j].File {
			return sorted[i].File < sorted[j].File
		}
		return sorted[i].Line < sorted[j].Line
	})
	if len(sorted) > limit {
		sorted = sorted[:limit]
	}
	return sorted
}

func severityRank(sev model.Severity) int {
	switch sev {
	case model.SeverityBlocker:
		return 0
	case model.SeverityCritical:
		return 1
	case model.SeverityMajor:
		return 2
	case model.SeverityMinor:
		return 3
	default:
		return 4
	}
}

func escapeTable(value string) string {
	value = strings.ReplaceAll(value, "|", "\\|")
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "\r", "")
	return value
}
