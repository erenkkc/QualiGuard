# Release downloads

Built by `scripts/package-releases.bat` (or make package-releases).

Expected files:

- `QualiGuard-Kurulum.exe` — Windows tek tık kurulum (önerilen)
- `qualiguard-mac-kurulum.zip` — Mac tek tık kurulum (önerilen)
- `qualiguard-panel-windows.zip` — Windows zip (yedek)
- `qualiguard-panel-macos.zip` — Mac zip (yedek)
- `qg-windows-amd64.exe`
- `qg-darwin-amd64`

On the VPS, keep this folder next to the app work dir so `/api/v1/downloads` can serve them.
