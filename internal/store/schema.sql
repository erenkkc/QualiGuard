CREATE TABLE IF NOT EXISTS projects (
    id TEXT PRIMARY KEY,
    key TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS analyses (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL,
    branch TEXT,
    commit_sha TEXT,
    status TEXT NOT NULL,
    scanner_version TEXT,
    duration_ms INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL,
    FOREIGN KEY (project_id) REFERENCES projects(id)
);

CREATE TABLE IF NOT EXISTS issues (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL,
    analysis_id TEXT NOT NULL,
    rule_key TEXT NOT NULL,
    severity TEXT NOT NULL,
    type TEXT NOT NULL,
    message TEXT NOT NULL,
    file_path TEXT NOT NULL,
    line INTEGER NOT NULL,
    column_num INTEGER NOT NULL DEFAULT 0,
    effort_minutes INTEGER NOT NULL DEFAULT 0,
    fingerprint TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'OPEN',
    resolution TEXT,
    snippet TEXT NOT NULL DEFAULT '',
    fix_suggestion TEXT NOT NULL DEFAULT '',
    first_seen_analysis_id TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    FOREIGN KEY (project_id) REFERENCES projects(id),
    FOREIGN KEY (analysis_id) REFERENCES analyses(id),
    UNIQUE(project_id, fingerprint)
);

CREATE TABLE IF NOT EXISTS measures (
    id TEXT PRIMARY KEY,
    analysis_id TEXT NOT NULL,
    metric_key TEXT NOT NULL,
    value REAL NOT NULL,
    FOREIGN KEY (analysis_id) REFERENCES analyses(id)
);

CREATE TABLE IF NOT EXISTS api_tokens (
    id TEXT PRIMARY KEY,
    token TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS user_sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    token TEXT NOT NULL UNIQUE,
    expires_at TEXT NOT NULL,
    created_at TEXT NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_issues_project_status ON issues(project_id, status);
CREATE INDEX IF NOT EXISTS idx_issues_fingerprint ON issues(project_id, fingerprint);
CREATE INDEX IF NOT EXISTS idx_analyses_project ON analyses(project_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_sessions_token ON user_sessions(token);
