# QualiGuard Mimari Tasarım

SonarQube'dan ilham alınmış, daha hafif ve modüler bir platform.

---

## 1. Tasarım Prensipleri

1. **Scanner-first** — analiz CI'da/lokalde çalışır, sunucu hafif kalır
2. **Single binary deploy** — kurulum tek komut
3. **Plugin/rule pack** — yeni dil = yeni rule pack, core değişmez
4. **Issue continuity** — analizler arası bulgu takibi (fingerprint)
5. **New code focus** — sadece diff'e sıkı gate uygula
6. **Offline capable** — sunucu olmadan da CLI rapor üretsin

---

## 2. Üst Düzey Mimari

```
┌─────────────────────────────────────────────────────────────┐
│                      QualiGuard Server                       │
│                                                              │
│  ┌────────────┐  ┌─────────────┐  ┌───────────────────────┐ │
│  │  Web UI    │  │  REST API   │  │  Auth (JWT/API Key)   │ │
│  │  (React)   │  │  (Go/FastAPI)│  │                       │ │
│  └────────────┘  └─────────────┘  └───────────────────────┘ │
│                                                              │
│  ┌────────────────────────────────────────────────────────┐ │
│  │              Report Processor (Worker)                  │ │
│  │  Issue merge · Metrics · Quality Gate · Notifications  │ │
│  └────────────────────────────────────────────────────────┘ │
│                                                              │
│  ┌────────────────────────────────────────────────────────┐ │
│  │              SQLite / PostgreSQL                        │ │
│  └────────────────────────────────────────────────────────┘ │
└──────────────────────────┬──────────────────────────────────┘
                           │ HTTP (report upload)
┌──────────────────────────┴──────────────────────────────────┐
│                    QualiGuard CLI (Scanner)                  │
│                                                              │
│  ┌──────────┐  ┌──────────────┐  ┌─────────────────────┐  │
│  │ Config   │→ │ Rule Engine  │→ │ Report Generator    │  │
│  │ Loader   │  │ (multi-lang) │  │ (JSON/SARIF)        │  │
│  └──────────┘  └──────────────┘  └─────────────────────┘  │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  Language Parsers: Python · JS/TS · Go · (genişler)  │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘

Opsiyonel:
┌──────────────────┐     ┌──────────────────┐
│  VS Code Ext     │     │  GitHub Action   │
│  (real-time)     │     │  (CI gate)       │
└──────────────────┘     └──────────────────┘
```

---

## 3. Bileşen Detayları

### 3.1 CLI Scanner (`qg scan`)

**Görev:** Kaynak kodu analiz et, rapor üret.

```
qg scan [flags]
  --project-key   Proje kimliği
  --sources       Kaynak dizin (default: .)
  --config        qualiguard.yaml
  --output        Rapor formatı: json | sarif | html
  --upload        Sunucuya gönder (opsiyonel)
  --server-url    Sunucu adresi
  --token         API token
```

**İç modüller:**

| Modül | Sorumluluk |
|-------|------------|
| `config` | YAML/TOML proje config okuma |
| `discovery` | Dosya keşfi, exclusion pattern |
| `parser` | Dil bazlı AST parse |
| `rules` | Kural yükleme ve çalıştırma |
| `metrics` | ncloc, complexity, duplication hesaplama |
| `reporter` | JSON/SARIF/HTML çıktı |
| `uploader` | HTTP POST ile sunucuya gönderme |

### 3.2 Server API

**REST endpoint'leri (MVP):**

```
POST   /api/v1/projects              → Proje oluştur
GET    /api/v1/projects              → Proje listesi
GET    /api/v1/projects/:key         → Proje detay + son analiz

POST   /api/v1/analyses              → Analiz raporu yükle
GET    /api/v1/analyses/:id          → Analiz sonucu
GET    /api/v1/analyses/:id/issues   → Issue listesi

GET    /api/v1/projects/:key/metrics → Metrik geçmişi
GET    /api/v1/projects/:key/gate    → Quality gate durumu

POST   /api/v1/quality-gates         → Gate tanımla
GET    /api/v1/rules                 → Kural listesi
GET    /api/v1/quality-profiles      → Kural profilleri
```

### 3.3 Report Processor (Worker)

Scanner'dan gelen ham raporu işler:

```
Input: AnalysisReport (JSON)
  │
  ├─→ Issue Fingerprint Match
  │     Yeni issue → INSERT
  │     Mevcut issue → UPDATE status
  │     Kaybolan issue → CLOSE (FIXED)
  │
  ├─→ Metrics Aggregation
  │     ncloc, complexity, duplication, debt
  │
  ├─→ Quality Gate Evaluation
  │     Koşulları kontrol et → PASS/FAIL
  │
  └─→ Notification (webhook, PR comment)
```

