# Quality Gates

"Bu kod release'e hazır mı?" sorusuna otomatik cevap.

---

## 1. Quality Gate Nedir?

Quality Gate, analiz sonucu metriklerin **önceden tanımlı koşullara** göre değerlendirilmesidir.

```
Analiz tamamlandı
    → Metrikler hesaplandı
    → Gate koşulları kontrol edildi
    → PASS ✅ / WARN ⚠️ / FAIL ❌
    → CI pipeline devam eder veya durur
```

---

## 2. Built-in Gate: "QualiGuard Way"

SonarQube'un "Sonar Way" profilinin karşılığı:

```yaml
name: QualiGuard Way
is_default: true
conditions:
  # Yeni kodda sıfır tolerans
  - metric: new_bugs
    operator: GT
    threshold: 0
    level: ERROR

  - metric: new_vulnerabilities
    operator: GT
    threshold: 0
    level: ERROR

  - metric: new_blocker_issues
    operator: GT
    threshold: 0
    level: ERROR

  - metric: new_critical_issues
    operator: GT
    threshold: 0
    level: ERROR

  # Coverage
  - metric: new_coverage
    operator: LT
    threshold: 80
    level: ERROR

  # Duplication
  - metric: new_duplicated_lines_density
    operator: GT
    threshold: 3
    level: ERROR

  # Maintainability
  - metric: new_maintainability_rating
    operator: GT
    threshold: 1  # 1=A, 2=B, ...
    level: ERROR

  # Genel kod kalitesi (tüm kod)
  - metric: security_rating
    operator: GT
    threshold: 1  # A olmalı
    level: WARN
```

---

## 3. Gate Koşul Yapısı

```yaml
condition:
  metric: string       # Metrik adı
  operator: string     # LT | GT | EQ | NE | LTE | GTE
  threshold: number    # Eşik değeri
  level: string        # ERROR (fail) | WARN (warning)
  scope: string        # new | overall (default: overall)
```

### Operatörler
| Operatör | Anlam |
|----------|-------|
| `LT` | Küçüktür (<) |
| `GT` | Büyüktür (>) |
| `EQ` | Eşittir (==) |
| `NE` | Eşit değil (!=) |
| `LTE` | Küçük eşit (≤) |
| `GTE` | Büyük eşit (≥) |

### Scope: "New Code" vs "Overall"
| Scope | Ne zaman | Kullanım |
|-------|----------|----------|
| `new` | PR/branch analizi | Sadece değişen kod |
| `overall` | Main branch | Tüm proje |

**New code** hesaplama:
```
1. Base branch belirle (main)
2. git diff ile değişen satırları bul
3. Sadece o satırlardaki issue/metrikleri say
```

---

## 4. Hazır Gate Şablonları

### Strict Security
```yaml
name: Strict Security
conditions:
  - metric: vulnerabilities
    operator: GT
    threshold: 0
    level: ERROR
  - metric: security_hotspots_reviewed
    operator: LT
    threshold: 100
    level: ERROR
  - metric: security_rating
    operator: GT
    threshold: 1
    level: ERROR
```

### Clean Release
```yaml
name: Clean Release
conditions:
  - metric: new_bugs
    operator: GT
    threshold: 0
    level: ERROR
  - metric: new_code_smells
    operator: GT
    threshold: 5
    level: WARN
  - metric: new_coverage
    operator: LT
    threshold: 70
    level: ERROR
```

### Legacy Project (eski projeler için gevşek)
```yaml
name: Legacy Friendly
conditions:
  - metric: new_bugs
    operator: GT
    threshold: 0
    level: ERROR
  - metric: new_vulnerabilities
    operator: GT
    threshold: 0
    level: ERROR
  # Eski code smell'lere tolerans
  - metric: new_code_smells
    operator: GT
    threshold: 20
    level: WARN
```

---

## 5. Gate Değerlendirme Algoritması

```python
class QualityGateEvaluator:
    def evaluate(self, gate, measures, new_measures):
        results = []
        
        for cond in gate.conditions:
            data = new_measures if cond.scope == 'new' else measures
            actual = data.get(cond.metric)
            
            if actual is None:
                results.append(Result(cond, None, skipped=True))
                continue
            
            passed = self.compare(actual, cond.operator, cond.threshold)
            results.append(Result(cond, actual, passed, cond.level))
        
        # Karar
        if any(r.level == 'ERROR' and not r.passed for r in results):
            return GateStatus.FAIL, results
        if any(r.level == 'WARN' and not r.passed for r in results):
            return GateStatus.WARN, results
        return GateStatus.PASS, results
    
    def compare(self, actual, op, threshold):
        ops = {
            'LT': lambda a, t: a < t,
            'GT': lambda a, t: a > t,
            'EQ': lambda a, t: a == t,
            'NE': lambda a, t: a != t,
            'LTE': lambda a, t: a <= t,
            'GTE': lambda a, t: a >= t,
        }
        return ops[op](actual, threshold)
```

---

## 6. Gate Sonucu Raporlama

### CLI çıktısı
```
╔══════════════════════════════════════╗
║         QUALITY GATE: FAIL ❌        ║
╠══════════════════════════════════════╣
║ ❌ new_bugs: 2 (threshold: 0)         ║
║ ❌ new_vulnerabilities: 1 (threshold: 0) ║
║ ✅ new_coverage: 85% (threshold: 80%) ║
║ ✅ new_duplicated_lines: 1.2% (threshold: 3%) ║
╚══════════════════════════════════════╝
```

### API response
```json
{
  "gate": "QualiGuard Way",
  "status": "FAIL",
  "conditions": [
    {
      "metric": "new_bugs",
      "operator": "GT",
      "threshold": 0,
      "actual": 2,
      "passed": false,
      "level": "ERROR"
    }
  ]
}
```

---

## 7. CI Davranışı

```yaml
# qualiguard.yaml
quality_gate:
  fail_on: ERROR    # ERROR → exit 1, WARN → exit 0
  wait: true        # Sunucu işlemesini bekle
  timeout: 300      # 5 dakika max bekleme
```

### GitHub PR Decoration (Faz 3)
```
🟢 QualiGuard — Quality Gate PASSED

| Metric | Value | Threshold | Status |
|--------|-------|-----------|--------|
| New Bugs | 0 | 0 | ✅ |
| New Vulnerabilities | 0 | 0 | ✅ |
| Coverage on New Code | 92% | 80% | ✅ |
```

---

## 8. Gate Geçmişi ve Trending

Her analiz gate sonucunu kaydeder:

```sql
-- gate_history tablosu
analysis_id, gate_name, status, conditions_json, evaluated_at
```

UI'da gösterim:
```
Gate History (last 10 analyses)
✅ ✅ ❌ ✅ ✅ ✅ ✅ ❌ ✅ ✅
main  main  feat  main  main  ...
```

Sonraki dosya: `06-tech-stack.md`
