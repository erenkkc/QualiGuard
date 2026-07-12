# Tech Stack Önerileri

QualiGuard için teknoloji seçimleri ve gerekçeleri.

---

## 1. Genel Yaklaşım

| Kriter | Tercih |
|--------|--------|
| Performans | Native/compiled (Go veya Rust) |
| Kurulum kolaylığı | Single binary |
| Cross-platform | Windows + Linux + macOS |
| Geliştirme hızı | MVP hızlı, sonra optimize |
| Ekosistem | AST/parser kütüphaneleri zengin |

---

## 2. Önerilen Stack

### Seçenek A: Go (Önerilen — MVP)

```
CLI Scanner:  Go
Server API:   Go (chi veya echo router)
Worker:       Go (aynı binary, subcommand)
Web UI:       React + TypeScript + Tailwind
Database:     SQLite (dev) → PostgreSQL (prod)
Queue:        Go channels (MVP) → Redis (scale)
```

**Neden Go?**
- Single binary cross-compile
- Hızlı derleme, hızlı çalışma
- `go/ast` built-in (Go analizi)
- Python AST için `go-python` veya exec wrapper
- Tree-sitter Go binding mevcut
- Docker image ~15MB

### Seçenek B: Rust (Performans odaklı)

```
CLI + Server: Rust
Web UI:       React + TypeScript
Database:     SQLx + PostgreSQL
```

**Neden Rust?**
- En yüksek performans
- Tree-sitter native Rust
- Memory safety
- Dezavantaj: Geliştirme süresi daha uzun

### Seçenek C: Python (Hızlı prototip)

```
CLI:     Python (click/typer)
Server:  FastAPI
Worker:  Celery + Redis
Web UI:  React
```

**Neden Python?**
- AST modülü built-in
- Hızlı kural geliştirme
- Dezavantaj: Yavaş, dağıtım zor

---

## 3. Karar: Go + React

MVP ve production için **Go backend + React frontend**.

```
qualiguard/
├── cmd/
│   ├── qg/           # CLI scanner binary
│   └── qg-server/    # Server binary
├── internal/
│   ├── scanner/      # Analiz motoru
│   ├── rules/        # Kural engine
│   ├── parser/       # Dil parser'ları
│   ├── reporter/     # Rapor oluşturucu
│   ├── server/       # HTTP API
│   ├── processor/    # Report worker
│   ├── gate/         # Quality gate
│   └── store/        # DB layer
├── rules/            # YAML kural tanımları
│   ├── python/
│   └── javascript/
├── web/              # React frontend
│   ├── src/
│   └── package.json
├── docker-compose.yml
├── go.mod
└── Makefile
```

---

## 4. Parser Stratejisi

### Katman 1: Native parser (hızlı MVP)
| Dil | Parser |
|-----|--------|
| Python | `go-python` veya Python subprocess + JSON |
| Go | `go/parser` + `go/ast` (built-in) |
| JS/TS | Tree-sitter (`tree-sitter-javascript`) |

### Katman 2: Tree-sitter (unified)
Tüm diller için tek parser framework:
```
tree-sitter-python
tree-sitter-javascript
tree-sitter-typescript
tree-sitter-go
tree-sitter-rust
tree-sitter-java
```

Avantaj: Error-tolerant parsing, hızlı, çok dilli.

### Katman 3: Harici tool import
```
ruff (Python) → JSON
eslint (JS) → JSON
golangci-lint (Go) → JSON
semgrep (multi) → SARIF
```

---

## 5. Veritabanı

### MVP: SQLite
```go
// embed DB, zero config
import _ "github.com/mattn/go-sqlite3"
```
- Kurulum gerektirmez
- Tek dosya
- < 100 proje için yeterli

### Production: PostgreSQL
```yaml
# docker-compose.yml
services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: qualiguard
```

Migration: `golang-migrate` veya `goose`

---

## 6. Web UI Stack

```
React 19 + TypeScript
Tailwind CSS 4
Recharts (metrik grafikleri)
TanStack Query (API cache)
React Router (sayfa yönlendirme)
```

### UI sayfa planı
| Route | Bileşen |
|-------|---------|
| `/` | Dashboard — proje listesi |
| `/projects/:key` | Project overview |
| `/projects/:key/issues` | Issue listesi + filtre |
| `/projects/:key/measures` | Metrik grafikleri |
| `/projects/:key/gate` | Quality gate durumu |
| `/admin/gates` | Gate yönetimi |
| `/admin/rules` | Kural browser |
| `/admin/profiles` | Quality profile yönetimi |

---

## 7. API Tasarımı

```
Framework: chi router
Auth: JWT + API key
Docs: OpenAPI 3.0 (swaggo/swag)
Validation: go-playground/validator
```

---

## 8. Dağıtım

### CLI kurulum
```bash
# Linux/macOS
curl -sSL https://get.qualiguard.dev | sh

# Windows
winget install QualiGuard.CLI

# Go install
go install github.com/qualiguard/qg@latest
```

### Server kurulum
```bash
# Docker (tek komut)
docker run -d -p 9000:9000 qualiguard/server

# Docker Compose
docker compose up -d
```

### Binary boyut hedefleri
| Binary | Hedef |
|--------|-------|
| qg (CLI) | < 30 MB |
| qg-server | < 40 MB |
| Docker image | < 80 MB |

---

## 9. Test Stratejisi

```
Unit tests:     Go testing + testify
Rule tests:     Her kural için good/bad code fixture
Integration:    testcontainers (PostgreSQL)
E2E:            Playwright (Web UI)
Benchmark:      Go benchmark (scan performance)
```

---

## 10. Bağımlılıklar (Go MVP)

```
github.com/spf13/cobra          # CLI framework
github.com/go-chi/chi/v5        # HTTP router
github.com/mattn/go-sqlite3     # SQLite
github.com/jackc/pgx/v5         # PostgreSQL
github.com/smacker/go-tree-sitter # Tree-sitter
gopkg.in/yaml.v3                # YAML config
github.com/google/uuid          # UUID
github.com/rs/zerolog           # Logging
```

Sonraki dosya: `07-yol-haritasi.md`
