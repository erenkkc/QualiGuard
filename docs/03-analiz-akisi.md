# Analiz Akışı

QualiGuard'da bir taramanın baştan sona nasıl işlediği.

---

## 1. Tam Akış Diyagramı

```
                    ┌─────────────────┐
                    │  qualiguard.yaml │
                    │  (proje config)  │
                    └────────┬────────┘
                             │
    Developer/CI             ▼
         │          ┌─────────────────┐
         │          │   qg scan       │
         └─────────→│   CLI Scanner   │
                    └────────┬────────┘
                             │
              ┌──────────────┼──────────────┐
              ▼              ▼              ▼
        ┌──────────┐  ┌──────────┐  ┌──────────────┐
        │  File    │  │  Rule    │  │  Metrics     │
        │ Discovery│  │  Engine  │  │  Calculator  │
        └────┬─────┘  └────┬─────┘  └──────┬───────┘
             │              │              │
             └──────────────┼──────────────┘
                            ▼
                    ┌───────────────┐
                    │ AnalysisReport │
                    │ (JSON/SARIF)   │
                    └───────┬───────┘
                            │
              ┌─────────────┴─────────────┐
              ▼                           ▼
    ┌─────────────────┐         ┌─────────────────┐
    │  Lokal çıktı    │         │  Server Upload  │
    │  (stdout/file)  │         │  POST /analyses │
    └─────────────────┘         └────────┬────────┘
                                         │
                                         ▼
                                ┌─────────────────┐
                                │ Report Processor│
                                │ (async worker)  │
                                └────────┬────────┘
                                         │
                    ┌────────────────────┼────────────────────┐
                    ▼                    ▼                    ▼
             ┌───────────┐      ┌─────────────┐     ┌────────────┐
             │  Issue    │      │  Metrics    │     │  Quality   │
             │  Merge    │      │  Store      │     │  Gate Eval │
             └───────────┘      └─────────────┘     └─────┬──────┘
                                                          │
                                                          ▼
                                                   PASS / FAIL
                                                          │
                                                          ▼
                                              ┌───────────────────┐
                                              │ Webhook / CI exit │
                                              │ code / PR comment │
                                              └───────────────────┘
```

---

## 2. Faz 1: Dosya Keşfi

```python
# Pseudocode
def discover_files(sources, exclusions, inclusions):
    files = []
    for path in walk(sources):
        if matches_exclusion(path, exclusions):
            continue
        if inclusions and not matches_inclusion(path, inclusions):
            continue
        lang = detect_language(path)  # extension + shebang
        if lang in SUPPORTED_LANGUAGES:
            files.append(File(path, lang))
    return files
```

### Exclusion pattern örnekleri
```yaml
exclusions:
  - "**/node_modules/**"
  - "**/vendor/**"
  - "**/*.min.js"
  - "**/dist/**"
  - "**/__pycache__/**"
```

### Dil tespiti
| Uzantı | Dil |
|--------|-----|
| `.py` | python |
| `.js`, `.jsx` | javascript |
| `.ts`, `.tsx` | typescript |
| `.go` | go |
| `.rs` | rust |
| `.java` | java |
| `.cs` | csharp |

---

## 3. Faz 2: Parse & AST

Her dosya için dil parser'ı çalışır:

```
source code → Parser → AST → Rule Visitor → Issues[]
```

### Parse hataları
Syntax error olan dosyalar atlanmaz — ayrı issue olarak raporlanır:
```
rule: parser:syntax-error
severity: MAJOR
message: "Unable to parse file: unexpected indent at line 42"
```

---

## 4. Faz 3: Kural Çalıştırma

```
for file in files:
    ast = parse(file)
    for rule in active_rules[file.language]:
        issues += rule.check(ast, file)
```

### Paralelleştirme
- Dosya bazlı paralel (worker pool)
- Büyük projelerde: `GOMAXPROCS` veya thread pool
- Memory limit: dosya başına max AST cache

### Kural profili
Proje hangi kuralları kullanacağını profile'dan alır:
```yaml
quality_profile: "QualiGuard Way Python"
# veya
quality_profile: "Strict Security"
```

---

## 5. Faz 4: Metrik Hesaplama

### ncloc (Non-Comment Lines of Code)
```
Toplam satır
- Boş satırlar
- Yorum satırları (# // /* */)
= ncloc
```

### Cyclomatic Complexity
```
Her karar noktası (if, for, while, catch, &&, ||) +1
Fonksiyon başına ve toplam hesaplanır
Threshold: >10 → CODE_SMELL issue
```

### Cognitive Complexity
```
İç içe yapılar daha ağır penalize edilir
Okunabilirlik odaklı (SonarQube'un kendi algoritması)
```

