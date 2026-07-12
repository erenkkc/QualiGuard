package reporter

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/qualiguard/qualiguard/internal/model"
)

func WriteJSON(outputPath string, report *model.Report) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal report: %w", err)
	}

	if outputPath == "" || outputPath == "-" {
		_, err = os.Stdout.Write(data)
		_, _ = os.Stdout.Write([]byte("\n"))
		return err
	}

	return os.WriteFile(outputPath, append(data, '\n'), 0o644)
}

func WriteSARIF(outputPath string, report *model.Report) error {
	doc := toSARIF(report)
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal sarif: %w", err)
	}

	if outputPath == "" || outputPath == "-" {
		_, err = os.Stdout.Write(data)
		_, _ = os.Stdout.Write([]byte("\n"))
		return err
	}

	return os.WriteFile(outputPath, append(data, '\n'), 0o644)
}

func toSARIF(report *model.Report) map[string]any {
	runs := []map[string]any{{
		"tool": map[string]any{
			"driver": map[string]any{
				"name":    "QualiGuard",
				"version": report.ScannerVersion,
				"rules":   sarifRules(report.Issues),
			},
		},
		"results": sarifResults(report.Issues),
	}}

	return map[string]any{
		"version": "2.1.0",
		"$schema": "https://json.schemastore.org/sarif-2.1.0.json",
		"runs":    runs,
	}
}

func sarifRules(issues []model.Issue) []map[string]any {
	seen := map[string]struct{}{}
	var rules []map[string]any
	for _, issue := range issues {
		if _, ok := seen[issue.RuleKey]; ok {
			continue
		}
		seen[issue.RuleKey] = struct{}{}
		rules = append(rules, map[string]any{
			"id": issue.RuleKey,
			"name": map[string]any{
				"text": issue.RuleKey,
			},
			"shortDescription": map[string]any{
				"text": issue.Message,
			},
			"defaultConfiguration": map[string]any{
				"level": sarifLevel(issue.Severity),
			},
		})
	}
	return rules
}

func sarifResults(issues []model.Issue) []map[string]any {
	results := make([]map[string]any, 0, len(issues))
	for _, issue := range issues {
		result := map[string]any{
			"ruleId": issue.RuleKey,
			"level":  sarifLevel(issue.Severity),
			"message": map[string]any{
				"text": issue.Message,
			},
			"locations": []map[string]any{{
				"physicalLocation": map[string]any{
					"artifactLocation": map[string]any{
						"uri": issue.File,
					},
					"region": map[string]any{
						"startLine": issue.Line,
					},
				},
			}},
		}
		if issue.Column > 0 {
			result["locations"].([]map[string]any)[0]["physicalLocation"].(map[string]any)["region"].(map[string]any)["startColumn"] = issue.Column
		}
		if issue.Fingerprint != "" {
			result["partialFingerprints"] = map[string]any{
				"primaryLocationLineHash": strings.TrimPrefix(issue.Fingerprint, "sha256:"),
			}
		}
		results = append(results, result)
	}
	return results
}

func sarifLevel(severity model.Severity) string {
	switch severity {
	case model.SeverityBlocker, model.SeverityCritical:
		return "error"
	case model.SeverityMajor:
		return "warning"
	default:
		return "note"
	}
}
