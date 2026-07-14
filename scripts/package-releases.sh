#!/usr/bin/env bash
# Build QualiGuard release downloads into ./releases/
set -euo pipefail
cd "$(dirname "$0")/.."
mkdir -p releases pack/staging-win pack/staging-mac cmd/qg-install/assets

echo "[1/6] Windows CLI..."
GOOS=windows GOARCH=amd64 go build -o releases/qg-windows-amd64.exe ./cmd/qg

echo "[2/6] Windows desktop app..."
GOOS=windows GOARCH=amd64 go build -ldflags="-H windowsgui" -o pack/staging-win/QualiGuard.exe ./cmd/qg-desktop
cp -f qualiguard.yaml pack/staging-win/qualiguard.yaml

echo "[3/6] Windows one-click installer..."
cp -f pack/staging-win/QualiGuard.exe cmd/qg-install/assets/QualiGuard.exe
cp -f qualiguard.yaml cmd/qg-install/assets/qualiguard.yaml
GOOS=windows GOARCH=amd64 go build -o releases/QualiGuard-Kurulum.exe ./cmd/qg-install

echo "[4/6] Windows panel zip (yedek)..."
GOOS=windows GOARCH=amd64 go build -o pack/staging-win/qg-server.exe ./cmd/qg-server
cp -f pack/windows/BASLA.bat pack/staging-win/BASLA.bat
(cd pack/staging-win && zip -q -j ../../releases/qualiguard-panel-windows.zip qg-server.exe qualiguard.yaml BASLA.bat)

echo "[5/6] Mac CLI..."
GOOS=darwin GOARCH=amd64 go build -o releases/qg-darwin-amd64 ./cmd/qg

echo "[6/6] Mac one-click kurulum..."
GOOS=darwin GOARCH=amd64 go build -o pack/staging-mac/qg-server ./cmd/qg-server
cp -f qualiguard.yaml pack/staging-mac/qualiguard.yaml
cp -f pack/macos/QualiGuard-Kur.command pack/staging-mac/QualiGuard-Kur.command
(cd pack/staging-mac && zip -q -j ../../releases/qualiguard-mac-kurulum.zip qg-server qualiguard.yaml QualiGuard-Kur.command)

echo "Done:"
ls -lh releases/
