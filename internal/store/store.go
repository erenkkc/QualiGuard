package store

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/qualiguard/qualiguard/internal/gate"
	"github.com/qualiguard/qualiguard/internal/model"
	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	db.SetMaxOpenConns(1)

	if _, err := db.Exec(schemaSQL); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	migrateColumns(db)

	return &Store{db: db}, nil
}

func migrateColumns(db *sql.DB) {
	_, _ = db.Exec(`ALTER TABLE issues ADD COLUMN snippet TEXT NOT NULL DEFAULT ''`)
	_, _ = db.Exec(`ALTER TABLE issues ADD COLUMN fix_suggestion TEXT NOT NULL DEFAULT ''`)
	_, _ = db.Exec(`ALTER TABLE analyses ADD COLUMN gate_status TEXT NOT NULL DEFAULT ''`)
	_, _ = db.Exec(`ALTER TABLE analyses ADD COLUMN issues_found INTEGER NOT NULL DEFAULT 0`)
	_, _ = db.Exec(`ALTER TABLE analyses ADD COLUMN issues_new INTEGER NOT NULL DEFAULT 0`)
	_, _ = db.Exec(`ALTER TABLE analyses ADD COLUMN issues_open INTEGER NOT NULL DEFAULT 0`)
	_, _ = db.Exec(`ALTER TABLE analyses ADD COLUMN issues_closed INTEGER NOT NULL DEFAULT 0`)
	_, _ = db.Exec(`ALTER TABLE analyses ADD COLUMN source_filename TEXT NOT NULL DEFAULT ''`)
	_, _ = db.Exec(`ALTER TABLE analyses ADD COLUMN source_text TEXT NOT NULL DEFAULT ''`)
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) EnsureDefaultToken(ctx context.Context) (string, error) {
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM api_tokens`).Scan(&count); err != nil {
		return "", err
	}
	if count > 0 {
		var token string
		err := s.db.QueryRowContext(ctx, `SELECT token FROM api_tokens ORDER BY created_at ASC LIMIT 1`).Scan(&token)
		return token, err
	}

	token := "qg_" + uuid.NewString()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO api_tokens (id, token, name, created_at) VALUES (?, ?, ?, ?)`,
		uuid.NewString(), token, "default", time.Now().UTC().Format(time.RFC3339),
	)
	return token, err
}

func (s *Store) ValidateToken(ctx context.Context, token string) bool {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM api_tokens WHERE token = ?`, token).Scan(&count)
	return err == nil && count > 0
}

func (s *Store) CreateProject(ctx context.Context, key, name string) (*model.StoredProject, error) {
	now := time.Now().UTC()
	project := &model.StoredProject{
		ID:        uuid.NewString(),
		Key:       key,
		Name:      name,
		CreatedAt: now,
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO projects (id, key, name, created_at) VALUES (?, ?, ?, ?)`,
		project.ID, project.Key, project.Name, now.Format(time.RFC3339),
	)
	if err != nil {
		return nil, fmt.Errorf("create project: %w", err)
	}
	return project, nil
}

func (s *Store) GetProjectByKey(ctx context.Context, key string) (*model.StoredProject, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, key, name, created_at FROM projects WHERE key = ?`, key,
	)
	var p model.StoredProject
	var created string
	if err := row.Scan(&p.ID, &p.Key, &p.Name, &created); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	p.CreatedAt, _ = time.Parse(time.RFC3339, created)
	return &p, nil
}

func (s *Store) EnsureProject(ctx context.Context, key, name string) (*model.StoredProject, error) {
	p, err := s.GetProjectByKey(ctx, key)
	if err != nil {
		return nil, err
	}
	if p != nil {
		return p, nil
	}
	if name == "" {
		name = key
	}
	return s.CreateProject(ctx, key, name)
}

func (s *Store) ListProjects(ctx context.Context) ([]model.StoredProject, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, key, name, created_at FROM projects ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []model.StoredProject
	for rows.Next() {
		var p model.StoredProject
		var created string
		if err := rows.Scan(&p.ID, &p.Key, &p.Name, &created); err != nil {
			return nil, err
		}
		p.CreatedAt, _ = time.Parse(time.RFC3339, created)
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

func (s *Store) CreateAnalysis(ctx context.Context, projectID string, report *model.Report, gateStatus string) (string, error) {
	id := uuid.NewString()
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO analyses (id, project_id, branch, commit_sha, status, scanner_version, duration_ms, gate_status, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, projectID, report.Analysis.Branch, report.Analysis.Commit,
		"SUCCESS", report.ScannerVersion, report.Analysis.DurationMS, gateStatus, now,
	)
	return id, err
}

func (s *Store) UpdateAnalysisSummary(ctx context.Context, analysisID, gateStatus string, found, newCount, openCount, closedCount int) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE analyses SET
			gate_status = ?,
			issues_found = ?,
			issues_new = ?,
			issues_open = ?,
			issues_closed = ?
		WHERE id = ?`,
		gateStatus, found, newCount, openCount, closedCount, analysisID,
	)
	return err
}

