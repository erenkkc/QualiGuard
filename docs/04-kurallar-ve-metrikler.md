# Kurallar ve Metrikler

QualiGuard rule engine ve metrik sistemi tasarımı.

---

## 1. Kural Taksonomisi

SonarQube ile uyumlu sınıflandırma:

### Issue Tipleri
| Tip | Açıklama | Örnek |
|-----|----------|-------|
| `BUG` | Runtime hatası riski | Null pointer, off-by-one |
| `VULNERABILITY` | Exploit edilebilir güvenlik açığı | SQL injection, XSS |
| `SECURITY_HOTSPOT` | Manuel inceleme gerektiren güvenlik noktası | Hardcoded password pattern |
| `CODE_SMELL` | Bakım/okunabilirlik sorunu | Uzun fonksiyon, unused var |

### Severity Seviyeleri
| Severity | Anlam | CI davranışı |
|----------|-------|--------------|
| `BLOCKER` | Derleme/deploy engelleyici | Gate fail |
| `CRITICAL` | Ciddi bug/güvenlik | Gate fail |
| `MAJOR` | Önemli sorun | Gate warn |
| `MINOR` | Küçük iyileştirme | Bilgi |
| `INFO` | Stil/convencion | Bilgi |

---

## 2. MVP Kural Seti

### Python (20 kural — Faz 1)
| Key | Tip | Severity | Açıklama |
|-----|-----|----------|----------|
| `python:unused-variable` | CODE_SMELL | MINOR | Kullanılmayan değişken |
| `python:unused-import` | CODE_SMELL | MINOR | Kullanılmayan import |
| `python:complex-function` | CODE_SMELL | MAJOR | Cyclomatic complexity > 10 |
| `python:long-function` | CODE_SMELL | MAJOR | Fonksiyon > 50 satır |
| `python:long-file` | CODE_SMELL | MAJOR | Dosya > 500 satır |
| `python:too-many-args` | CODE_SMELL | MAJOR | > 7 parametre |
| `python:empty-except` | BUG | MAJOR | Boş except bloğu |
| `python:bare-except` | BUG | CRITICAL | Bare `except:` |
| `python:sql-injection` | VULNERABILITY | BLOCKER | String concat SQL |
| `python:hardcoded-password` | VULNERABILITY | CRITICAL | Hardcoded credential |
| `python:eval-usage` | VULNERABILITY | CRITICAL | eval()/exec() kullanımı |
| `python:pickle-usage` | SECURITY_HOTSPOT | MAJOR | pickle.loads untrusted |
| `python:weak-crypto` | VULNERABILITY | CRITICAL | MD5/SHA1 for passwords |
| `python:command-injection` | VULNERABILITY | BLOCKER | os.system(user_input) |
| `python:path-traversal` | VULNERABILITY | CRITICAL | open(user_input) |
| `python:duplicate-literal` | CODE_SMELL | MINOR | Aynı string 3+ kez |
| `python:nested-if` | CODE_SMELL | MAJOR | 3+ nested if |
| `python:missing-docstring` | CODE_SMELL | INFO | Public func docstring yok |
| `python:mutable-default` | BUG | MAJOR | `def f(x=[])` |
| `python:syntax-error` | BUG | BLOCKER | Parse edilemeyen dosya |

### JavaScript/TypeScript (15 kural — Faz 2)
| Key | Tip | Severity |
|-----|-----|----------|
| `js:no-console` | CODE_SMELL | MINOR |
| `js:no-eval` | VULNERABILITY | CRITICAL |
| `js:xss-innerHTML` | VULNERABILITY | CRITICAL |
| `js:unused-variable` | CODE_SMELL | MINOR |
| `js:complex-function` | CODE_SMELL | MAJOR |
| `js:await-in-loop` | CODE_SMELL | MAJOR |
| `js:empty-catch` | BUG | MAJOR |
| `js:hardcoded-secret` | VULNERABILITY | CRITICAL |
| `js:sql-injection` | VULNERABILITY | BLOCKER |
| `js:prototype-pollution` | VULNERABILITY | CRITICAL |
| `js:weak-random` | VULNERABILITY | MAJOR |
| `js:missing-await` | BUG | CRITICAL |
| `js:any-usage` | CODE_SMELL | MINOR |
| `js:long-function` | CODE_SMELL | MAJOR |
| `js:syntax-error` | BUG | BLOCKER |

---

## 3. Kural Tanım Formatı

```yaml
# rules/python/sql-injection.yaml
key: python:sql-injection
name: SQL queries should use parameterization
description: |
  Building SQL queries with string concatenation or f-strings
  allows SQL injection attacks. Use parameterized queries instead.
severity: BLOCKER
type: VULNERABILITY
language: python
tags:
  - security
  - owasp-a03
  - cwe-89
effort_minutes: 30
security_standards:
  - OWASP Top 10 2021 - A03 Injection
  - CWE-89

detection:
  engine: ast_visitor
  logic: |
    # f-string veya concat ile cursor.execute / execute çağrısı
    match Call(func=Attribute(value=Name(id='cursor'), attr='execute'))
    where: has_string_interpolation(args[0])
  
  # Alternatif: regex fallback
  fallback:
    pattern: 'execute\s*\(\s*[f"\'].*\{'
    confidence: 0.7  # düşük confidence → SECURITY_HOTSPOT

message: "Use parameterized queries instead of string formatting in SQL"
quick_fix: null  # Faz 3+
```

