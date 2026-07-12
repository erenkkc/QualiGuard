package model

import "time"

const SchemaVersion = "1.0"

type Severity string

const (
	SeverityBlocker  Severity = "BLOCKER"
	SeverityCritical Severity = "CRITICAL"
	SeverityMajor    Severity = "MAJOR"
	SeverityMinor    Severity = "MINOR"
	SeverityInfo     Severity = "INFO"
)

type IssueType string

const (
	TypeBug             IssueType = "BUG"
	TypeVulnerability   IssueType = "VULNERABILITY"
	TypeCodeSmell       IssueType = "CODE_SMELL"
	TypeSecurityHotspot IssueType = "SECURITY_HOTSPOT"
)

type IssueStatus string

const (
	StatusOpen     IssueStatus = "OPEN"
	StatusClosed   IssueStatus = "CLOSED"
	StatusReopened IssueStatus = "REOPENED"
)

type Resolution string

const (
	ResolutionFixed         Resolution = "FIXED"
	ResolutionFalsePositive Resolution = "FALSE_POSITIVE"
	ResolutionWontFix       Resolution = "WONTFIX"
)

type Project struct {
	Key     string `json:"key" yaml:"key"`
	Name    string `json:"name,omitempty" yaml:"name"`
	Version string `json:"version,omitempty" yaml:"version"`
}

type AnalysisMeta struct {
	ID           string    `json:"id,omitempty"`
	Branch       string    `json:"branch,omitempty"`
	Commit       string    `json:"commit,omitempty"`
	Timestamp    time.Time `json:"timestamp"`
	DurationMS   int64     `json:"duration_ms"`
	Incremental  bool      `json:"incremental,omitempty"`
	BaseRef      string    `json:"base_ref,omitempty"`
	ScannedFiles int       `json:"scanned_files,omitempty"`
}

type FileSource struct {
	Filename string `json:"filename"`
	Text     string `json:"text"`
	Language string `json:"language,omitempty"`
}

type Issue struct {
	ID            string      `json:"id,omitempty"`
	RuleKey       string      `json:"rule_key"`
	Severity      Severity    `json:"severity"`
	Type          IssueType   `json:"type"`
	Message       string      `json:"message"`
	File          string      `json:"file"`
	Line          int         `json:"line"`
	Column        int         `json:"column,omitempty"`
	EffortMin     int         `json:"effort_minutes,omitempty"`
	Fingerprint   string      `json:"fingerprint"`
	Status        IssueStatus `json:"status,omitempty"`
	Resolution    Resolution  `json:"resolution,omitempty"`
	Snippet       string      `json:"snippet,omitempty"`
	FixSuggestion string      `json:"fix_suggestion,omitempty"`
	AIExplanation *AIExplanation `json:"ai_explanation,omitempty"`
}

type AIExplanation struct {
	SummaryTR string `json:"summary_tr"`
	RiskTR    string `json:"risk_tr"`
	ExampleTR string `json:"example_tr,omitempty"`
	Source    string `json:"source"`
	Provider  string `json:"provider,omitempty"`
}

type Measures struct {
	Ncloc               int `json:"ncloc"`
	Files               int `json:"files"`
	Complexity          int `json:"complexity"`
	CognitiveComplexity int `json:"cognitive_complexity,omitempty"`
	Bugs                int `json:"bugs"`
	Vulnerabilities     int `json:"vulnerabilities"`
	CodeSmells          int `json:"code_smells"`
	SecurityHotspots    int `json:"security_hotspots"`
}

type Report struct {
	SchemaVersion  string       `json:"schema_version"`
	ScannerVersion string       `json:"scanner_version"`
	Project        Project      `json:"project"`
	Analysis       AnalysisMeta `json:"analysis"`
	Source         *FileSource    `json:"source,omitempty"`
	Archive        *ArchiveSource `json:"archive,omitempty"`
	Issues         []Issue        `json:"issues"`
	Measures       Measures     `json:"measures"`
	Gate           *GateResult  `json:"gate,omitempty"`
}

