package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/qualiguard/qualiguard/internal/ai"
	"github.com/qualiguard/qualiguard/internal/config"
	"github.com/qualiguard/qualiguard/internal/model"
	"github.com/qualiguard/qualiguard/internal/processor"
	"github.com/qualiguard/qualiguard/internal/scanner"
	"github.com/qualiguard/qualiguard/internal/store"
	"github.com/qualiguard/qualiguard/internal/webui"
)

type Server struct {
	store         *store.Store
	processor     *processor.Processor
	token         string
	explainer     *ai.Explainer
	workDir       string
	configPath    string
	brand         config.BrandConfig
	panelPassword string
	publicSite    bool
}

func publicSiteFromEnv() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("QG_PUBLIC_SITE")))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

func New(s *store.Store, token, workDir, configPath string) *Server {
	brand := config.DefaultBrand()
	if cfg, err := config.Load(configPath); err == nil {
		ai.ConfigureFromYAML(cfg.AI)
		brand = cfg.Brand
	}
	return &Server{
		store:         s,
		processor:     processor.New(s),
		token:         token,
		explainer:     ai.NewExplainer(ai.LoadConfig().Enabled),
		workDir:       workDir,
		configPath:    configPath,
		brand:         brand.WithDefaults(),
		panelPassword: panelPasswordFromEnv(),
		publicSite:    publicSiteFromEnv(),
	}
}

func (s *Server) Handler() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(180 * time.Second))
	r.Use(SecurityHeaders)

	r.Get("/api/health", s.handleHealth)
	r.Get("/api/public/brand", s.handlePublicConfig)
	r.Get("/api/public/config", s.handlePublicConfig)
	r.Post("/api/auth/register", s.handleRegister)
	r.Post("/api/auth/login", s.handlePanelLogin)
	r.Post("/api/auth/logout", s.handleLogout)
	r.Get("/api/auth/me", s.handleMe)
	r.Get("/api/bootstrap", s.handleBootstrap)

	r.Route("/api/v1", func(r chi.Router) {
		r.Use(s.authMiddleware)
		// Website: only downloads. Workstation panel APIs stay on desktop app.
		r.Get("/downloads", s.handleDownloadsList)
		r.Get("/downloads/{id}", s.handleDownloadFile)
		if !s.publicSite {
			r.Get("/projects/overview", s.handleListProjectOverviews)
			r.Get("/projects", s.handleListProjects)
			r.Post("/projects", s.handleCreateProject)
			r.Get("/projects/{key}", s.handleGetProject)
			r.Get("/projects/{key}/overview", s.handleGetProjectOverview)
			r.Get("/projects/{key}/gate", s.handleGetProjectGate)
			r.Get("/projects/{key}/history", s.handleGetProjectHistory)
			r.Get("/projects/{key}/issues", s.handleListIssues)
			r.Patch("/projects/{key}/issues/{id}", s.handleResolveIssue)
			r.Post("/projects/bulk-delete", s.handleBulkDeleteProjects)
			r.Delete("/projects/{key}", s.handleDeleteProject)
			r.Get("/projects/{key}/export", s.handleExportProject)
			r.Get("/projects/{key}/measures", s.handleGetMeasures)
			r.Post("/projects/{key}/rescan", s.handleProjectRescan)
			r.Get("/demo/{name}", s.handleDemoSample)
			r.Get("/history", s.handleGlobalHistory)
			r.Post("/import/preview", s.handleImportPreview)
			r.Post("/import/file", s.handleImportFile)
			r.Post("/analyses", s.handleUploadAnalysis)
			r.Post("/analyze/code", s.handleAnalyzeCode)
			r.Post("/explain/issue", s.handleExplainIssue)
			r.Post("/chat", s.handleAIChat)
		}
	})

	r.Handle("/*", webui.Handler(webui.PanelAuth{
		InjectToken: !s.panelAuthRequired() && !s.publicSite,
		Token:       s.token,
		PublicSite:  s.publicSite,
	}, s.brand))

	return r
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		token = strings.TrimSpace(token)
		if token == "" {
			token = r.Header.Get("X-QualiGuard-Token")
		}
		if token == "" || !s.store.ValidateAuthToken(r.Context(), token) {
			writeError(w, http.StatusUnauthorized, "invalid or missing API token")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleBootstrap(w http.ResponseWriter, r *http.Request) {
	if s.publicSite {
		writeError(w, http.StatusForbidden, "panel yalnızca masaüstü uygulamasında")
		return
	}
	if s.panelAuthRequired() {
		writeError(w, http.StatusForbidden, "use /api/auth/login")
		return
	}
	host := r.Host
	if !strings.HasPrefix(host, "127.0.0.1") && !strings.HasPrefix(host, "localhost") {
		writeError(w, http.StatusForbidden, "bootstrap only available on localhost")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"token": s.token,
		"ai":    ai.AssistantStatus(),
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"service": "qualiguard-server",
		"ai":      ai.AssistantStatus(),
	})
}

