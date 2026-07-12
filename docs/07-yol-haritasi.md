# Yol Haritası

QualiGuard MVP'den tam ürüne giden plan.

---

## Tamamlanan (staj sürümü)

| Faz | Durum |
|-----|-------|
| Faz 1 — CLI Scanner | ✅ |
| Faz 2 — Server + API + SQLite | ✅ |
| Faz 3 — Web Dashboard + Kalite Kapısı | ✅ |
| Faz 4 — CI/CD + PR yorumu | ✅ |
| AI-1 — Ollama sohbet + sorun açıklaması | ✅ |

**Panel özellikleri:** zip yükleme, canlı analiz, proje silme (tekli/toplu), JSON/HTML/PDF rapor, YZ Sohbet, **VS Code eklentisi (MVP)**.

**Sonraki büyük adımlar:** özel kural editörü, PostgreSQL (çok kullanıcılı kurulum), eklenti workspace taraması.

---

## Faz Özeti

```
Faz 1 ─── CLI Scanner (Python)           [2-3 hafta]
Faz 2 ─── Server + API + DB              [3-4 hafta]
Faz 3 ─── Web UI + Quality Gate          [3-4 hafta]
Faz 4 ─── CI/CD + PR Decoration          [2 hafta]
Faz 5 ─── Multi-language + IDE Plugin    [4+ hafta]
```

---

## Faz 1: CLI Scanner (MVP Core)

**Hedef:** `qg scan` komutu Python projelerini tarayıp JSON/SARIF rapor üretsin.

### Deliverable'lar
- [ ] Go proje iskeleti (`cmd/qg`, `internal/scanner`)
- [ ] `qualiguard.yaml` config parser
- [ ] Dosya keşfi + exclusion
- [ ] Python AST parser (subprocess veya go-python)
- [ ] 10 temel Python kuralı:
  - unused-variable
  - unused-import
  - bare-except
  - empty-except
  - complex-function (>10)
  - long-function (>50 satır)
  - sql-injection (pattern)
  - hardcoded-password (pattern)
  - eval-usage
  - syntax-error
- [ ] ncloc + cyclomatic complexity metrikleri
- [ ] JSON rapor çıktısı
- [ ] SARIF rapor çıktısı
- [ ] `--output`, `--format` flag'leri
- [ ] Exit code: 0 (ok), 2 (scan error)

### Test
```bash
qg scan --sources ./my-python-project --output report.json
# report.json'da issues + measures var
```

### Başarı kriteri
- 1000 dosyalık Python projesini < 10 saniyede tarar
- SARIF çıktısı GitHub'da görüntülenebilir

---

## Faz 2: Server + API + Issue Tracking

**Hedef:** Analiz raporlarını sunucuya yükle, geçmiş takibi yap.

### Deliverable'lar
- [ ] `qg-server` binary (Go HTTP server)
- [ ] SQLite DB + migration
- [ ] REST API:
  - POST /api/v1/projects
  - POST /api/v1/analyses (report upload)
  - GET /api/v1/projects/:key/issues
  - GET /api/v1/projects/:key/measures
- [ ] Report processor (issue fingerprint merge)
- [ ] Issue yaşam döngüsü (OPEN → CLOSED)
- [ ] `qg scan --upload --server-url --token`
- [ ] API key auth
- [ ] Docker Compose (server + sqlite)

### Test
```bash
# Terminal 1
qg-server --port 9000

# Terminal 2
qg scan --upload --server-url http://localhost:9000 --token xxx
curl http://localhost:9000/api/v1/projects/my-app/issues
```

### Başarı kriteri
- 2 ardışık analizde issue merge doğru çalışır
- Düzeltilen issue CLOSED olur

---

## Faz 3: Web UI + Quality Gate

**Hedef:** Dashboard, issue görüntüleme, quality gate.

### Deliverable'lar
- [ ] React frontend (Tailwind)
- [ ] Dashboard: proje listesi + gate durumu
- [ ] Project overview: metrik kartları + rating
- [ ] Issue listesi: filtre (severity, type, status)
- [ ] Quality gate evaluator
- [ ] Built-in "QualiGuard Way" gate
- [ ] `--fail-on-gate` CLI flag
- [ ] Gate geçmişi grafiği
- [ ] PostgreSQL desteği (production)

