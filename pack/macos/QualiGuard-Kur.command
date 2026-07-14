#!/bin/bash
# QualiGuard Mac baslatici — cift tikla
set -e
APP_DIR="$HOME/Applications/QualiGuard"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

mkdir -p "$APP_DIR"
cp -f "$SCRIPT_DIR/qg-server" "$APP_DIR/qg-server"
cp -f "$SCRIPT_DIR/qualiguard.yaml" "$APP_DIR/qualiguard.yaml"
chmod +x "$APP_DIR/qg-server"

# Arka planda sunucu
pkill -f "$APP_DIR/qg-server" 2>/dev/null || true
nohup "$APP_DIR/qg-server" --host 127.0.0.1 --port 9000 \
  --data-dir "$HOME/.qualiguard-local" \
  --work-dir "$APP_DIR" \
  --config qualiguard.yaml >/dev/null 2>&1 &

# Hazir olana kadar bekle
for i in $(seq 1 30); do
  if curl -sf "http://127.0.0.1:9000/api/health" >/dev/null 2>&1; then
    break
  fi
  sleep 0.3
done

# Uygulama penceresi gibi (Chrome/Edge app mode veya varsayilan tarayici)
URL="http://127.0.0.1:9000/desktop"
if [ -d "/Applications/Google Chrome.app" ]; then
  open -na "Google Chrome" --args --app="$URL" --new-window
elif [ -d "/Applications/Microsoft Edge.app" ]; then
  open -na "Microsoft Edge" --args --app="$URL" --new-window
else
  open "$URL"
fi