---

## 4. Quality Profile (Kural Profili)

Proje bazlı hangi kuralların aktif olduğunu belirler:

```yaml
# profiles/qualiguard-way-python.yaml
name: QualiGuard Way Python
language: python
is_default: true
rules:
  - key: python:unused-variable
    severity: MINOR
    enabled: true
  - key: python:sql-injection
    severity: BLOCKER
    enabled: true
  - key: python:missing-docstring
    enabled: false  # kapalı
```

### Built-in profiller
| Profil | Açıklama |
|--------|----------|
| `QualiGuard Way` | Dengeli, varsayılan |
| `Strict Security` | Tüm güvenlik kuralları BLOCKER/CRITICAL |
| `Clean Code` | Code smell odaklı |
| `Minimal` | Sadece BUG + VULNERABILITY |

---

## 5. Metrik Sistemi

### Core metrikler
```yaml
metrics:
  # Size
  - key: ncloc
    type: INT
    description: Non-comment lines of code
  - key: files
    type: INT
    description: Analyzed file count

  # Complexity
  - key: complexity
    type: INT
    description: Total cyclomatic complexity
  - key: cognitive_complexity
    type: INT
    description: Total cognitive complexity
  - key: complexity_per_function
    type: FLOAT
    description: Average complexity per function

  # Issues
  - key: bugs
    type: INT
    domain: RELIABILITY
  - key: vulnerabilities
    type: INT
    domain: SECURITY
  - key: code_smells
    type: INT
    domain: MAINTAINABILITY
  - key: security_hotspots
    type: INT
    domain: SECURITY

  # Quality
  - key: duplicated_lines_density
    type: PERCENT
    domain: MAINTAINABILITY
  - key: coverage
    type: PERCENT
    domain: COVERAGE
  - key: sqale_index
    type: WORK_DAYS
    description: Technical debt in minutes

  # Ratings (computed)
  - key: reliability_rating
    type: RATING  # 1=A, 2=B, 3=C, 4=D, 5=E
  - key: security_rating
    type: RATING
  - key: maintainability_rating
    type: RATING
```

### Rating hesaplama
```python
def reliability_rating(bugs, ncloc):
    density = bugs / ncloc * 1000  # bugs per 1000 lines
    if density <= 0.05: return 'A'
    if density <= 0.1:  return 'B'
    if density <= 0.2:  return 'C'
    if density <= 0.5:  return 'D'
    return 'E'

def maintainability_rating(debt_minutes, ncloc):
    ratio = debt_minutes / (ncloc * 0.5)  # debt ratio
    if ratio <= 0.05: return 'A'
    if ratio <= 0.1:  return 'B'
    if ratio <= 0.2:  return 'C'
    if ratio <= 0.5:  return 'D'
    return 'E'
```

### Technical Debt
Her kuralın `effort_minutes` değeri toplanır:
```
sqale_index = sum(issue.effort_minutes for issue in open_issues)
```

---

## 6. Harici Tool Import

MVP'de kendi kurallarımız sınırlı — mevcut linter'ları entegre et:

```yaml
# qualiguard.yaml
imports:
  - name: ruff
    command: ruff check --output-format=json .
    language: python
    mapping:
      rule_id: external_id
      severity: map_severity  # E→CRITICAL, W→MAJOR, F→BLOCKER
  
  - name: bandit
    command: bandit -r src -f json
    language: python
    type_override: VULNERABILITY
  
  - name: eslint
    command: npx eslint --format json src/
    language: javascript
```

### SARIF import (universal)
```yaml
imports:
  - name: codeql
    file: codeql-results.sarif
    format: sarif
```

---

## 7. Kural Geliştirme Rehberi

### Adım 1: Kural tanımı yaz
```yaml
# rules/python/my-rule.yaml
key: python:my-rule
...
```

### Adım 2: Detector implement et
```python
# detectors/python/my_rule.py
from ..base import RuleDetector

class MyRuleDetector(RuleDetector):
    rule_key = "python:my-rule"
    
    def visit(self, node, context):
        if isinstance(node, ast.Call) and ...:
            yield Issue(
                rule_key=self.rule_key,
                line=node.lineno,
                message="..."
            )
```

### Adım 3: Test yaz
```python
# tests/rules/test_my_rule.py
def test_detects_bad_code():
    issues = scan("bad_code.py", rules=["python:my-rule"])
    assert len(issues) == 1
    assert issues[0].rule_key == "python:my-rule"

def test_ignores_good_code():
    issues = scan("good_code.py", rules=["python:my-rule"])
    assert len(issues) == 0
```

---

## 8. Güvenlik Standartları Mapping

Her güvenlik kuralı standartlara map edilir:

| Standart | Açıklama |
|----------|----------|
| OWASP Top 10 2021 | A01–A10 kategorileri |
| CWE | Common Weakness Enumeration ID |
| SANS Top 25 | En tehlikeli yazılım hataları |
| PCI DSS | Ödeme kartı güvenliği |

UI'da issue detayında gösterilir:
```
python:sql-injection
├── OWASP A03:2021 — Injection
├── CWE-89 — SQL Injection
└── Severity: BLOCKER
```

Sonraki dosya: `05-quality-gates.md`
