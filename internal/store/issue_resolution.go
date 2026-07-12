package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/qualiguard/qualiguard/internal/gate"
	"github.com/qualiguard/qualiguard/internal/model"
)

type IssueFingerprintRef struct {
	ID         string
	Status     model.IssueStatus
	Resolution model.Resolution
}

func (s *Store) IssueFingerprintIndex(ctx context.Context, projectID string) (map[string]IssueFingerprintRef, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT fingerprint, id, status, COALESCE(resolution, '')
		FROM issues WHERE project_id = ?`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string]IssueFingerprintRef{}
	for rows.Next() {
		var fp, id, status, resolution string
		if err := rows.Scan(&fp, &id, &status, &resolution); err != nil {
			return nil, err
		}
		out[fp] = IssueFingerprintRef{
			ID:         id,
			Status:     model.IssueStatus(status),
			Resolution: model.Resolution(resolution),
		}
	}
	return out, rows.Err()
}

func (s *Store) GetIssueByID(ctx context.Context, projectID, issueID string) (*model.Issue, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, rule_key, severity, type, message, file_path, line, column_num,
		       effort_minutes, fingerprint, status, COALESCE(resolution, ''),
		       snippet, fix_suggestion
		FROM issues WHERE project_id = ? AND id = ?`, projectID, issueID)

	var issue model.Issue
	var resolution string
	if err := row.Scan(
		&issue.ID, &issue.RuleKey, &issue.Severity, &issue.Type, &issue.Message,
		&issue.File, &issue.Line, &issue.Column, &issue.EffortMin,
		&issue.Fingerprint, &issue.Status, &resolution,
		&issue.Snippet, &issue.FixSuggestion,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	issue.Resolution = model.Resolution(resolution)
	return &issue, nil
}

func (s *Store) ResolveIssue(ctx context.Context, projectID, issueID string, resolution model.Resolution) error {
	if resolution != model.ResolutionFalsePositive && resolution != model.ResolutionWontFix {
		return fmt.Errorf("unsupported resolution: %s", resolution)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.ExecContext(ctx, `
		UPDATE issues SET status = 'CLOSED', resolution = ?, updated_at = ?
		WHERE project_id = ? AND id = ? AND status IN ('OPEN', 'REOPENED')`,
		string(resolution), now, projectID, issueID,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("issue not found or already closed")
	}
	return nil
}

func (s *Store) ReopenIssue(ctx context.Context, projectID, issueID string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.ExecContext(ctx, `
		UPDATE issues SET status = 'OPEN', resolution = NULL, updated_at = ?
		WHERE project_id = ? AND id = ?
		  AND status = 'CLOSED'
		  AND resolution IN ('FALSE_POSITIVE', 'WONTFIX')`,
		now, projectID, issueID,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("issue not found or not a suppressed issue")
	}
	return nil
}

func (s *Store) ProjectGateResult(ctx context.Context, projectID string) (model.GateResult, error) {
	input, err := s.OpenGateInput(ctx, projectID)
	if err != nil {
		return model.GateResult{}, err
	}
	return gate.ToModelResult(gate.Evaluate(input)), nil
}

func IsSuppressedResolution(res model.Resolution) bool {
	return res == model.ResolutionFalsePositive || res == model.ResolutionWontFix
}
