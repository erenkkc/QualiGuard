# QUALIGUARD — STAJ PROJE RAPORU

**Proje Adı:** QualiGuard  
**Konu:** SonarQube Benzeri Statik Kod Analizi ve Kalite Yönetim Platformu  
**Geliştirici:** [Adınız Soyadınız]  
**Kurum:** [Şirket / Üniversite Adı]  
**Tarih:** Temmuz 2026  

---

## 1. PROJENİN AMACI

QualiGuard, şirket içi kullanım için geliştirilmiş, **SonarQube benzeri** bir kod kalitesi platformudur. Amaç; yazılım projelerindeki güvenlik açıkları, hatalar ve kod kokularını otomatik tespit etmek, sonuçları Türkçe arayüzle sunmak ve CI/CD sürecinde **kalite kapısı (quality gate)** ile birleştirmeyi engellemektir.

SonarQube kurumsal lisans ve altyapı gerektirdiği için, daha hafif ve özelleştirilebilir bir alternatif ihtiyacı doğmuştur.

---

## 2. KULLANILAN TEKNOLOJİLER

| Katman | Teknoloji | Kullanım Amacı |
|--------|-----------|----------------|
| **Backend / CLI** | Go (Golang) 1.22+ | Ana uygulama, tarayıcı motoru, HTTP sunucu |
| **CLI framework** | Cobra | `qg scan` komut satırı arayüzü |
| **HTTP router** | Chi v5 | REST API yönlendirme |
| **Veritabanı** | SQLite | Proje, analiz ve bulgu geçmişi |
| **Frontend** | Vanilla JavaScript + HTML + CSS | Web paneli (framework yok, gömülü statik dosyalar) |
| **Python analizi** | Python 3 + gömülü AST script | Python dosya ayrıştırma |
| **Harici linter — JS** | ESLint (npx) | JavaScript/TypeScript stil ve güvenlik |
| **Harici linter — Python** | Ruff (pip) | Python stil, import ve güvenlik kuralları |
| **Versiyon kontrolü** | Git | Artımlı tarama (değişen dosyalar) |
| **CI/CD** | GitHub Actions | PR otomatik tarama ve kalite kapısı |
| **Rapor formatı** | JSON, SARIF | CLI çıktısı ve GitHub Security entegrasyonu |
| **Yapay zeka (opsiyonel)** | Ollama (yerel LLM) | Bulgu açıklamaları — kod dışarı çıkmaz |
| **Konteyner** | Docker Compose | İsteğe bağlı sunucu dağıtımı |

---

## 3. YAPILAN İŞLER (ÖZELLİKLER)

### 3.1. Komut Satırı Tarayıcı (CLI)

- `qg scan` komutu ile proje taraması
- JSON ve SARIF rapor üretimi
- `--fail-on-gate` ile CI'da hata kodu (exit 1)
- `--incremental --base main` ile yalnızca değişen dosyaların taranması
- `--pr-comment` ile GitHub PR özet yorumu (Markdown)
- `--baseline` ile bilinen bulguları yok sayma

### 3.2. Sunucu ve REST API

- `qg-server` HTTP sunucusu (port 9000)
- Proje oluşturma ve analiz yükleme
- Bulgu listesi, ölçümler, kalite kapısı durumu
- API token ile kimlik doğrulama
- Dosya yükleme ve kaynak kod saklama (tam kod görüntüleme)

### 3.3. Web Paneli (Dashboard)

- **Genel Bakış:** Proje listesi, açık sorun sayısı
- **Dosya Yükle:** Python, JavaScript, Go vb. — önizleme + kayıt
- **Canlı Analiz:** Editörde anında tarama, satır vurgulama
- **Projeler:** Bulgu listesi, filtreler, düzeltme önerileri
- **Kalite Kapısı görünümü:** Toplam uyarı / kritik ayrımı (Türkçe)

### 3.4. Kod Analizi Kuralları

**Python (yerleşik ~15 kural):**
- SQL injection, eval kullanımı, hardcoded şifre
- Bare except, kullanılmayan import/değişken
- Karmaşık/uzun fonksiyon, sözdizimi hatası

**JavaScript (yerleşik + ESLint):**
- eval, XSS (innerHTML), gizli anahtar tespiti
- `var` kullanımı, `==` yerine `===` (ESLint yoksa yerleşik kurallar)

**Go, Java, C#:** Temel güvenlik kuralları

**Ruff (Python):** F401, E501, S serisi güvenlik kuralları

### 3.5. Kalite Kapısı (Quality Gate)

- **QualiGuard Yolu** adlı varsayılan kapı
- Kapı yalnızca şunlarda kırılır: engelleyici, kritik, güvenlik açığı, hata (bug)
- Stil uyarıları (kod kokusu) kapıyı **düşürmez** — SonarQube mantığına uygun
- Arayüzde net ayrım: "46 toplam uyarı, kritik: 0, kapı: Geçti"

### 3.6. Yanlış Alarm İşaretleme