### 3.4 Web UI

**Sayfalar (MVP):**

| Sayfa | İçerik |
|-------|--------|
| Dashboard | Proje listesi, gate durumu özeti |
| Project Overview | Son analiz metrikleri, rating'ler |
| Issues | Filtrelenebilir issue listesi |
| Code | Dosya bazlı issue görünümü |
| Quality Gate | Gate koşulları ve geçmiş |
| Settings | Proje config, profil, token |

---

## 4. Veri Modeli

### projects
```sql
id, key, name, main_branch, quality_gate_id, quality_profile_id, created_at
```

### analyses
```sql
id, project_id, branch, commit_sha, status, gate_status,
started_at, finished_at, report_path
```

### issues
```sql
id, project_id, analysis_id, rule_key, severity, type,
message, file_path, line, column, effort_minutes,
fingerprint, status, resolution, first_seen_analysis_id,
created_at, updated_at
```

### measures (metrikler)
```sql
id, analysis_id, metric_key, value
-- metric_key: ncloc, complexity, coverage, bugs, ...
```

### quality_gates
```sql
id, name, is_default, conditions (JSON)
```

### rules
```sql
id, key, name, description, severity, type, language, tags, enabled
```

---

## 5. Issue Fingerprint Algoritması

Aynı bulgunun analizler arası takibi kritik:

```
fingerprint = hash(
  rule_key +
  file_path +
  normalized_message +
  anchor_line_context  // ±3 satır snippet hash
)
```

Bu sayede:
- Satır kayması → hâlâ aynı issue
- Kural değişirse → yeni issue
- Dosya taşınırsa → path güncellenir (opsiyonel rename detection)

---

## 6. Rule Engine Tasarımı

### Kural Pack Formatı (YAML)

```yaml
# rules/python/unused-variable.yaml
key: python:unused-variable
name: Unused local variable
severity: MINOR
type: CODE_SMELL
language: python
tags: [unused, dead-code]
effort: 2  # dakika

check:
  type: ast_visitor
  pattern: |
    # FunctionDef içinde Assign → Name hiç kullanılmamış
    Assign(targets=[Name(id=$var)], value=$val)
    where: not used($var, scope=function)
  
message: "Remove unused variable '{{var}}'"
```

### MVP için basit yaklaşım
- Python: `ast` modülü + visitor pattern
- JS/TS: `@typescript-eslint/parser` veya `tree-sitter`
- Go: `go/ast` + `go/parser`

İleri seviye: tree-sitter ile unified parser (tüm diller).

### Harici tool import (kısayol)
MVP'de mevcut linter'ları içe aktar:
```yaml
imports:
  - tool: ruff
    format: sarif
  - tool: eslint
    format: json
```

---

## 7. Rapor Formatları

### Native JSON (QualiGuard format)
```json
{
  "projectKey": "my-app",
  "branch": "main",
  "commit": "abc123",
  "issues": [...],
  "measures": {
    "ncloc": 12500,
    "complexity": 890,
    "coverage": 72.5
  },
  "duplications": [...]
}
```

### SARIF (Standart — GitHub/GitLab uyumlu)
GitHub Advanced Security, Azure DevOps SARIF okur. CI entegrasyonu için ideal.

---

## 8. Deployment Modelleri

### Mod 1: Tek makine (MVP)
```
docker compose up
  - qualiguard-server (API + UI)
  - qualiguard-worker (report processor)
  - postgres (opsiyonel, default sqlite)
```

### Mod 2: CI-only (sunucusuz)
```
qg scan --output sarif > report.sarif
# GitHub Actions SARIF upload
```

### Mod 3: Self-hosted enterprise
```
Server cluster + PostgreSQL + Redis queue
```

---

## 9. SonarQube vs QualiGuard Karşılaştırma

| Özellik | SonarQube | QualiGuard |
|---------|-----------|------------|
| Kurulum | Java + DB + plugin | Single binary / Docker |
| Min. RAM | ~2 GB | ~128 MB |
| Dil ekleme | Java plugin JAR | YAML rule pack |
| Offline scan | Evet | Evet |
| SARIF export | Sınırlı | Birincil format |
| Issue tracking | Evet | Evet |
| PR decoration | Evet | Faz 3 |
| IDE plugin | SonarLint | Faz 4 |
| AI code rules | Enterprise | Faz 5 |

---

## 10. Güvenlik

- API token bazlı auth (Bearer)
- Proje bazlı token scope
- Scanner token: sadece upload yetkisi
- Admin token: full access
- Rate limiting on upload endpoint
- Input validation on report size (max 50MB)

Sonraki dosya: `03-analiz-akisi.md`