func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := s.store.ListProjects(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if projects == nil {
		projects = []model.StoredProject{}
	}
	writeJSON(w, http.StatusOK, projects)
}

func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Key  string `json:"key"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Key == "" {
		writeError(w, http.StatusBadRequest, "key is required")
		return
	}
	if req.Name == "" {
		req.Name = req.Key
	}

	existing, err := s.store.GetProjectByKey(r.Context(), req.Key)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if existing != nil {
		writeError(w, http.StatusConflict, "project already exists")
		return
	}

	project, err := s.store.CreateProject(r.Context(), req.Key, req.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, project)
}

func (s *Server) handleListProjectOverviews(w http.ResponseWriter, r *http.Request) {
	projects, err := s.store.ListProjectOverviews(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if projects == nil {
		projects = []model.ProjectOverview{}
	}
	writeJSON(w, http.StatusOK, projects)
}

func (s *Server) handleGetProjectOverview(w http.ResponseWriter, r *http.Request) {
	project, err := s.getProjectByKey(w, r)
	if err != nil || project == nil {
		return
	}
	overview, err := s.store.GetProjectOverview(r.Context(), project.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, overview)
}

func (s *Server) handleGetProject(w http.ResponseWriter, r *http.Request) {
	project, err := s.getProjectByKey(w, r)
	if err != nil || project == nil {
		return
	}
	writeJSON(w, http.StatusOK, project)
}

func (s *Server) handleGetProjectHistory(w http.ResponseWriter, r *http.Request) {
	project, err := s.getProjectByKey(w, r)
	if err != nil || project == nil {
		return
	}
	history, err := s.store.ListGateHistory(r.Context(), project.ID, 10)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if history == nil {
		history = []model.GateHistoryEntry{}
	}
	writeJSON(w, http.StatusOK, history)
}

func (s *Server) handleGetProjectGate(w http.ResponseWriter, r *http.Request) {
	project, err := s.getProjectByKey(w, r)
	if err != nil || project == nil {
		return
	}
	overview, err := s.store.GetProjectOverview(r.Context(), project.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if overview.Gate == nil {
		writeJSON(w, http.StatusOK, map[string]string{"status": "PASS", "status_tr": "Geçti"})
		return
	}
	writeJSON(w, http.StatusOK, overview.Gate)
}

func (s *Server) handleListIssues(w http.ResponseWriter, r *http.Request) {
	project, err := s.getProjectByKey(w, r)
	if err != nil || project == nil {
		return
	}
	issues, err := s.store.ListIssuesByProject(r.Context(), project.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if issues == nil {
		issues = []model.Issue{}
	}
	writeJSON(w, http.StatusOK, issues)
}

func (s *Server) handleResolveIssue(w http.ResponseWriter, r *http.Request) {
	project, err := s.getProjectByKey(w, r)
	if err != nil {
		return
	}

	issueID := chi.URLParam(r, "id")
	var body struct {
		Resolution string `json:"resolution"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	switch body.Resolution {
	case string(model.ResolutionFalsePositive), string(model.ResolutionWontFix):
		if err := s.store.ResolveIssue(r.Context(), project.ID, issueID, model.Resolution(body.Resolution)); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	case "", "REOPEN":
		if err := s.store.ReopenIssue(r.Context(), project.ID, issueID); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	default:
		writeError(w, http.StatusBadRequest, "resolution must be FALSE_POSITIVE, WONTFIX, or REOPEN")
		return
	}

	issue, err := s.store.GetIssueByID(r.Context(), project.ID, issueID)
	if err != nil || issue == nil {
		writeError(w, http.StatusNotFound, "issue not found")
		return
	}

	gateResult, err := s.store.ProjectGateResult(r.Context(), project.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	openIssues, err := s.store.CountOpenIssues(r.Context(), project.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"issue":       issue,
		"gate":        gateResult,
		"open_issues": openIssues,
	})
}

func (s *Server) handleGetMeasures(w http.ResponseWriter, r *http.Request) {
	project, err := s.getProjectByKey(w, r)
	if err != nil || project == nil {
		return
	}
	measures, err := s.store.GetLatestMeasures(r.Context(), project.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, measures)
}

