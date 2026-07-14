#!/usr/bin/env bash
# Build QualiGuard release downloads into ./releases/
set -euo pipefail
cd "$(dirname "$0")/.."
mkdir -p releases pack/staging-win pack/staging-mac

echo "[1/4] Windows CLI..."
GOOS=windows GOARCH=amd64 go build -o releases/qg-windows-amd64.exe ./cmd/qg

echo "[2/4] Windows panel..."
GOOS=windows GOARCH=amd64 go build -o pack/staging-win/qg-server.exe ./cmd/qg-server
cp -f qualiguard.yaml pack/staging-win/qualiguard.yaml
cp -f pack/windows/BASLA.bat pack/staging-win/BASLA.bat
(cd pack/staging-win && zip -q -r ../../releases/qualiguard-panel-windows.zip .)

echo "[3/4] Mac CLI..."
GOOS=darwin GOARCH=amd64 go build -o releases/qg-darwin-amd64 ./cmd/qg

echo "[4/4] Mac panel..."
GOOS=darwin GOARCH=amd64 go build -o pack/staging-mac/qg-server ./cmd/qg-server
cp -f qualiguard.yaml pack/staging-mac/qualiguard.yaml
cp -f pack/macos/start.sh pack/staging-mac/start.sh
(cd pack/staging-mac && zip -q -r ../../releases/qualiguard-panel-macos.zip .)

echo "Done:"
ls -lh releases/
