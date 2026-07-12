package gate

import "github.com/qualiguard/qualiguard/internal/model"

type Status string

const (
	StatusPass Status = "PASS"
	StatusWarn Status = "WARN"
	StatusFail Status = "FAIL"
)

type Operator string

const (
	OpGT Operator = "GT"
	OpLT Operator = "LT"
)

type Level string

const (
	LevelError Level = "ERROR"
	LevelWarn  Level = "WARN"
)

type Input struct {
	BlockerIssues   int
	CriticalIssues  int
	Bugs            int
	Vulnerabilities int
	CodeSmells      int
}

type Condition struct {
	Metric    string
	LabelTR   string
	Operator  Operator
	Threshold float64
	Level     Level
}

type ConditionResult struct {
	Metric    string  `json:"metric"`
	LabelTR   string  `json:"label_tr"`
	Operator  string  `json:"operator"`
	Threshold float64 `json:"threshold"`
	Actual    float64 `json:"actual"`
	Passed    bool    `json:"passed"`
	Level     string  `json:"level"`
}

type Result struct {
	Name       string            `json:"name"`
	NameTR     string            `json:"name_tr"`
	Status     Status            `json:"status"`
	StatusTR   string            `json:"status_tr"`
	Conditions []ConditionResult `json:"conditions"`
}

func DefaultGate() []Condition {
	return []Condition{
		{
			Metric:    "blocker_issues",
			LabelTR:   "Engelleyici sorun sayısı",
			Operator:  OpGT,
			Threshold: 0,
			Level:     LevelError,
		},
		{
			Metric:    "critical_issues",
			LabelTR:   "Kritik sorun sayısı",
			Operator:  OpGT,
			Threshold: 0,
			Level:     LevelError,
		},
		{
			Metric:    "vulnerabilities",
			LabelTR:   "Güvenlik açığı sayısı",
			Operator:  OpGT,
			Threshold: 0,
			Level:     LevelError,
		},
		{
			Metric:    "bugs",
			LabelTR:   "Hata (bug) sayısı",
			Operator:  OpGT,
			Threshold: 0,
			Level:     LevelWarn,
		},
	}
}

func InputFromIssues(issues []model.Issue) Input {
	var in Input
	for _, issue := range issues {
		switch issue.Severity {
		case model.SeverityBlocker:
			in.BlockerIssues++
		case model.SeverityCritical:
			in.CriticalIssues++
		}
		switch issue.Type {
		case model.TypeBug:
			in.Bugs++
		case model.TypeVulnerability:
			in.Vulnerabilities++
		case model.TypeCodeSmell:
			in.CodeSmells++
		}
	}
	return in
}

func Evaluate(input Input) Result {
	return EvaluateWithConditions("QualiGuard Way", "QualiGuard Yolu", DefaultGate(), input)
}

func EvaluateWithConditions(name, nameTR string, conditions []Condition, input Input) Result {
	results := make([]ConditionResult, 0, len(conditions))
	status := StatusPass

	for _, cond := range conditions {
		actual := metricValue(input, cond.Metric)
		passed := compare(actual, cond.Operator, cond.Threshold)
		results = append(results, ConditionResult{
			Metric:    cond.Metric,
			LabelTR:   cond.LabelTR,
			Operator:  string(cond.Operator),
			Threshold: cond.Threshold,
			Actual:    actual,
			Passed:    passed,
			Level:     string(cond.Level),
		})
		if passed {
			continue
		}
		if cond.Level == LevelError {
			status = StatusFail
		} else if status != StatusFail {
			status = StatusWarn
		}
	}

	return Result{
		Name:       name,
		NameTR:     nameTR,
		Status:     status,
		StatusTR:   statusTR(status),
		Conditions: results,
	}
}

func metricValue(input Input, metric string) float64 {
	switch metric {
	case "blocker_issues":
		return float64(input.BlockerIssues)
	case "critical_issues":
		return float64(input.CriticalIssues)
	case "bugs":
		return float64(input.Bugs)
	case "vulnerabilities":
		return float64(input.Vulnerabilities)
	case "code_smells":
		return float64(input.CodeSmells)
	default:
		return 0
	}
}

func compare(actual float64, op Operator, threshold float64) bool {
	switch op {
	case OpGT:
		return actual <= threshold
	case OpLT:
		return actual >= threshold
	default:
		return actual <= threshold
	}
}

func StatusTR(status string) string {
	return statusTR(Status(status))
}

func statusTR(status Status) string {
	switch status {
	case StatusPass:
		return "Geçti"
	case StatusWarn:
		return "Uyarı"
	case StatusFail:
		return "Kaldı"
	default:
		return string(status)
	}
}
