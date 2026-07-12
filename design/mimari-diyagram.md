# Mimari Diyagramlar

QualiGuard sistem diyagramları (Mermaid).

---

## 1. Genel Sistem Mimarisi

```mermaid
graph TB
    subgraph "Developer / CI"
        DEV[Developer]
        CI[GitHub Actions / GitLab CI]
    end

    subgraph "QualiGuard CLI"
        SCAN[qg scan]
        CONFIG[qualiguard.yaml]
        PARSER[Language Parsers]
        RULES[Rule Engine]
        METRICS[Metrics Calculator]
        REPORT[Report Generator]
    end

    subgraph "QualiGuard Server"
        API[REST API]
        WORKER[Report Processor]
        GATE[Quality Gate Evaluator]
        UI[Web UI - React]
        DB[(SQLite / PostgreSQL)]
    end

    DEV --> SCAN
    CI --> SCAN
    CONFIG --> SCAN
    SCAN --> PARSER
    PARSER --> RULES
    RULES --> METRICS
    METRICS --> REPORT

    REPORT -->|JSON/SARIF| DEV
    REPORT -->|HTTP POST| API
    API --> WORKER
    WORKER --> DB
    WORKER --> GATE
    GATE --> DB
    UI --> API
    API --> DB
```

---

## 2. Analiz Pipeline

```mermaid
sequenceDiagram
    participant CLI as qg scan
    participant DISC as File Discovery
    participant PAR as Parser
    participant ENG as Rule Engine
    participant MET as Metrics
    participant REP as Reporter
    participant SRV as Server
    participant WRK as Worker
    participant GATE as Quality Gate

    CLI->>DISC: sources + exclusions
    DISC-->>CLI: file list

    loop Her dosya
        CLI->>PAR: parse(file)
        PAR-->>CLI: AST
        CLI->>ENG: check(rules, AST)
        ENG-->>CLI: issues[]
    end

    CLI->>MET: calculate(all files)
    MET-->>CLI: measures

    CLI->>REP: generate(issues, measures)
    REP-->>CLI: AnalysisReport

    opt --upload flag
        CLI->>SRV: POST /api/v1/analyses
        SRV->>WRK: queue report
        WRK->>WRK: issue fingerprint merge
        WRK->>GATE: evaluate conditions
        GATE-->>SRV: PASS/FAIL
        SRV-->>CLI: gate result
    end
```

---

## 3. Issue Yaşam Döngüsü

```mermaid
stateDiagram-v2
    [*] --> OPEN: Yeni analiz buldu

    OPEN --> CONFIRMED: Manuel onayla
    OPEN --> CLOSED: Kod düzeltildi (FIXED)
    OPEN --> CLOSED: False positive (FALSE_POSITIVE)
    OPEN --> CLOSED: Won't fix (WONTFIX)

    CONFIRMED --> CLOSED: Düzeltildi
    CONFIRMED --> REOPENED: Tekrar ortaya çıktı

    CLOSED --> REOPENED: Sonraki analiz tekrar buldu
    REOPENED --> OPEN: Otomatik

    CLOSED --> [*]
```

---

## 4. Quality Gate Akışı

```mermaid
flowchart TD
    A[Analiz Tamamlandı] --> B[Metrikler Hesaplandı]
    B --> C{New Code mu?}
    C -->|Evet| D[Diff metrikleri]
    C -->|Hayır| E[Overall metrikleri]
    D --> F[Gate koşullarını kontrol et]
    E --> F
    F --> G{ERROR koşul ihlali?}
    G -->|Evet| H[❌ FAIL]
    G -->|Hayır| I{WARN koşul ihlali?}
    I -->|Evet| J[⚠️ WARN]
    I -->|Hayır| K[✅ PASS]
    H --> L[CI exit code 1]
    J --> M[CI exit code 0 + warning]
    K --> N[CI exit code 0]
```

---

## 5. Veri Modeli (ER)

```mermaid
erDiagram
    PROJECT ||--o{ ANALYSIS : has
    PROJECT ||--o{ ISSUE : has
    PROJECT }o--|| QUALITY_GATE : uses
    PROJECT }o--|| QUALITY_PROFILE : uses
    ANALYSIS ||--o{ MEASURE : contains
    ANALYSIS ||--o{ ISSUE : found
    ANALYSIS ||--|| GATE_RESULT : produces
    QUALITY_GATE ||--o{ GATE_CONDITION : has
    QUALITY_PROFILE ||--o{ PROFILE_RULE : contains
    RULE ||--o{ PROFILE_RULE : referenced

    PROJECT {
        uuid id PK
        string key UK
        string name
        string main_branch
        timestamp created_at
    }

    ANALYSIS {
        uuid id PK
        uuid project_id FK
        string branch
        string commit_sha
        string status
        timestamp started_at
        timestamp finished_at
    }

    ISSUE {
        uuid id PK
        uuid project_id FK
        uuid analysis_id FK
        string rule_key
        string severity
        string type
        string file_path
        int line
        string fingerprint
        string status
        string resolution
    }

    MEASURE {
        uuid id PK
        uuid analysis_id FK
        string metric_key
        float value
    }

    QUALITY_GATE {
        uuid id PK
        string name
        bool is_default
    }

    GATE_CONDITION {
        uuid id PK
        uuid gate_id FK
        string metric
        string operator
        float threshold
        string level
    }
```

---

## 6. Deployment Mimarisi

```mermaid
graph LR
    subgraph "CI Pipeline"
        GH[GitHub Actions]
        GL[GitLab CI]
        JEN[Jenkins]
    end

    subgraph "QualiGuard Server (Docker)"
        NGINX[Nginx / Caddy]
        API[qg-server :9000]
        WORKER[Report Worker]
        DB[(PostgreSQL)]
    end

    subgraph "Developer Machine"
        CLI[qg CLI]
        IDE[VS Code Extension]
    end

    GH -->|qg scan --upload| NGINX
    GL -->|qg scan --upload| NGINX
    JEN -->|qg scan --upload| NGINX
    CLI -->|qg scan --upload| NGINX
    IDE -->|connected mode| NGINX

    NGINX --> API
    API --> WORKER
    API --> DB
    WORKER --> DB
```

---

## 7. Rule Engine İç Yapısı

```mermaid
graph TD
    subgraph "Rule Engine"
        LOADER[Rule Loader]
        PROFILE[Quality Profile]
        REGISTRY[Rule Registry]

        subgraph "Detectors"
            AST[AST Visitor]
            REGEX[Regex Matcher]
            SEM[Semantic Analyzer]
            EXT[External Tool Import]
        end

        COLLECTOR[Issue Collector]
    end

    YAML[rules/*.yaml] --> LOADER
    LOADER --> REGISTRY
    PROFILE --> REGISTRY

    REGISTRY --> AST
    REGISTRY --> REGEX
    REGISTRY --> SEM
    REGISTRY --> EXT

    AST --> COLLECTOR
    REGEX --> COLLECTOR
    SEM --> COLLECTOR
    EXT --> COLLECTOR

    COLLECTOR --> ISSUES[Issue List]
```

---

## 8. Faz Timeline

```mermaid
gantt
    title QualiGuard Yol Haritası
    dateFormat YYYY-MM-DD
    section Faz 1
    CLI Scanner           :f1, 2026-07-08, 21d
    section Faz 2
    Server + API          :f2, after f1, 28d
    section Faz 3
    Web UI + Gate         :f3, after f2, 28d
    section Faz 4
    CI/CD Integration     :f4, after f3, 14d
    section Faz 5
    Multi-lang + IDE      :f5, after f4, 28d
```