func (s *Store) SaveAnalysisSource(ctx context.Context, analysisID, filename, source string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE analyses SET source_filename = ?, source_text = ? WHERE id = ?`,
		filename, source, analysisID,
	)
	return err
}

func (s *Store) GetLatestProjectSource(ctx context.Context, projectID string) (filename, source string, err error) {
	err = s.db.QueryRowContext(ctx, `
		SELECT source_filename, source_text
		FROM analyses
		WHERE project_id = ? AND source_text != ''
		ORDER BY created_at DESC
		LIMIT 1`, projectID).Scan(&filename, &source)
	if err == sql.ErrNoRows {
		return "", "", nil
	}
	return filename, source, err
}

func (s *Store) SaveMeasures(ctx context.Context, analysisID string, measures model.Measures) error {
	values := map[string]float64{
		"ncloc":             float64(measures.Ncloc),
		"files":             float64(measures.Files),
		"complexity":        float64(measures.Complexity),
		"bugs":              float64(measures.Bugs),
		"vulnerabilities":   float64(measures.Vulnerabilities),
		"code_smells":       float64(measures.CodeSmells),
		"security_hotspots": float64(measures.SecurityHotspots),
	}
	for key, value := range values {
		if _, err := s.db.ExecContext(ctx,
			`INSERT INTO measures (id, analysis_id, metric_key, value) VALUES (?, ?, ?, ?)`,
			uuid.NewString(), analysisID, key, value,
		); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) ListIssuesByProject(ctx context.Context, projectID string) ([]model.Issue, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, rule_key, severity, type, message, file_path, line, column_num,
		       effort_minutes, fingerprint, status, COALESCE(resolution, ''),
		       snippet, fix_suggestion
		FROM issues WHERE project_id = ? ORDER BY
		  CASE severity
		    WHEN 'BLOCKER' THEN 1 WHEN 'CRITICAL' THEN 2 WHEN 'MAJOR' THEN 3
		    WHEN 'MINOR' THEN 4 ELSE 5 END, file_path, line`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var issues []model.Issue
	for rows.Next() {
		var issue model.Issue
		var resolution string
		if err := rows.Scan(
			&issue.ID, &issue.RuleKey, &issue.Severity, &issue.Type, &issue.Message,
			&issue.File, &issue.Line, &issue.Column, &issue.EffortMin,
			&issue.Fingerprint, &issue.Status, &resolution,
			&issue.Snippet, &issue.FixSuggestion,
		); err != nil {
			return nil, err
		}
		issue.Resolution = model.Resolution(resolution)
		issues = append(issues, issue)
	}
	return issues, rows.Err()
}

func (s *Store) GetLatestMeasures(ctx context.Context, projectID string) (map[string]float64, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id FROM analyses WHERE project_id = ? ORDER BY created_at DESC LIMIT 1`, projectID)
	var analysisID string
	if err := row.Scan(&analysisID); err != nil {
		if err == sql.ErrNoRows {
			return map[string]float64{}, nil
		}
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT metric_key, value FROM measures WHERE analysis_id = ?`, analysisID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string]float64{}
	for rows.Next() {
		var key string
		var value float64
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		out[key] = value
	}
	return out, rows.Err()
}

