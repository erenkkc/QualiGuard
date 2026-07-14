#!/bin/bash
# QualiGuard Mac kurulum — cift tikla
set -e
APP_DIR="$HOME/Library/Application Support/QualiGuard"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

echo ""
echo "  QualiGuard kuruluyor..."
mkdir -p "$APP_DIR"
cp -f "$SCRIPT_DIR/qg-server" "$APP_DIR/qg-server"
cp -f "$SCRIPT_DIR/qualiguard.yaml" "$APP_DIR/qualiguard.yaml"
chmod +x "$APP_DIR/qg-server"

cat > "$APP_DIR/QualiGuard.command" << 'INNER'
#!/bin/bash
cd "$HOME/Library/Application Support/QualiGuard"
open "http://127.0.0.1:9000/app" 2>/dev/null || true
exec ./qg-server --host 127.0.0.1 --port 9000 --data-dir "$HOME/.qualiguard-local" --work-dir "$(pwd)" --config qualiguard.yaml
INNER
chmod +x "$APP_DIR/QualiGuard.command"

DESKTOP="$HOME/Desktop"
if [ -d "$DESKTOP" ]; then
  ln -sf "$APP_DIR/QualiGuard.command" "$DESKTOP/QualiGuard.command" 2>/dev/null || cp "$APP_DIR/QualiGuard.command" "$DESKTOP/QualiGuard.command"
  chmod +x "$DESKTOP/QualiGuard.command" 2>/dev/null || true
fi

# Sunucuyu arka planda baslat
nohup "$APP_DIR/qg-server" --host 127.0.0.1 --port 9000 \
  --data-dir "$HOME/.qualiguard-local" \
  --work-dir "$APP_DIR" \
  --config qualiguard.yaml >/dev/null 2>&1 &
sleep 2
open "http://127.0.0.1:9000/app" 2>/dev/null || true

echo ""
echo "  QualiGuard kuruldu!"
echo "  Panel: http://127.0.0.1:9000/app"
echo "  Masaustunde QualiGuard.command kisayolu var."
echo ""
read -r -p "  Kapatmak icin Enter'a basin..."
