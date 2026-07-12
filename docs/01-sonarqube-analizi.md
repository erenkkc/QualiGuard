# SonarQube Derin Analizi

SonarQube'u kopyalamadan önce nasıl çalıştığını katman katman anlamak gerekir.

---

## 1. SonarQube Nedir?

SonarQube, **Continuous Code Quality** platformudur. Üç temel iş yapar:

1. **Statik kod analizi** — kaynak kodu çalıştırmadan tarar
2. **Kalite metrikleri** — karmaşıklık, tekrar, kapsam (coverage) hesaplar
3. **Quality Gate** — "bu kod release'e hazır mı?" sorusuna otomatik cevap verir

SonarSource ekosistemi 3 üründen oluşur:

| Ürün | Rol |
|------|-----|
| **SonarQube Server** | Merkezi sunucu, UI, API, geçmiş, quality gate |
| **SonarScanner** | CI/CD veya lokal ortamda çalışan analiz aracı |
| **SonarLint** | IDE eklentisi, anlık geri bildirim (connected/disconnected mode) |

---

## 2. Mimari Bileşenler

```
┌─────────────────────────────────────────────────────────┐
│                    SonarQube Server                      │
│  ┌──────────┐  ┌──────────────┐  ┌──────────────────┐  │
│  │ Web UI   │  │  REST API    │  │  Plugin Manager  │  │
│  └──────────┘  └──────────────┘  └──────────────────┘  │
│  ┌──────────────────────────────────────────────────┐   │
│  │           Compute Engine (CE)                     │   │
│  │  Rapor işleme · Metrik hesaplama · Gate eval     │   │
│  └──────────────────────────────────────────────────┘   │
└────────────────────────┬────────────────────────────────┘
                         │
              ┌──────────┴──────────┐
              │   PostgreSQL / H2     │
              │   (config + history)  │
              └───────────────────────┘

┌─────────────────────────────────────────────────────────┐
│              SonarScanner (CI / Developer)               │
│  ┌────────────┐  ┌─────────────┐  ┌────────────────┐  │
│  │ Config     │  │ Language    │  │ Report Upload  │  │
│  │ Reader     │→ │ Analyzers   │→ │ (HTTP POST)    │  │
│  └────────────┘  └─────────────┘  └────────────────┘  │
└─────────────────────────────────────────────────────────┘
```

### 2.1 Web Server Katmanı
- Spring Boot tabanlı Java uygulaması
- Proje yönetimi, kullanıcı/rol, dashboard
- REST API (`/api/...`) — tüm entegrasyonlar buradan