func (s *Server) handleExplainIssue(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Issue    model.Issue `json:"issue"`
		CodeLine string      `json:"code_line"`
		Question string      `json:"question"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if strings.TrimSpace(req.Issue.RuleKey) == "" {
		writeError(w, http.StatusBadRequest, "issue.rule_key is required")
		return
	}

	exp := s.explainer.ExplainOnDemand(r.Context(), req.Issue, req.CodeLine, req.Question)
	status := ai.AssistantStatus()
	writeJSON(w, http.StatusOK, map[string]any{
		"explanation": exp,
		"mode":        status["mode"],
		"llm_active":  status["active"],
		"provider":    status["provider"],
		"model":       status["model"],
	})
}

func (s *Server) handleAIChat(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Messages []ai.ChatMessage `json:"messages"`
		Stream   bool             `json:"stream"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Stream {
		s.handleAIChatStream(w, r, req.Messages)
		return
	}
	reply, err := s.explainer.Chat(r.Context(), req.Messages)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	status := ai.AssistantStatus()
	writeJSON(w, http.StatusOK, map[string]any{
		"reply":      reply,
		"llm_active": status["active"],
		"provider":   status["provider"],
		"model":      status["model"],
	})
}

func (s *Server) handleAIChatStream(w http.ResponseWriter, r *http.Request, messages []ai.ChatMessage) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming desteklenmiyor")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	writeSSE := func(payload any) {
		data, _ := json.Marshal(payload)
		_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}

	reply, err := s.explainer.ChatStream(r.Context(), messages, func(delta string) error {
		writeSSE(map[string]string{"delta": delta})
		return nil
	})
	status := ai.AssistantStatus()
	if err != nil {
		writeSSE(map[string]string{"error": err.Error()})
		return
	}
	writeSSE(map[string]any{
		"done":       reply,
		"llm_active": status["active"],
		"provider":   status["provider"],
		"model":      status["model"],
	})
}

func (s *Server) handleAnalyzeCode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Language string `json:"language"`
		Source   string `json:"source"`
		Filename string `json:"filename"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if strings.TrimSpace(req.Source) == "" {
		writeError(w, http.StatusBadRequest, "source is required")
		return
	}

	var report *model.Report
	var err error
	filename := strings.TrimSpace(req.Filename)
	if filename != "" {
		report, err = scanner.AnalyzeSourceFile(r.Context(), req.Source, filename, "1.0.0", "vscode", "VS Code")
	} else {
		report, err = scanner.AnalyzePlaygroundSource(r.Context(), req.Source, "1.0.0")
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (s *Server) handleProjectRescan(w http.ResponseWriter, r *http.Request) {
	project, err := s.getProjectByKey(w, r)
	if err != nil || project == nil {
		return
	}
	filename, source, err := s.store.GetLatestProjectSource(r.Context(), project.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if strings.TrimSpace(source) == "" {
		writeError(w, http.StatusBadRequest, "bu projede kaynak kod yok")
		return
	}

	report, err := scanner.RescanStoredSource(r.Context(), project.Key, project.Name, filename, source, "1.0.0")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	result, err := s.processor.ProcessReport(r.Context(), report)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleDemoSample(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(chi.URLParam(r, "name"))
	allowed := map[string]string{
		"python-kritik":   "demos/python-kritik.py",
		"python-stil":     "demos/python-stil.py",
		"javascript-stil": "demos/javascript-stil.js",
	}
	rel, ok := allowed[name]
	if !ok {
		writeError(w, http.StatusNotFound, "demo bulunamadı")
		return
	}
	path := rel
	if s.workDir != "" {
		path = filepath.Join(s.workDir, rel)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		writeError(w, http.StatusNotFound, "demo dosyası okunamadı")
		return
	}
	base := filepath.Base(path)
	writeJSON(w, http.StatusOK, map[string]any{
		"name":     name,
		"filename": base,
		"source":   string(data),
	})
}

func (s *Server) handleGlobalHistory(w http.ResponseWriter, r *http.Request) {
	history, err := s.store.ListGlobalHistory(r.Context(), 100)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if history == nil {
		history = []model.AnalysisHistoryEntry{}
	}
	writeJSON(w, http.StatusOK, history)
}

func (s *Server) handleScanAndUpload(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ProjectKey  string `json:"project_key"`
		ProjectName string `json:"project_name"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	cfg, err := config.Load(s.configPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "config yüklenemedi: "+err.Error())
		return
	}
	if req.ProjectKey != "" {
		cfg.Project.Key = req.ProjectKey
	}
	if req.ProjectName != "" {
		cfg.Project.Name = req.ProjectName
	}

	sc := scanner.New(cfg, s.workDir, "1.0.0")
	report, err := sc.Scan(r.Context(), scanner.ScanOptions{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	result, err := s.processor.ProcessReport(r.Context(), report)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (s *Server) handleImportPreview(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/") {
		s.handleImportZip(w, r, false)
		return
	}
	report, err := s.analyzeImportRequest(r)
	if err != nil {
		writeImportError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"filename": report.Source.Filename,
		"language": report.Source.Language,
		"source":   report.Source.Text,
		"issues":   report.Issues,
		"gate":     report.Gate,
		"measures": report.Measures,
	})
}

