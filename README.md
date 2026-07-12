# QualiGuard

SonarQube benzeri statik kod analizi ve kalite yönetim platformu.

## Durum

| Aşama | Durum |
|-------|-------|
| Faz 1 — CLI Scanner | ✅ |
| Faz 2 — Server + API | ✅ |
| Faz 3 — Web Dashboard + Kalite Kapısı | ✅ |
| AI-1 — Sorun açıklaması (şablon + opsiyonel Ollama) | ✅ |
| Çoklu dil — Python, JavaScript/TS, Go | ✅ (temel kurallar) |
| Faz 4 — CI/CD şablonları | ✅ |
| Docker | ✅ |

## Desteklenen diller

| Dil | Uzantılar | Kurallar |
|-----|-----------|----------|
| Python | `.py` | ~15 kural (güvenlik, kalite) |
| JavaScript/TypeScript | `.js`, `.jsx`, `.ts`, `.tsx` | eval, XSS, secret |
| Go | `.go` | secret, zayıf crypto, SQL format |

## Sorun açıklamaları (opsiyonel)

Tarama kuralları her zaman çalışır. Bir sorun bulunduğunda QualiGuard **Türkçe açıklama** üretir:

- **Şablon** — varsayılan, internet/API gerekmez
- **Ollama** — kuruluysa daha ayrıntılı açıklama (kod şirket dışına çıkmaz)

`qualiguard.yaml` içinde `ai.ollama` bölümünü açıp yerel modeli kurun:

```powershell
ollama pull llama3.1:8b
```

Varsayılan model: **llama3.1:8b** (daha akıcı Türkçe sohbet). Ollama yoksa şablon açıklamalar kullanılır.

Panelde sorun detayında **“Bu sorun ne anlama geliyor?”** kutusunu görürsünüz — ana sayfada ayrı bir AI bölümü yoktur.

### JavaScript — ESLint

`.js` / `.ts` dosyalarında QualiGuard kendi güvenlik kurallarına ek olarak **ESLint** (`eslint:recommended`) çalıştırır. İlk çalıştırmada `npx` paketi indirebilir; Node.js kurulu olmalıdır.

### Python — Ruff

`.py` dosyalarında yerleşik kurallara ek olarak **Ruff** çalıştırılır (stil, import, güvenlik). Kurulum:

```powershell
pip install ruff
```

Ruff yoksa yalnızca QualiGuard'ın kendi Python kuralları kullanılır.

## Masaüstü kısayolları

| Dosya | Ne yapar |
|-------|----------|
| **QualiGuard-Dashboard.bat** | Sunucu + tarayıcı |
| **QualiGuard-Server.bat** | Sadece sunucu |

(Kısayollar normal masaüstü ve OneDrive masaüstünde.)

## Kurulum sırası

| Adım | Dosya | Ne yapar |
|------|-------|----------|
| 0 | **`BASLA.bat`** | Tüm adımları listeler |
| Günlük | **`QualiGuard-Dashboard.bat`** | Sunucu + tarayıcı panel |
| **1** | **`GitHub-Push.bat`** | GitHub repo + push + CI |
| **2** | **`Domain-Deploy.bat`** | Domain + HTTPS + panel şifresi |
| Docker | **`Docker-Baslat.bat`** | Go kurmadan Docker ile çalıştır |

## Hızlı başlangıç (VS Code gerekmez)

1. **`QualiGuard-Dashboard.bat`** — çift tıkla (sunucu + tarayıcı paneli)
2. veya **`server.bat`** → tarayıcıda http://127.0.0.1:9000/app
3. `.py`, `.js`, `.go` dosyasını yükle veya Canlı Analiz kullan

## GitHub (CI)

1. **`GitHub-Hazirla.bat`** — adım adım rehber
2. Repo'ya push edince `.github/workflows/qualiguard.yml` otomatik çalışır (PR taraması + kalite kapısı)

## Docker (alternatif)

Docker Desktop kuruluysa:

```powershell
Docker-Baslat.bat
```

Panel: http://127.0.0.1:9000/app — `docker compose down` ile durdur.

## Production deploy

Marka adı, slogan ve renkler `qualiguard.yaml` içindeki `brand:` bölümünden gelir (white-label).

```powershell
cp .env.example .env
# .env içinde QG_DOMAIN ve QG_EMAIL ayarla
docker compose -f docker-compose.prod.yml up -d --build
```

- Landing: `https://alan-adiniz/`  
- Panel: `https://alan-adiniz/app`  
- İnternete açarken `.env` içinde `QG_PANEL_PASSWORD` ayarlayın → `/login` ile giriş  
- Yerel Docker (HTTPS yok): `docker compose up -d`

## CLI

```powershell
.\bin\qg.exe scan --config qualiguard.yaml -v
.\bin\qg.exe scan --incremental --base main   # sadece değişen dosyalar (git gerekir)
.\bin\qg.exe scan --fail-on-gate
.\bin\qg.exe scan --format sarif -o report.sarif
.\bin\qg.exe scan --pr-comment pr-summary.md   # GitHub PR yorumu (markdown)
```

## GitHub PR kontrolü

Repo kökünde `.github/workflows/qualiguard.yml` bulunur. Her PR'da:

- **QualiGuard Quality Gate** check'i çalışır (geçti/kaldı)
- PR'a özet yorum eklenir (toplam uyarı, kapı durumu, öne çıkan bulgular)
- SARIF GitHub Security sekmesine yüklenir

Kapı yalnızca kritik sorunlarda kırılır; stil uyarıları (`var` vb.) yorumda görünür ama merge'i engellemez.

Proje sayfasında bulguları **Yanlış alarm** veya **Düzeltilmeyecek** olarak işaretleyebilirsiniz — sonraki taramada yok sayılır.

## Web paneli — yeni özellikler

| Özellik | Açıklama |
|---------|----------|
| **YZ Sohbet** | `#/chat` — Ollama ile kod/kalite sohbeti |
| **Zip yükleme** | `.zip` arşivi güvenli çıkarılır ve taranır |
| **Canlı analiz** | Kod değişince ~700 ms sonra otomatik yeniden tarama |
| **Proje silme** | Proje listesi veya detay sayfasından |
| **Rapor dışa aktarma** | JSON veya HTML; *PDF / Yazdır* ile yazdırılabilir rapor |
| **Toplu proje silme** | Projeler sayfasında çoklu seçim + sil |
| **Proje arama** | Projeler sayfasında ada veya koda göre filtre |

## VS Code eklentisi

Kaydettiğiniz dosyayı yerel sunucuda tarar; uyarıları editörde gösterir (SonarLint benzeri).

```powershell
# Sunucu çalışsın
.\server.bat

# Eklenti derle
make build-extension
# veya: cd extension\qualiguard && npm install && npm run compile
```

Cursor/VS Code'da `extension/qualiguard` → F5 (Run Extension).  
Ayrıntı: `extension/qualiguard/README.md`

```powershell
make build-all    # CLI + sunucu
make build-server # yalnızca qg-server.exe
```

## Docker

```bash
docker compose up --build
```

## Dokümantasyon

`docs/` — mimari, kurallar, kalite kapısı, AI planı.

**Staj sunumu:** `docs/09-staj-sunumu-demo.md` — adım adım demo senaryosu. Hızlı başlat: `scripts\Demo-Sunum.bat`
