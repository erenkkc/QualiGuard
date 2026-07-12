# QualiGuard — VS Code Eklentisi

Kaydettiğiniz veya workspace'teki Python, JavaScript/TypeScript, Go, Java ve C# dosyalarını yerel QualiGuard sunucusunda tarar; sorunları editörde gösterir, hover ile açıklama sunar.

## v0.2 yenilikleri

- **Workspace taraması** — komut paleti: `QualiGuard: Workspace'i tara`
- **Hover açıklama** — uyarılı satırın üzerine gelince kural, mesaj ve öneri
- **Panel şifresi** — uzak sunucuda bağlanırken şifre sorar

## Gereksinimler

- [QualiGuard sunucusu](http://127.0.0.1:9000) çalışıyor olmalı (`server.bat`)
- Node.js 18+ (derleme için)

## Kurulum (geliştirme)

```powershell
cd extension\qualiguard
npm install
npm run compile
```

VS Code / Cursor'da **Run Extension** (F5) ile test penceresi açılır.

## Kurulum (paket)

```powershell
cd extension\qualiguard
npm install
npm run package
code --install-extension qualiguard-0.2.0.vsix
```

## Kullanım

1. `server.bat` ile QualiGuard'ı başlatın
2. Desteklenen bir dosya açın (`.py`, `.js`, `.ts`, `.go`, …)
3. Kaydedince otomatik tarama yapılır
4. Komut paleti: **QualiGuard: Bu dosyayı tara** veya **QualiGuard: Workspace'i tara**
5. Uyarılı satırın üzerine gelin — hover ile detay görün

Sol alt durum çubuğunda bağlantı ve uyarı sayısı görünür.

## Ayarlar

| Ayar | Varsayılan | Açıklama |
|------|------------|----------|
| `qualiguard.serverUrl` | `http://127.0.0.1:9000` | Sunucu adresi |
| `qualiguard.token` | (boş) | API token; boşsa localhost bootstrap |
| `qualiguard.scanOnSave` | `true` | Kaydedince tara |
| `qualiguard.scanOnOpen` | `true` | Dosya açılınca tara |

## API

Eklenti `POST /api/v1/analyze/code` kullanır (`filename` + `source`).