func (s *Server) handleImportFile(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/") {
		s.handleImportZip(w, r, true)
		return
	}
	report, err := s.analyzeImportRequest(r)
	if err != nil {
		writeImportError(w, err)
		return
	}

	result, err := s.processor.ProcessReport(r.Context(), report)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (s *Server) handleImportZip(w http.ResponseWriter, r *http.Request, save bool) {
	const maxUpload = 32 << 20
	if err := r.ParseMultipartForm(maxUpload); err != nil {
		writeError(w, http.StatusBadRequest, "zip yüklenemedi (max 32 MB)")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "zip dosyası gerekli")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxUpload+1))
	if err != nil {
		writeError(w, http.StatusBadRequest, "zip okunamadı")
		return
	}
	if len(data) > maxUpload {
		writeError(w, http.StatusBadRequest, "zip çok büyük (max 32 MB)")
		return
	}

	projectKey := strings.TrimSpace(r.FormValue("project_key"))
	projectName := strings.TrimSpace(r.FormValue("project_name"))

	report, err := scanner.AnalyzeZipArchive(r.Context(), data, header.Filename, "1.0.0", projectKey, projectName)
	if err != nil {
		writeImportError(w, err)
		return
	}

	if save {
		result, err := s.processor.ProcessReport(r.Context(), report)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, result)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"filename":  report.Archive.ZipName,
		"language":  "archive",
		"archive":   true,
		"file_count": len(report.Archive.Files),
		"files":     report.Archive.Files,
		"issues":    report.Issues,
		"gate":      report.Gate,
		"measures":  report.Measures,
	})
}

func (s *Server) analyzeImportRequest(r *http.Request) (*model.Report, error) {
	var req struct {
		ProjectKey  string `json:"project_key"`
		ProjectName string `json:"project_name"`
		Filename    string `json:"filename"`
		Source      string `json:"source"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, fmt.Errorf("geçersiz istek")
	}
	if strings.TrimSpace(req.Source) == "" {
		return nil, fmt.Errorf("dosya içeriği boş")
	}

	key := strings.TrimSpace(req.ProjectKey)
	name := strings.TrimSpace(req.ProjectName)
	if key == "" {
		key = projectKeyFromFilename(req.Filename)
	}
	if name == "" {
		name = key
	}

	return scanner.AnalyzeSourceFile(r.Context(), req.Source, req.Filename, "1.0.0", key, name)
}

func writeImportError(w http.ResponseWriter, err error) {
	msg := err.Error()
	status := http.StatusInternalServerError
	if msg == "geçersiz istek" || msg == "dosya içeriği boş" {
		status = http.StatusBadRequest
	}
	writeError(w, status, msg)
}

func (s *Server) handleProjectSource(w http.ResponseWriter, r *http.Request) {
	project, err := s.getProjectByKey(w, r)
	if err != nil || project == nil {
		return
	}
	filename, source, err := s.store.GetLatestProjectSource(r.Context(), project.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if source == "" {
		writeJSON(w, http.StatusOK, map[string]any{"available": false})
		return
	}
	if arch, ok := model.ParseArchiveManifest(source); ok {
		active := arch.Files[0]
		writeJSON(w, http.StatusOK, map[string]any{
			"available":   true,
			"archive":     true,
			"filename":    filename,
			"zip_name":    arch.ZipName,
			"files":       arch.Files,
			"source":      active.Text,
			"active_file": active.Filename,
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"available": true,
		"filename":  filename,
		"source":    source,
	})
}

func projectKeyFromFilename(filename string) string {
	base := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filepath.Base(filename)))
	base = strings.ToLower(base)
	var b strings.Builder
	for _, ch := range base {
		if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-' || ch == '_' {
			b.WriteRune(ch)
		} else if ch == ' ' || ch == '.' {
			b.WriteRune('-')
		}
	}
	key := strings.Trim(b.String(), "-")
	if key == "" {
		return "python-dosyasi"
	}
	return key
}

func (s *Server) handleUploadAnalysis(w http.ResponseWriter, r *http.Request) {
	var report model.Report
	if err := json.NewDecoder(r.Body).Decode(&report); err != nil {
		writeError(w, http.StatusBadRequest, "invalid report JSON")
		return
	}

	result, err := s.processor.ProcessReport(r.Context(), &report)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (s *Server) getProjectByKey(w http.ResponseWriter, r *http.Request) (*model.StoredProject, error) {
	key := chi.URLParam(r, "key")
	project, err := s.store.GetProjectByKey(r.Context(), key)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return nil, err
	}
	if project == nil {
		writeError(w, http.StatusNotFound, "project not found")
		return nil, fmt.Errorf("not found")
	}
	return project, nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
