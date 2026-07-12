# Staj Sunumu — QualiGuard Demo Senaryosu

**Süre:** 10–15 dakika  
**Hedef kitle:** Teknik ekip / staj değerlendirme komitesi  
**Ana mesaj:** QualiGuard, şirket içi hafif bir SonarQube alternatifidir — CI, kapı, Türkçe arayüz, yerel AI.

---

## Sunumdan önce hazırlık (5 dk)

### 1. Ortamı aç
```powershell
# Masaüstü kısayolu varsa:
QualiGuard-Dashboard.bat

# Yoksa:
cd C:\Users\Eren\Desktop\QualiGuard
.\server.bat
# Tarayıcı: http://127.0.0.1:9000
```

### 2. Opsiyonel araçlar (daha zengin sonuç için)
```powershell
ollama pull llama3.1:8b   # YZ Sohbet (varsayılan model)
pip install ruff          # Python ek uyarıları
# Node.js kuruluysa ESLint otomatik çalışır (npx)
```

### 3. Demo dosyalarını hazır tut
| Dosya | Ne gösterir |
|-------|-------------|
| `demos/javascript-stil.js` | Kapı **Geçti**, çok stil uyarısı |
| `testdata/mehmet_script.js` | Gerçek ödev — ~46 uyarı, kapı geçer |
| `demos/python-stil.py` | Python stil (Ruff + yerleşik kurallar) |
| `demos/python-kritik.py` | Kapı **Kaldı** — SQL injection, eval, şifre |
| `testdata/sample_project/bad.py` | CLI taraması için kritik örnek |

### 4. Kontrol listesi
- [ ] Sunucu yeşil: "Sunucu çalışıyor" (sol alt)
- [ ] Tarayıcıda Ctrl+F5 (önbellek temiz)
- [ ] İnternet gerekmez (Ollama kapalıysa şablon açıklamalar yeterli)
- [ ] Genel Bakış'taki **Demo örnekleri** kartlarını dene

---

## Akış — adım adım

### Bölüm 1: Giriş (1 dk)

**Söyle:**
> "QualiGuard, SonarQube benzeri statik kod analizi platformu. Go ile yazıldı, SQLite kullanıyor, şirket içinde kurulumu kolay. Python, JavaScript ve Go destekliyor. CI'da PR kontrolü, web paneli ve kalite kapısı var."

**Göster:** Genel Bakış sayfası (`#/`) — proje sayısı, açık sorun özeti.

---

### Bölüm 2: Dosya yükleme + stil uyarıları (3 dk)

**Göster:** Dosya Yükle → `testdata/mehmet_script.js` (veya `demos/javascript-stil.js`)

1. **Analiz Et** — önizleme, kod tam görünür
2. **Projeye Kaydet**
3. Proje sayfasında dikkat çek:

| Ekranda | Ne anlama gelir |
|---------|-----------------|
| **46 toplam uyarı** | Tüm bulgular |
| **Kritik: 0** | Kapıyı etkilemeyen stil |
| **Kalite Kapısı: Geçti** | Merge engellenmez |
| Satır vurgusu | Tıklayınca ilgili satır |

**Söyle:**
> "Öğrenci ödevinde 46 `var` uyarısı var ama bunlar kod kokusu — güvenlik açığı değil. SonarQube'da da stil uyarıları kapıyı her zaman düşürmez; bizde de aynı mantık. 0'lar hata yok demek, 46 ise stil."

**Göster:** Bir sorun seç → **Önerilen düzeltme** (Türkçe adımlar) → **Bu sorun ne anlama geliyor?**

---

### Bölüm 3: Yanlış alarm işaretleme (2 dk)

**Göster:** Aynı projede bir `var` uyarısı seç

1. **Yanlış alarm** butonuna tıkla
2. Toplam uyarı sayısının düştüğünü göster (46 → 45)
3. Filtre: **Yanlış alarm** — işaretlenen bulgu görünür
4. **Geri al** ile geri açılabildiğini kısaca göster

