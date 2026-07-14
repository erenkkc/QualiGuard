# Release downloads

Built by `scripts/package-releases.bat` (or make package-releases).

Expected files:

- `qualiguard-panel-windows.zip`
- `qualiguard-panel-macos.zip`
- `qg-windows-amd64.exe`
- `qg-darwin-amd64`

On the VPS, keep this folder next to the app work dir so `/api/v1/downloads` can serve them.