- Bulgu detayında **Yanlış alarm** ve **Düzeltilmeyecek** butonları
- İşaretlenen bulgular kapanır ve sonraki taramada yok sayılır
- Kapı sayımından çıkarılır; **Geri al** ile tekrar açılabilir

### 3.7. GitHub PR Entegrasyonu

- `.github/workflows/qualiguard.yml` workflow dosyası
- Her PR'da **QualiGuard Quality Gate** check'i
- PR'a otomatik özet yorum (toplam uyarı, kapı durumu, öne çıkan bulgular)
- SARIF → GitHub Security sekmesi
- Kapı kırılırsa merge engellenir

### 3.8. Türkçe Kullanıcı Deneyimi

- Tüm panel Türkçe
- Bulgu başına **Önerilen düzeltme** (adım adım Türkçe)
- **Bu sorun ne anlama geliyor?** açıklama kutusu (şablon veya Ollama)

### 3.9. Yapay Zeka Sadeleştirmesi

- Çoklu sağlayıcı (OpenAI, Gemini vb.) kaldırıldı
- Yalnızca **Ollama** (yerel, ücretsiz, gizlilik dostu)
- Ollama yoksa şablon tabanlı Türkçe açıklamalar

---

## 4. MİMARİ YAPISI

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│  qg scan    │────►│   Scanner    │────►│   Kurallar  │
│  (CLI)      │     │  + ESLint    │     │  + Ruff     │
└─────────────┘     │  + Ruff      │     └─────────────┘
       │            └──────────────┘            │
       │ upload                                ▼
       ▼                              ┌─────────────┐
┌─────────────┐                       │ Kalite Kapısı│
│ qg-server   │◄──────────────────────│  Değerlendir │
│ + SQLite    │                       └─────────────┘
│ + Web UI    │
└─────────────┘
       ▲
       │ GitHub Actions (PR)
┌─────────────┐
│  qualiguard │
│  .yml       │
└─────────────┘
```

**Proje klasör yapısı (özet):**

| Klasör | İçerik |
|--------|--------|
| `cmd/qg` | CLI giriş noktası |
| `cmd/qg-server` | HTTP sunucu |
| `internal/scanner` | Tarama orkestrasyonu |
| `internal/rules` | Yerleşik kurallar ve düzeltme önerileri |
| `internal/linter` | ESLint ve Ruff entegrasyonu |
| `internal/gate` | Kalite kapısı mantığı |
| `internal/store` | SQLite veritabanı |
| `internal/webui` | Web arayüzü (HTML/JS/CSS) |
| `internal/github` | PR yorumu üretici |
| `docs/` | Teknik dokümantasyon ve sunum rehberi |
| `demos/` | Sunum demo dosyaları |

---

## 5. SONUÇLAR VE KAZANIMLAR

### Teknik kazanımlar
- Go ile CLI ve HTTP sunucu geliştirme
- Statik kod analizi ve AST tabanlı kural motoru
- SQLite ile bulgu takibi ve fingerprint birleştirme
- GitHub Actions CI/CD entegrasyonu
- Harici linter (ESLint, Ruff) orkestrasyonu
- Türkçe UX ve kalite kapısı tasarımı

### İş değeri
- Şirket içi SonarQube alternatifi — lisans maliyeti yok
- PR aşamasında otomatik kod kalitesi kontrolü
- Öğrenci/ödev kodlarında stil vs kritik ayrımı (gerçek kullanım senaryosu)
- Yerel AI ile gizlilik korunarak bulgu açıklama

### Örnek tarama sonucu (ödev dosyası)
- `script.js`: 46 stil uyarısı (`var`, innerHTML), **0 kritik**, kapı **Geçti**
- `bad.py`: SQL injection, eval, hardcoded şifre → kapı **Kaldı**

---

## 6. YOL HARİTASI (GELECEK ÇALIŞMALAR)

| Öncelik | Özellik | Durum |
|---------|---------|-------|
| 1 | Artımlı tarama | Tamamlandı |
| 2 | ESLint (JS) + Ruff (Python) | Tamamlandı |
| 3 | GitHub PR kontrolü ve yorum | Tamamlandı |
| 4 | Yanlış alarm işaretleme | Tamamlandı |
| 5 | VS Code eklentisi | Planlandı |
| 6 | PostgreSQL (production DB) | Planlandı |
| 7 | Özel kural editörü (UI) | Planlandı |

---

## 7. KAYNAK VE DOSYALAR

| Dosya / Konum | Açıklama |
|---------------|----------|
| `C:\Users\Eren\Desktop\QualiGuard\` | Proje kök dizini |
| `README.md` | Hızlı başlangıç |
| `docs/09-staj-sunumu-demo.md` | Canlı sunum adımları |
| `docs/Staj-Raporu-QualiGuard.md` | Bu rapor |
| `scripts/Demo-Sunum.bat` | Sunum başlatıcı |
| `demos/` | Demo kod dosyaları |
| `http://127.0.0.1:9000` | Web paneli adresi |

---

**Hazırlayan:** [Adınız Soyadınız]  
**Staj dönemi:** [Başlangıç — Bitiş tarihi]  
**Danışman / Süpervizör:** [İsim]
