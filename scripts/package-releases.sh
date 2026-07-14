#!/usr/bin/env bash
# Build QualiGuard release downloads into ./releases/
set -euo pipefail
cd "$(dirname "$0")/.."
mkdir -p releases pack/staging-win pack/staging-mac cmd/qg-install/assets

echo "[1/5] Windows CLI..."
GOOS=windows GOARCH=amd64 go build -o releases/qg-windows-amd64.exe ./cmd/qg

echo "[2/5] Windows panel zip..."
GOOS=windows GOARCH=amd64 go build -o pack/staging-win/qg-server.exe ./cmd/qg-server
cp -f qualiguard.yaml pack/staging-win/qualiguard.yaml
cp -f pack/windows/BASLA.bat pack/staging-win/BASLA.bat
(cd pack/staging-win && zip -q -r ../../releases/qualiguard-panel-windows.zip .)

echo "[3/5] Windows one-click installer..."
cp -f pack/staging-win/qg-server.exe cmd/qg-install/assets/qg-server.exe
cp -f qualiguard.yaml cmd/qg-install/assets/qualiguard.yaml
GOOS=windows GOARCH=amd64 go build -o releases/QualiGuard-Kurulum.exe ./cmd/qg-install

echo "[4/5] Mac CLI..."
GOOS=darwin GOARCH=amd64 go build -o releases/qg-darwin-amd64 ./cmd/qg

echo "[5/5] Mac one-click kurulum..."
GOOS=darwin GOARCH=amd64 go build -o pack/staging-mac/qg-server ./cmd/qg-server
cp -f qualiguard.yaml pack/staging-mac/qualiguard.yaml
cp -f pack/macos/QualiGuard-Kur.command pack/staging-mac/QualiGuard-Kur.command
(cd pack/staging-mac && zip -q -j ../../releases/qualiguard-mac-kurulum.zip qg-server qualiguard.yaml QualiGuard-Kur.command)

echo "Done:"
ls -lh releases/
