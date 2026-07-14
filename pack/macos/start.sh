#!/bin/bash
cd "$(dirname "$0")"
chmod +x ./qg-server 2>/dev/null || true
echo "QualiGuard starting on http://127.0.0.1:9000"
open "http://127.0.0.1:9000/app" 2>/dev/null || true
exec ./qg-server --host 127.0.0.1 --port 9000 --data-dir "$HOME/.qualiguard-local" --work-dir "$(pwd)" --config qualiguard.yaml
