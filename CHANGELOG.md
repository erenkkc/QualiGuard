# Changelog

Tüm proje güncellemeleri burada takip edilir.

## [1.0.0] - 2026-07-08

### Eklendi
- Kalite Kapısı (QualiGuard Yolu) — CLI, sunucu, dashboard
- AI açıklama katmanı (Türkçe şablon + opsiyonel OpenAI LLM)
- Canlı Analiz: kod satırı vurgusu, Kalite Kapısı paneli
- Proje sayfası: önem/tür filtreleri, kapı geçmişi
- CLI: `--fail-on-gate`, `--baseline`, `--save-baseline`, `--no-ai`
- CI: GitHub Actions workflow, GitLab CI şablonu, SARIF upload
- Docker + docker-compose
- Kısmi parse (syntax hatası olsa da geçerli kod taranır)

## [0.5.0] - 2026-07-08

### Eklendi
- Faz 3 başlangıcı: Web dashboard (embedded HTML/CSS/JS)
- Dashboard: proje listesi, metrik kartları, issue tablosu
- Severity filtreleme
- API: `/api/v1/projects/overview`, `/api/v1/projects/{key}/overview`
- Masaüstü kısayolu: `QualiGuard-Dashboard.bat` (sunucu + tarayıcı)

## [0.4.0] - 2026-07-08

### Eklendi
- Faz 2.1: `qg-server` HTTP sunucusu
- SQLite veritabanı + issue merge (OPEN/CLOSED)
- REST API: projects, analyses upload, issues, measures
- API token auth
- `qg scan --upload --server-url --token`
- Masaüstü kısayolları: QualiGuard-Server.bat, QualiGuard-Scan.bat, QualiGuard-Upload.bat

## [0.3.0] - 2026-07-08

### Eklendi
- Faz 1 CLI scanner iskeleti (Go + Cobra)
- Python AST analyzer (`internal/parser/python_analyzer.py`)
- 10 Python kuralı (unused import/variable, bare/empty except, complexity, SQL injection, secrets, eval)
- JSON ve SARIF rapor çıktısı
- Dosya keşfi + exclusion pattern desteği
- Issue fingerprint sistemi
- Metrik hesaplama (ncloc, complexity, issue counts)
- Test fixtures (`testdata/sample_project/`)
- Örnek `qualiguard.yaml` config

## [0.2.0] - 2026-07-08

### Eklendi
- AI entegrasyon stratejisi (`docs/08-yapay-zeka-entegrasyonu.md`)
- 7 AI modülü tanımı: Explain, Triage, Semantic Security, Auto-Fix, NL Rules, PR Summary, RAG Chat
- LLM provider abstraction tasarımı (OpenAI, Claude, Ollama)
- Maliyet kontrolü ve gizlilik kuralları
- AI mimari diyagramları (`design/ai-mimari-diyagram.md`)

## [0.1.0] - 2026-07-08

### Eklendi
- Proje klasör yapısı oluşturuldu (`Desktop/QualiGuard`)
- SonarQube derin analizi (`docs/01-sonarqube-analizi.md`)
- Platform mimari tasarımı (`docs/02-mimari-tasarim.md`)
- Analiz akışı dokümantasyonu (`docs/03-analiz-akisi.md`)
- Kurallar ve metrikler rehberi (`docs/04-kurallar-ve-metrikler.md`)
- Quality gate tasarımı (`docs/05-quality-gates.md`)
- Tech stack önerileri (`docs/06-tech-stack.md`)
- Yol haritası (`docs/07-yol-haritasi.md`)
- Mimari diyagramlar (`design/mimari-diyagram.md`)