### Test
```bash
docker compose up
# http://localhost:9000 → UI
# Proje oluştur, scan upload et, gate sonucunu gör
```

### Başarı kriteri
- Gate FAIL → CLI exit code 1
- UI'da issue filtreleme çalışır

---

## Faz 4: CI/CD Entegrasyonu

**Hedef:** GitHub Actions, GitLab CI, PR decoration.

### Deliverable'lar
- [ ] GitHub Action (`qualiguard/scan-action`)
- [ ] GitLab CI template
- [x] PR/MR decoration (yorum + check)
- [ ] New code analysis (branch diff)
- [ ] SARIF → GitHub Security tab upload
- [ ] Webhook notifications (Slack, Discord)
- [ ] `--baseline` flag (mevcut issue'ları ignore)

### Test
```yaml
# .github/workflows/qualiguard.yml
- uses: qualiguard/scan-action@v1
  with:
    server-url: ${{ secrets.QG_URL }}
    token: ${{ secrets.QG_TOKEN }}
    fail-on-gate: true
```

### Başarı kriteri
- PR'da gate sonucu yorum olarak görünür
- Gate fail → PR merge engellenir

---

## Faz 5: Multi-Language + IDE

**Hedef:** JS/TS/Go desteği, VS Code eklentisi.

### Deliverable'lar
- [ ] JavaScript/TypeScript kuralları (tree-sitter)
- [ ] Go kuralları (go/ast)
- [ ] Harici tool import (ruff, eslint, bandit) — **ESLint (JS) + Ruff (Python) eklendi**
- [x] VS Code extension (kaydet/açılış tarama, diagnostics) — `extension/qualiguard`
- [x] Incremental scan (sadece değişen dosyalar)
- [ ] Duplication detection
- [ ] Coverage import (coverage.py, istanbul)
- [ ] Custom rule yazma API
- [ ] Quality profile editor (UI)

---

## Öncelik Matrisi

```
                Etki
                ↑
    Faz 3 UI  │  Faz 1 CLI ★
    Faz 4 CI  │  Faz 2 Server ★
              │
    Faz 5 IDE │  Faz 5 Multi-lang
              └──────────────────→ Efor
              
★ = İlk yapılacak
```

---

## Milestone Takvimi (tahmini)

| Milestone | Hedef tarih | Durum |
|-----------|-------------|-------|
| Faz 1 — CLI Scanner | +3 hafta | ⏳ Bekliyor |
| Faz 2 — Server + API | +7 hafta | ⏳ Bekliyor |
| Faz 3 — Web UI + Gate | +11 hafta | ⏳ Bekliyor |
| Faz 4 — CI/CD | +13 hafta | ⏳ Bekliyor |
| Faz 5 — Multi-lang + IDE | +17 hafta | ⏳ Bekliyor |

---

## Hemen Şimdi: Faz 1 İlk Adımlar

```bash
# 1. Go modül oluştur
mkdir -p ~/Desktop/QualiGuard/src
cd ~/Desktop/QualiGuard/src
go mod init github.com/qualiguard/qualiguard

# 2. CLI iskelet
mkdir -p cmd/qg internal/{scanner,rules,parser,reporter,config}

# 3. İlk komut
# qg scan --sources . --format json
```

### İlk kodlanacak dosyalar
1. `cmd/qg/main.go` — CLI entry point
2. `internal/config/config.go` — YAML config
3. `internal/scanner/scanner.go` — orchestrator
4. `internal/parser/python.go` — Python AST
5. `internal/rules/engine.go` — kural runner
6. `rules/python/*.yaml` — kural tanımları
7. `internal/reporter/json.go` — JSON output

---

## Açık Kararlar

| Karar | Seçenekler | Tercih |
|-------|-----------|--------|
| Python parse | subprocess vs go-python | subprocess (MVP) |
| Proje adı | QualiGuard vs başka | QualiGuard |
| Lisans | MIT vs Apache 2.0 | MIT |
| Monorepo | Tek repo vs ayrı | Tek monorepo |
| Auth | API key vs OAuth | API key (MVP) |

Bu kararlar geliştirme sırasında kesinleştirilecek.