**Söyle:**
> "Gerçek projelerde gürültüyü azaltmak için yanlış alarm işaretleme var. İşaretlenen bulgu sonraki taramada yok sayılır ve kapı sayımına girmez."

---

### Bölüm 4: Kritik sorun — kapı kaldı (2 dk)

**Göster:** Dosya Yükle → `demos/python-kritik.py`

- Kapı: **Kaldı**
- Kritik / güvenlik bulguları: SQL injection, eval, hardcoded password

**Söyle:**
> "Kritik sorun olduğunda kapı kırılır. CI'da PR merge engellenir. Stil uyarısı ile kritik sorun ayrımı QualiGuard'ın temel değer önerisi."

---

### Bölüm 5: Canlı analiz (1 dk)

**Göster:** Canlı Analiz → kısa Python veya JS yapıştır → **Analiz Et**

```python
password = "12345"
eval("print(1)")
```

**Söyle:** "Geliştirici IDE'ye girmeden hızlı test edebilir."

---

### Bölüm 6: CLI ve CI (2 dk)

**Terminalde (opsiyonel, hazır komut):**
```powershell
cd C:\Users\Eren\Desktop\QualiGuard
.\bin\qg.exe scan --config qualiguard.yaml -v
.\bin\qg.exe scan --incremental --base main -v
.\bin\qg.exe scan --pr-comment pr.md -o rapor.sarif -f sarif
type pr.md
```

**Söyle:**
> "Aynı motor CLI'dan çalışır. GitHub Actions her PR'da QualiGuard Quality Gate check'i ve özet yorum ekler. SARIF GitHub Security sekmesine gider."

**Göster (varsa):** GitHub repo → Actions → PR yorumu ekran görüntüsü.

---

### Bölüm 7: Mimari ve yol haritası (1 dk)

**Söyle — tek slayt veya sözlü:**

```
CLI (qg) ──► Scanner ──► Kurallar + ESLint/Ruff
                │
                ▼
         Server + SQLite + Web UI
                │
                ▼
         GitHub Actions (PR check)
```

**Tamamlananlar:** Artımlı tarama, ESLint, Ruff, PR yorumu, yanlış alarm, Türkçe UI  
**Sonraki:** Daha fazla dil, workspace taraması, özel kural editörü

**VS Code eklentisi:** `scripts\build-extension.bat` → F5 ile test

---

## Sık sorulan sorular — hazır cevaplar

| Soru | Cevap |
|------|--------|
| SonarQube'dan farkı? | Daha hafif, tek binary, Türkçe UI, şirket içi kurulum kolay, Ollama ile yerel AI |
| Neden kapı geçti 46 uyarı varken? | Kapı yalnızca kritik/güvenlik/hata sayar; stil uyarıları bilgi amaçlı |
| Node/Ruff şart mı? | Hayır; yoksa yerleşik kurallar çalışır |
| Veri dışarı çıkar mı? | Hayır; varsayılan Ollama yerel çalışır |
| Production DB? | SQLite şimdilik; PostgreSQL yol haritasında |

---

## Sorun giderme (sunum anı)

| Problem | Çözüm |
|---------|--------|
| Eski arayüz | Sunucuyu yeniden başlat + Ctrl+F5 |
| 0 bulgu | Dosya uzantısı `.js` / `.py` olmalı |
| Python hatası | `python --version` — PATH'te Python 3 gerekli |
| Sunucu yok | `server.bat` veya `go run ./cmd/qg-server` |

---

## Tek cümlelik kapanış

> "QualiGuard, şirketimizin kod kalitesini PR aşamasında otomatik kontrol eden, Türkçe arayüzlü ve hafif bir kalite platformu — SonarQube karmaşıklığı olmadan aynı iş akışını sunuyor."

---

## Ek: Hızlı demo scripti

Masaüstünde `Demo-Sunum.bat` çalıştırılabilir (`scripts/Demo-Sunum.bat`).