type StoredProject struct {
	ID        string    `json:"id"`
	Key       string    `json:"key"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type AnalysisHistoryEntry struct {
	AnalysisID     string `json:"analysis_id"`
	ProjectKey     string `json:"project_key"`
	ProjectName    string `json:"project_name"`
	GateStatus     string `json:"gate_status"`
	GateStatusTR   string `json:"gate_status_tr"`
	IssuesFound    int    `json:"issues_found"`
	IssuesNew      int    `json:"issues_new"`
	IssuesOpen     int    `json:"issues_open"`
	IssuesClosed   int    `json:"issues_closed"`
	ScannerVersion string `json:"scanner_version,omitempty"`
	CreatedAt      string `json:"created_at"`
}

type GateHistoryEntry struct {
	AnalysisID string `json:"analysis_id"`
	GateStatus string `json:"gate_status"`
	GateStatusTR string `json:"gate_status_tr"`
	OpenIssues int    `json:"open_issues"`
	CreatedAt  string `json:"created_at"`
}

type UploadResult struct {
	AnalysisID   string     `json:"analysis_id"`
	ProjectKey   string     `json:"project_key"`
	IssuesFound  int        `json:"issues_found"`
	IssuesNew    int        `json:"issues_new"`
	IssuesOpen   int        `json:"issues_open"`
	IssuesClosed int        `json:"issues_closed"`
	Gate         GateResult `json:"gate"`
}

type GateResult struct {
	Name       string            `json:"name"`
	NameTR     string            `json:"name_tr"`
	Status     string            `json:"status"`
	StatusTR   string            `json:"status_tr"`
	Conditions []GateCondition   `json:"conditions"`
}

type GateCondition struct {
	Metric    string  `json:"metric"`
	LabelTR   string  `json:"label_tr"`
	Operator  string  `json:"operator"`
	Threshold float64 `json:"threshold"`
	Actual    float64 `json:"actual"`
	Passed    bool    `json:"passed"`
	Level     string  `json:"level"`
}

type ProjectOverview struct {
	StoredProject
	OpenIssues      int                `json:"open_issues"`
	Bugs            int                `json:"bugs"`
	Vulnerabilities int                `json:"vulnerabilities"`
	CodeSmells      int                `json:"code_smells"`
	Measures        map[string]float64 `json:"measures"`
	LastAnalysisAt  string             `json:"last_analysis_at,omitempty"`
	Gate            *GateResult        `json:"gate,omitempty"`
}

type FileAnalysis struct {
	File         string         `json:"file"`
	ParseError   *ParseError    `json:"parse_error,omitempty"`
	Ncloc        int            `json:"ncloc"`
	Imports      []ImportInfo   `json:"imports"`
	Functions    []FunctionInfo `json:"functions"`
	Assignments  []AssignInfo   `json:"assignments"`
	ExceptBlocks []ExceptInfo   `json:"except_blocks"`
	Calls        []CallInfo     `json:"calls"`
	Strings      []StringInfo   `json:"strings"`
	Secrets      []SecretInfo   `json:"secrets"`
}

type ParseError struct {
	Message string `json:"message"`
	Line    int    `json:"line"`
	Column  int    `json:"column"`
}

type ImportInfo struct {
	Name  string `json:"name"`
	Alias string `json:"alias,omitempty"`
	Line  int    `json:"line"`
	Used  bool   `json:"used"`
}

type FunctionInfo struct {
	Name       string `json:"name"`
	Line       int    `json:"line"`
	EndLine    int    `json:"end_line"`
	Complexity int    `json:"complexity"`
	ParamCount int    `json:"param_count"`
}

type AssignInfo struct {
	Name  string `json:"name"`
	Line  int    `json:"line"`
	Used  bool   `json:"used"`
	Scope string `json:"scope"`
}

type ExceptInfo struct {
	Line  int  `json:"line"`
	Bare  bool `json:"bare"`
	Empty bool `json:"empty"`
}

type CallInfo struct {
	Func         string `json:"func"`
	Line         int    `json:"line"`
	HasUserInput bool   `json:"has_user_input,omitempty"`
	IsFString    bool   `json:"is_fstring,omitempty"`
	DynamicSQL   bool   `json:"dynamic_sql,omitempty"`
	VariableArg  bool   `json:"variable_arg,omitempty"`
}

type StringInfo struct {
	Value string `json:"value"`
	Line  int    `json:"line"`
	Kind  string `json:"kind"`
}

type SecretInfo struct {
	Name string `json:"name"`
	Line int    `json:"line"`
}
