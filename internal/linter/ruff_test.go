package linter

import "testing"

func TestParseRuffOutput(t *testing.T) {
	raw := `[
		{
			"code": "F401",
			"message": "os imported but unused",
			"filename": "app.py",
			"location": {"row": 1, "column": 8}
		},
		{
			"code": "S105",
			"message": "Possible hardcoded password",
			"filename": "app.py",
			"location": {"row": 3, "column": 1}
		}
	]`

	issues, err := parseRuffOutput([]byte(raw), "app.py")
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(issues))
	}
	if issues[0].RuleKey != "ruff:F401" {
		t.Fatalf("unexpected rule key: %s", issues[0].RuleKey)
	}
	if issues[1].Severity != "CRITICAL" {
		t.Fatalf("expected critical for S105, got %s", issues[1].Severity)
	}
}

func TestMapRuffType(t *testing.T) {
	if mapRuffType("B006") != "BUG" {
		t.Fatal("expected bug type for B006")
	}
	if mapRuffType("F401") != "CODE_SMELL" {
		t.Fatal("expected code smell for F401")
	}
}