### 2.2 Compute Engine (CE)
- Analiz **asenkron** işlenir; scanner upload eder, CE kuyruğa alır
- Yeni analizi geçmişle birleştirir (issue'ların yaşam döngüsü)
- Metrikleri hesaplar, quality gate'i değerlendirir
- **Neden ayrı?** Büyük projelerde analiz saniyeler-saatler sürebilir; UI bloklanmaz

### 2.3 Veritabanı
- PostgreSQL (production), H2 (dev/test)
- Saklananlar: projeler, analiz geçmişi, issue'lar, kurallar, quality gate config, kullanıcılar

### 2.4 Plugin Sistemi
- Her dil ayrı plugin: `sonar-java-plugin`, `sonar-python-plugin`, vb.
- Plugin'ler rule tanımları + AST analiz motoru içerir
- Community vs Enterprise: dil desteği ve kural sayısı farklı

---

## 3. Analiz Pipeline (Adım Adım)

```
Developer/CI
    │
    ▼
[1] sonar-scanner -Dsonar.projectKey=myapp
    │
    ▼
[2] sonar-project.properties okunur
    │  (projectKey, sources, exclusions, language)
    ▼
[3] Dil plugin'leri devreye girer
    │  → Kaynak dosyalar parse edilir (AST)
    │  → Her kural AST üzerinde çalışır
    │  → Issue listesi + ham metrikler üretilir
    ▼
[4] Scanner "Analysis Report" oluşturur
    │  (issues, measures, duplications, coverage import)
    ▼
[5] HTTP POST → Server /api/ce/submit
    │
    ▼
[6] Compute Engine kuyruğa alır
    │  → Issue'ları DB'ye yazar (yeni/açık/kapalı/çözüldü)
    │  → Metrikleri aggregate eder
    │  → Quality gate değerlendirir
    ▼
[7] Sonuç UI'da görünür
    │  → Dashboard, issue listesi, PR decoration (CI entegrasyonu)
```

### Kritik Tasarım Kararı
**Analiz client-side (scanner), aggregation server-side (CE).**

Scanner ağır işi yapar; sunucu sadece birleştirir ve raporlar. Bu sayede:
- Sunucu CPU'su korunur
- CI pipeline'da paralel tarama yapılabilir
- Offline analiz mümkün (rapor sonradan upload)

---

## 4. Issue (Bulgu) Modeli

Her bulgu şu alanlara sahip:

| Alan | Açıklama |
|------|----------|
| `ruleKey` | Kural kimliği, örn. `python:S1481` |
| `severity` | BLOCKER, CRITICAL, MAJOR, MINOR, INFO |
| `type` | BUG, VULNERABILITY, CODE_SMELL, SECURITY_HOTSPOT |
| `message` | İnsan okunabilir açıklama |
| `component` | Dosya yolu |
| `line` | Satır numarası |
| `effort` | Düzeltme süresi tahmini (technical debt) |
| `status` | OPEN, CONFIRMED, RESOLVED, CLOSED, REOPENED |
| `resolution` | FIXED, WONTFIX, FALSE_POSITIVE, REMOVED |

### Issue Yaşam Döngüsü
```
Yeni analiz → Issue bulundu → OPEN
Kod düzeltildi → Sonraki analizde yok → CLOSED (FIXED)
Yanlış alarm → Manuel FALSE_POSITIVE
Aynı satırda farklı kural → Ayrı issue
```

Sonraki analizlerde **fingerprint/hash** ile aynı issue takip edilir.

---

## 5. Kural (Rule) Sistemi

### Kural Tipleri
- **Bug** — runtime hatası riski
- **Vulnerability** — güvenlik açığı (CVE benzeri)
- **Security Hotspot** — incelenmesi gereken güvenlik noktası
- **Code Smell** — bakım zorluğu, okunabilirlik

### Kural Tanımı (XML/YAML benzeri)
```xml
<rule key="S1481">
  <name>Unused local variables should be removed</name>
  <severity>MINOR</severity>
  <type>CODE_SMELL</type>
  <tag>unused</tag>
  <description>...</description>
</rule>
```

### Kural Uygulama Yöntemleri
1. **AST Visitor** — parse tree üzerinde gezinme (en yaygın)
2. **Semantic Analysis** — tip çözümleme, scope analizi
3. **Regex/Pattern** — basit kurallar (sınırlı)
4. **Data Flow Analysis** — taint tracking, null pointer (ileri seviye)
5. **External Tool Import** — ESLint, Pylint, SpotBugs sonuçlarını içe aktarma

---

## 6. Metrikler

### Kod Kalitesi Rating'leri (A–E)
Letter grade, threshold tabanlı:

| Rating | Anlam |
|--------|-------|
| A | Mükemmel |
| B | İyi |
| C | Orta |
| D | Zayıf |
| E | Kritik |

**Maintainability Rating** = Technical Debt Ratio'ya göre  
**Reliability Rating** = Bug sayısına/kod satırına göre  
**Security Rating** = Vulnerability sayısına göre

### Temel Metrikler
| Metrik | Açıklama |
|--------|----------|
| `ncloc` | Non-comment lines of code |
| `complexity` | Cyclomatic complexity toplamı |
| `cognitive_complexity` | Okunabilirlik karmaşıklığı |
| `duplicated_lines_density` | Tekrar eden kod yüzdesi |
| `coverage` | Test kapsamı (harici tool'dan import) |
| `sqale_index` | Technical debt (dakika cinsinden) |
| `bugs` | Bug tipi issue sayısı |
| `vulnerabilities` | Güvenlik açığı sayısı |
| `code_smells` | Code smell sayısı |
| `security_hotspots` | İncelenmemiş hotspot sayısı |

### Coverage Entegrasyonu
SonarQube kendi test çalıştırmaz. Coverage harici araçlardan gelir:
- Java: JaCoCo
- JS/TS: Istanbul/nyc
- Python: coverage.py
- C#: dotCover, OpenCover

Scanner bu raporları `sonar.coverage.jacoco.xmlReportPaths` gibi config ile okur.

---

## 7. Quality Gate

Tek soru: **"Bu kod release'e hazır mı?"**

### Varsayılan "Sonar Way" Gate
```
Yeni kodda:
  - 0 yeni BLOCKER issue
  - 0 yeni CRITICAL issue
  - Coverage on new code ≥ %80
  - Duplicated lines on new code ≤ %3
  - Maintainability rating on new code = A
```

### Gate Koşul Yapısı
```yaml
condition:
  metric: new_coverage
  operator: LT          # LT, GT, EQ, NE
  threshold: 80
  error_level: ERROR    # ERROR → FAIL, WARN → WARNING
```

### Sonuçlar
- **OK** — tüm koşullar geçti
- **WARN** — uyarı seviyesi ihlal
- **ERROR** — gate failed, CI pipeline kırılabilir

---

## 8. Proje Konfigürasyonu

`sonar-project.properties` örneği:
```properties
sonar.projectKey=my-app
sonar.projectName=My Application
sonar.projectVersion=1.0
sonar.sources=src
sonar.tests=tests
sonar.exclusions=**/*_test.py,**/vendor/**
sonar.python.coverage.reportPaths=coverage.xml
sonar.sourceEncoding=UTF-8
```

---

## 9. CI/CD Entegrasyonu

### GitHub Actions / GitLab CI / Jenkins
```yaml
- name: SonarQube Scan
  run: |
    sonar-scanner \
      -Dsonar.host.url=$SONAR_HOST \
      -Dsonar.token=$SONAR_TOKEN
```

### Pull Request Decoration
- PR'a yorum ekler: "Quality Gate: PASSED/FAILED"
- Yeni issue'ları PR diff'inde gösterir
- Branch analizi vs main branch karşılaştırması ("new code" metrikleri)

---

## 10. SonarLint (IDE)

| Mod | Davranış |
|-----|----------|
| **Standalone** | Lokal kural seti, sunucu gerekmez |
| **Connected** | Sunucu kuralları + quality profile sync |

Connected mode avantajı: takımın aynı kuralları IDE'de de görür.

---

## 11. SonarQube'un Güçlü Yanları

1. **Plugin mimarisi** — yeni dil eklemek modüler
2. **Issue tracking** — analizler arası süreklilik
3. **New code focus** — sadece yeni/değişen kodu sıkı kontrol
4. **Quality profiles** — proje/takım bazlı kural setleri
5. **Security rules** — OWASP Top 10, CWE mapping
6. **Olgun CI entegrasyonu** — her platformda hazır action/plugin
7. **Historical trending** — zaman içinde kalite grafiği

---

## 12. SonarQube'un Zayıf Yanları / Karmaşıklıkları

1. **Ağır altyapı** — Java server + PostgreSQL + (eski sürümlerde ES)
2. **Enterprise fiyatlandırma** — birçok dil/kural enterprise'da
3. **Yavaş ilk kurulum** — plugin, DB, CE tuning gerekir
4. **False positive yönetimi** — büyük projelerde issue gürültüsü
5. **Coverage dış bağımlılık** — test framework entegrasyonu manuel
6. **Monolitik server** — microservice değil, scale zorluğu

---

## 13. Bizim İçin Çıkarımlar

SonarQube klonlamak yerine **aynı problemleri çözen, daha hafif bir platform** hedeflenmeli:

| SonarQube | QualiGuard hedefi |
|-----------|-------------------|
| Java monolith server | Go/Rust veya Node API |
| PostgreSQL zorunlu | SQLite (dev) → PostgreSQL (prod) |
| Plugin JAR sistemi | WASM veya JSON rule pack |
| Ağır kurulum | Single binary + Docker |
| Enterprise dil kilidi | Açık kaynak, tüm diller |

Sonraki dosya: `02-mimari-tasarim.md`