### Duplication Detection
```
1. Dosyaları token stream'e çevir (yorum/whitespace strip)
2. Rolling hash ile minimum 10 satırlık blokları bul
3. Aynı hash → duplication issue
4. duplicated_lines_density = dup_lines / total_lines * 100
```

---

## 6. Faz 5: Rapor Oluşturma

### AnalysisReport şeması
```json
{
  "schema_version": "1.0",
  "scanner_version": "0.1.0",
  "project": {
    "key": "my-app",
    "name": "My Application",
    "version": "1.2.0"
  },
  "analysis": {
    "id": "uuid",
    "branch": "feature/auth",
    "commit": "a1b2c3d",
    "timestamp": "2026-07-08T10:00:00Z",
    "duration_ms": 4500
  },
  "issues": [
    {
      "rule_key": "python:unused-variable",
      "severity": "MINOR",
      "type": "CODE_SMELL",
      "message": "Remove unused variable 'result'",
      "file": "src/utils.py",
      "line": 42,
      "column": 5,
      "effort_minutes": 2,
      "fingerprint": "sha256:abc..."
    }
  ],
  "measures": {
    "ncloc": 5200,
    "files": 87,
    "complexity": 340,
    "cognitive_complexity": 280,
    "duplicated_lines_density": 2.1,
    "coverage": null
  },
  "coverage": {
    "line_coverage": 78.5,
    "branch_coverage": 65.0,
    "report_path": "coverage.xml"
  }
}
```

---

## 7. Faz 6: Sunucu İşleme (Issue Merge)

En kritik ve en zor parça:

```
Previous issues (DB)     New report issues
        │                        │
        └──────────┬─────────────┘
                   ▼
           Fingerprint Match
                   │
     ┌─────────────┼─────────────┐
     ▼             ▼             ▼
  MATCHED       NEW          MISSING
  (update)    (insert)      (close)
```

### Match senaryoları
| Durum | Aksiyon |
|-------|---------|
| Fingerprint eşleşti, hâlâ var | Status: OPEN (unchanged) |
| Fingerprint eşleşti, düzeltildi | Status: CLOSED, Resolution: FIXED |
| Yeni fingerprint | Status: OPEN (new issue) |
| Eski fingerprint yok artık | Status: CLOSED, Resolution: FIXED |
| Manuel FALSE_POSITIVE | Bir sonraki analizde ignore |

---

## 8. Faz 7: Quality Gate Değerlendirme

```python
def evaluate_gate(gate, measures, new_measures, issues):
    results = []
    for condition in gate.conditions:
        value = get_metric(condition.metric, new_measures or measures)
        passed = compare(value, condition.operator, condition.threshold)
        results.append(GateResult(condition, value, passed))
    
    if any(r.level == ERROR and not r.passed for r in results):
        return FAIL
    if any(r.level == WARN and not r.passed for r in results):
        return WARN
    return PASS
```

### "New Code" metrikleri
Branch analizi yapıldığında:
```
new_code = diff(main_branch, current_branch)
new_measures = calculate_only(new_code files)
gate applies to new_measures only
```

---

## 9. CI Entegrasyon Akışı

### GitHub Actions
```yaml
name: QualiGuard
on: [push, pull_request]

jobs:
  scan:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0  # full history for new code analysis

      - name: Install QualiGuard
        run: curl -sSL https://get.qualiguard.dev | sh

      - name: Run scan
        run: |
          qg scan \
            --project-key ${{ github.repository }} \
            --branch ${{ github.head_ref || github.ref_name }} \
            --upload \
            --server-url ${{ secrets.QG_SERVER }} \
            --token ${{ secrets.QG_TOKEN }} \
            --fail-on-gate

      - name: Upload SARIF
        uses: github/codeql-action/upload-sarif@v3
        with:
          sarif_file: qualiguard-report.sarif
```

### Exit code davranışı
| Code | Anlam |
|------|-------|
| 0 | Scan OK, gate passed |
| 1 | Gate failed |
| 2 | Scan error (parse fail, config error) |
| 3 | Upload failed |

---

## 10. Performans Hedefleri

| Proje boyutu | Hedef süre |
|-------------|------------|
| < 10K satır | < 5 saniye |
| 10K–100K satır | < 30 saniye |
| 100K–1M satır | < 5 dakika |
| > 1M satır | Incremental scan |

### Incremental scan (Faz 2+)
```
git diff --name-only main → sadece değişen dosyaları tara
önceki analiz issue'larını merge et
```

Sonraki dosya: `04-kurallar-ve-metrikler.md`