func (s *Store) OpenIssueFingerprints(ctx context.Context, projectID string) (map[string]string, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT fingerprint, id FROM issues WHERE project_id = ? AND status IN ('OPEN', 'REOPENED')`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string]string{}
	for rows.Next() {
		var fp, id string
		if err := rows.Scan(&fp, &id); err != nil {
			return nil, err
		}
		out[fp] = id
	}
	return out, rows.Err()
}

func (s *Store) UpsertIssue(ctx context.Context, projectID, analysisID string, issue model.Issue, existingID string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	if existingID == "" {
		id := uuid.NewString()
		_, err := s.db.ExecContext(ctx, `
			INSERT INTO issues (
				id, project_id, analysis_id, rule_key, severity, type, message,
				file_path, line, column_num, effort_minutes, fingerprint, status,
				snippet, fix_suggestion,
				first_seen_analysis_id, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'OPEN', ?, ?, ?, ?, ?)`,
			id, projectID, analysisID, issue.RuleKey, issue.Severity, issue.Type, issue.Message,
			issue.File, issue.Line, issue.Column, issue.EffortMin, issue.Fingerprint,
			issue.Snippet, issue.FixSuggestion,
			analysisID, now, now,
		)
		return err
	}

	_, err := s.db.ExecContext(ctx, `
		UPDATE issues SET
			analysis_id = ?, rule_key = ?, severity = ?, type = ?, message = ?,
			file_path = ?, line = ?, column_num = ?, effort_minutes = ?,
			snippet = ?, fix_suggestion = ?,
			status = 'OPEN', resolution = NULL, updated_at = ?
		WHERE id = ?`,
		analysisID, issue.RuleKey, issue.Severity, issue.Type, issue.Message,
		issue.File, issue.Line, issue.Column, issue.EffortMin,
		issue.Snippet, issue.FixSuggestion,
		now, existingID,
	)
	return err
}

func (s *Store) CloseMissingIssues(ctx context.Context, projectID string, seen map[string]struct{}) (int, error) {
	openFPs, err := s.OpenIssueFingerprints(ctx, projectID)
	if err != nil {
		return 0, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	closed := 0
	for fp, id := range openFPs {
		if _, ok := seen[fp]; ok {
			continue
		}
		if _, err := s.db.ExecContext(ctx, `
			UPDATE issues SET status = 'CLOSED', resolution = 'FIXED', updated_at = ? WHERE id = ?`,
			now, id,
		); err != nil {
			return closed, err
		}
		closed++
	}
	return closed, nil
}

func (s *Store) CountOpenIssues(ctx context.Context, projectID string) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM issues WHERE project_id = ? AND status IN ('OPEN', 'REOPENED')`, projectID,
	).Scan(&count)
	return count, err
}

func (s *Store) GetLastAnalysisTime(ctx context.Context, projectID string) (string, error) {
	var created string
	err := s.db.QueryRowContext(ctx,
		`SELECT created_at FROM analyses WHERE project_id = ? ORDER BY created_at DESC LIMIT 1`, projectID,
	).Scan(&created)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return created, err
}

func (s *Store) CountOpenIssuesByType(ctx context.Context, projectID string) (bugs, vulns, smells int, err error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT type, COUNT(*) FROM issues
		WHERE project_id = ? AND status IN ('OPEN', 'REOPENED')
		GROUP BY type`, projectID)
	if err != nil {
		return 0, 0, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var typ string
		var count int
		if err := rows.Scan(&typ, &count); err != nil {
			return 0, 0, 0, err
		}
		switch model.IssueType(typ) {
		case model.TypeBug:
			bugs = count
		case model.TypeVulnerability:
			vulns = count
		case model.TypeCodeSmell:
			smells = count
		}
	}
	return bugs, vulns, smells, rows.Err()
}

func (s *Store) OpenGateInput(ctx context.Context, projectID string) (gate.Input, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT severity, type, COUNT(*) FROM issues
		 WHERE project_id = ? AND status IN ('OPEN', 'REOPENED')
		 GROUP BY severity, type`, projectID,
	)
	if err != nil {
		return gate.Input{}, err
	}
	defer rows.Close()

	var input gate.Input
	for rows.Next() {
		var severity, issueType string
		var count int
		if err := rows.Scan(&severity, &issueType, &count); err != nil {
			return gate.Input{}, err
		}
		switch severity {
		case string(model.SeverityBlocker):
			input.BlockerIssues += count
		case string(model.SeverityCritical):
			input.CriticalIssues += count
		}
		switch issueType {
		case string(model.TypeBug):
			input.Bugs += count
		case string(model.TypeVulnerability):
			input.Vulnerabilities += count
		case string(model.TypeCodeSmell):
			input.CodeSmells += count
		}
	}
	return input, rows.Err()
}

func (s *Store) GetProjectOverview(ctx context.Context, projectID string) (*model.ProjectOverview, error) {
	project, err := s.getProjectByID(ctx, projectID)
	if err != nil {
		return nil, err
	}

	open, err := s.CountOpenIssues(ctx, projectID)
	if err != nil {
		return nil, err
	}
	bugs, vulns, smells, err := s.CountOpenIssuesByType(ctx, projectID)
	if err != nil {
		return nil, err
	}
	measures, err := s.GetLatestMeasures(ctx, projectID)
	if err != nil {
		return nil, err
	}
	lastAt, err := s.GetLastAnalysisTime(ctx, projectID)
	if err != nil {
		return nil, err
	}

	gateInput, err := s.OpenGateInput(ctx, projectID)
	if err != nil {
		return nil, err
	}
	gateResult := gate.ToModelResult(gate.Evaluate(gateInput))

	return &model.ProjectOverview{
		StoredProject:   *project,
		OpenIssues:      open,
		Bugs:            bugs,
		Vulnerabilities: vulns,
		CodeSmells:      smells,
		Measures:        measures,
		LastAnalysisAt:  lastAt,
		Gate:            &gateResult,
	}, nil
}

