package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/qualiguard/qualiguard/internal/gate"
	"github.com/qualiguard/qualiguard/internal/model"
	"github.com/qualiguard/qualiguard/internal/reporter"
)

func (s *Server) handleDeleteProject(w http.ResponseWriter, r *http.Request) {
	project, err := s.getProjectByKey(w, r)
	if err != nil || project == nil {
		return
	}
	if err := s.store.DeleteProject(r.Context(), project.ID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "key": project.Key})
}

func (s *Server) handleBulkDeleteProjects(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Keys []string `json:"keys"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if len(req.Keys) == 0 {
		writeError(w, http.StatusBadRequest, "keys required")
		return
	}
	deleted, notFound, err := s.store.DeleteProjectsByKeys(r.Context(), req.Keys)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":    "ok",
		"deleted":   deleted,
		"not_found": notFound,
		"count":     len(deleted),
	})
}

func (s *Server) handleExportProject(w http.ResponseWriter, r *http.Request) {
	project, err := s.getProjectByKey(w, r)
	if err != nil || project == nil {
		return
	}
	issues, err := s.store.ListIssuesByProject(r.Context(), project.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	overview, err := s.store.GetProjectOverview(r.Context(), project.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
	if format == "" {
		format = "json"
	}

	filename := fmt.Sprintf("qualiguard-%s-%s", project.Key, time.Now().Format("20060102"))

	switch format {
	case "html":
		html := reporter.RenderHTMLReport(project.Name, project.Key, overview.Gate, issues, overview.Measures)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.html"`, filename))
		_, _ = w.Write([]byte(html))
		return
	case "json":
		report := buildExportReport(project, overview, issues)
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.json"`, filename))
		_, _ = w.Write(data)
		return
	default:
		writeError(w, http.StatusBadRequest, "format must be json or html")
	}
}

func buildExportReport(project *model.StoredProject, overview *model.ProjectOverview, issues []model.Issue) map[string]any {
	measures := model.Measures{}
	if overview.Measures != nil {
		measures = model.Measures{
			Files:           int(overview.Measures["files"]),
			Ncloc:           int(overview.Measures["ncloc"]),
			Complexity:      int(overview.Measures["complexity"]),
			Bugs:            int(overview.Measures["bugs"]),
			Vulnerabilities: int(overview.Measures["vulnerabilities"]),
			CodeSmells:      int(overview.Measures["code_smells"]),
		}
	}
	gateResult := gate.ToModelResult(gate.Evaluate(gate.InputFromIssues(openIssuesOnly(issues))))
	if overview.Gate != nil {
		gateResult = *overview.Gate
	}
	return map[string]any{
		"exported_at": time.Now().UTC().Format(time.RFC3339),
		"project": map[string]string{
			"key":  project.Key,
			"name": project.Name,
		},
		"gate":     gateResult,
		"measures": measures,
		"issues":   issues,
		"summary": map[string]int{
			"open_issues":     overview.OpenIssues,
			"bugs":            overview.Bugs,
			"vulnerabilities": overview.Vulnerabilities,
			"code_smells":     overview.CodeSmells,
		},
	}
}

func openIssuesOnly(issues []model.Issue) []model.Issue {
	out := make([]model.Issue, 0, len(issues))
	for _, i := range issues {
		if i.Status == "OPEN" || i.Status == "REOPENED" {
			out = append(out, i)
		}
	}
	return out
}
