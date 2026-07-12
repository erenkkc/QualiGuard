package gate

import "github.com/qualiguard/qualiguard/internal/model"

func ToModelResult(r Result) model.GateResult {
	conds := make([]model.GateCondition, len(r.Conditions))
	for i, c := range r.Conditions {
		conds[i] = model.GateCondition{
			Metric:    c.Metric,
			LabelTR:   c.LabelTR,
			Operator:  c.Operator,
			Threshold: c.Threshold,
			Actual:    c.Actual,
			Passed:    c.Passed,
			Level:     c.Level,
		}
	}
	return model.GateResult{
		Name:       r.Name,
		NameTR:     r.NameTR,
		Status:     string(r.Status),
		StatusTR:   r.StatusTR,
		Conditions: conds,
	}
}