func (s *Store) ListProjectOverviews(ctx context.Context) ([]model.ProjectOverview, error) {
	projects, err := s.ListProjects(ctx)
	if err != nil {
		return nil, err
	}
	var out []model.ProjectOverview
	for _, p := range projects {
		overview, err := s.GetProjectOverview(ctx, p.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, *overview)
	}
	return out, nil
}

func (s *Store) getProjectByID(ctx context.Context, id string) (*model.StoredProject, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, key, name, created_at FROM projects WHERE id = ?`, id,
	)
	var p model.StoredProject
	var created string
	if err := row.Scan(&p.ID, &p.Key, &p.Name, &created); err != nil {
		return nil, err
	}
	p.CreatedAt, _ = time.Parse(time.RFC3339, created)
	return &p, nil
}

func (s *Store) ListGateHistory(ctx context.Context, projectID string, limit int) ([]model.GateHistoryEntry, error) {
	entries, err := s.listAnalysisHistory(ctx, projectID, limit)
	if err != nil {
		return nil, err
	}
	out := make([]model.GateHistoryEntry, len(entries))
	for i, e := range entries {
		out[i] = model.GateHistoryEntry{
			AnalysisID:   e.AnalysisID,
			GateStatus:   e.GateStatus,
			GateStatusTR: e.GateStatusTR,
			OpenIssues:   e.IssuesOpen,
			CreatedAt:    e.CreatedAt,
		}
	}
	return out, nil
}

func (s *Store) ListGlobalHistory(ctx context.Context, limit int) ([]model.AnalysisHistoryEntry, error) {
	return s.listAnalysisHistory(ctx, "", limit)
}

func (s *Store) listAnalysisHistory(ctx context.Context, projectID string, limit int) ([]model.AnalysisHistoryEntry, error) {
	if limit <= 0 {
		limit = 50
	}

	query := `
		SELECT a.id, p.key, p.name, COALESCE(a.gate_status, ''), COALESCE(a.issues_found, 0),
		       COALESCE(a.issues_new, 0), COALESCE(a.issues_open, 0), COALESCE(a.issues_closed, 0),
		       COALESCE(a.scanner_version, ''), a.created_at
		FROM analyses a
		JOIN projects p ON p.id = a.project_id`
	args := []any{}
	if projectID != "" {
		query += ` WHERE a.project_id = ?`
		args = append(args, projectID)
	}
	query += ` ORDER BY a.created_at DESC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []model.AnalysisHistoryEntry
	for rows.Next() {
		var entry model.AnalysisHistoryEntry
		if err := rows.Scan(
			&entry.AnalysisID, &entry.ProjectKey, &entry.ProjectName, &entry.GateStatus,
			&entry.IssuesFound, &entry.IssuesNew, &entry.IssuesOpen, &entry.IssuesClosed,
			&entry.ScannerVersion, &entry.CreatedAt,
		); err != nil {
			return nil, err
		}
		if entry.GateStatus == "" {
			entry.GateStatus = "PASS"
		}
		entry.GateStatusTR = gateStatusLabel(entry.GateStatus)
		out = append(out, entry)
	}
	return out, rows.Err()
}

func gateStatusLabel(status string) string {
	switch status {
	case "PASS":
		return "Geçti"
	case "WARN":
		return "Uyarı"
	case "FAIL":
		return "Kaldı"
	default:
		return status
	}
}

// DeleteProject removes a project and all related analyses, issues, and measures.
func (s *Store) DeleteProject(ctx context.Context, projectID string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	rows, err := tx.QueryContext(ctx, `SELECT id FROM analyses WHERE project_id = ?`, projectID)
	if err != nil {
		return err
	}
	var analysisIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		analysisIDs = append(analysisIDs, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	for _, aid := range analysisIDs {
		if _, err := tx.ExecContext(ctx, `DELETE FROM measures WHERE analysis_id = ?`, aid); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM issues WHERE project_id = ?`, projectID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM analyses WHERE project_id = ?`, projectID); err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx, `DELETE FROM projects WHERE id = ?`, projectID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return tx.Commit()
}

// DeleteProjectsByKeys removes multiple projects by key. Returns deleted keys and keys not found.
func (s *Store) DeleteProjectsByKeys(ctx context.Context, keys []string) (deleted []string, notFound []string, err error) {
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		project, lookupErr := s.GetProjectByKey(ctx, key)
		if lookupErr != nil || project == nil {
			notFound = append(notFound, key)
			continue
		}
		if err := s.DeleteProject(ctx, project.ID); err != nil {
			return deleted, notFound, err
		}
		deleted = append(deleted, key)
	}
	return deleted, notFound, nil
}
