package linter

import "testing"

func TestParseESLintOutput(t *testing.T) {
	raw := `[{
		"filePath": "C:\\tmp\\script.js",
		"messages": [
			{"ruleId":"no-var","severity":1,"message":"Unexpected var, use let or const instead.","line":21,"column":3},
			{"ruleId":"no-eval","severity":2,"message":"eval can be harmful.","line":10,"column":5}
		]
	}]`

	issues, err := parseESLintOutput([]byte(raw), "script.js")
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(issues))
	}
	if issues[0].RuleKey != "eslint:no-var" {
		t.Fatalf("unexpected rule key: %s", issues[0].RuleKey)
	}
	if issues[1].Severity != "CRITICAL" {
		t.Fatalf("expected critical for no-eval, got %s", issues[1].Severity)
	}
}
