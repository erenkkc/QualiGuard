package reporter

import (
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/qualiguard/qualiguard/internal/model"
)

func RenderHTMLReport(projectName, projectKey string, gate *model.GateResult, issues []model.Issue, measures map[string]float64) string {
	gateText := "—"
	gateClass := "pass"
	if gate != nil {
		gateText = gate.StatusTR
		if gate.StatusTR == "" {
			gateText = gate.Status
		}
		gateClass = strings.ToLower(gate.Status)
	}

	var rows strings.Builder
	for _, issue := range issues {
		if issue.Status == "CLOSED" {
			continue
		}
		rows.WriteString(fmt.Sprintf(
			`<tr><td>%s</td><td>%s</td><td>%s</td><td>%d</td><td>%s</td></tr>`,
			html.EscapeString(string(issue.Severity)),
			html.EscapeString(string(issue.Type)),
			html.EscapeString(issue.File),
			issue.Line,
			html.EscapeString(issue.Message),
		))
	}

	measHTML := ""
	if len(measures) > 0 {
		measHTML = "<ul class=\"measures\">"
		for _, key := range []string{"files", "ncloc", "complexity", "bugs", "vulnerabilities", "code_smells"} {
			if v, ok := measures[key]; ok && v > 0 {
				measHTML += fmt.Sprintf("<li><strong>%s:</strong> %.0f</li>", html.EscapeString(key), v)
			}
		}
		measHTML += "</ul>"
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="tr">
<head>
  <meta charset="UTF-8">
  <title>QualiGuard Rapor — %s</title>
  <style>
    body { font-family: Segoe UI, sans-serif; margin: 32px; color: #111; }
    h1 { margin-bottom: 4px; }
    .meta { color: #555; margin-bottom: 20px; }
    .toolbar { margin-bottom: 20px; }
    .toolbar button { background: #1a56db; color: #fff; border: 0; border-radius: 8px; padding: 10px 18px; font-size: 14px; cursor: pointer; }
    .toolbar button:hover { background: #1446b8; }
    .gate { display: inline-block; padding: 8px 16px; border-radius: 999px; font-weight: 700; margin-bottom: 20px; }
    .gate.pass { background: #e8f8ee; color: #137333; }
    .gate.warn { background: #fff7e6; color: #9a6700; }
    .gate.fail { background: #fde8e8; color: #b42318; }
    table { width: 100%%; border-collapse: collapse; font-size: 14px; }
    th, td { border: 1px solid #ddd; padding: 8px 10px; text-align: left; vertical-align: top; }
    th { background: #f5f5f5; }
    .measures { list-style: none; padding: 0; display: flex; gap: 16px; flex-wrap: wrap; }
    .footer { margin-top: 24px; color: #777; font-size: 12px; }
    @media print {
      .no-print { display: none !important; }
      body { margin: 16px; }
    }
  </style>
</head>
<body>
  <div class="toolbar no-print">
    <button type="button" onclick="window.print()">PDF olarak kaydet / Yazdır</button>
    <span style="margin-left:12px;color:#666;font-size:13px">Yazdır penceresinde hedef: PDF olarak kaydet</span>
  </div>
  <h1>%s</h1>
  <div class="meta">Proje kodu: <code>%s</code></div>
  <div class="gate %s">Kalite kapısı: %s</div>
  %s
  <h2>Açık sorunlar</h2>
  <table>
    <thead><tr><th>Önem</th><th>Tür</th><th>Dosya</th><th>Satır</th><th>Mesaj</th></tr></thead>
    <tbody>%s</tbody>
  </table>
  <div class="footer">QualiGuard · %s</div>
</body>
</html>`,
		html.EscapeString(projectName),
		html.EscapeString(projectName),
		html.EscapeString(projectKey),
		gateClass,
		html.EscapeString(gateText),
		measHTML,
		rows.String(),
		html.EscapeString(time.Now().Format("02.01.2006 15:04")),
	)
}
