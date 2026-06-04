#!/usr/bin/env bash
# build.sh — Build WASM + server + Docker image + push to GHCR
# Run from project root (connect-4/)
set -e

echo "==> Building WASM client..."
cd client
GOOS=js GOARCH=wasm go build -o ../static/game.wasm .
cd ..
echo "    → static/game.wasm"

echo "==> Copying wasm_exec.js..."
WASM_EXEC="$(go env GOROOT)/lib/wasm/wasm_exec.js"
if [ ! -f "$WASM_EXEC" ]; then
  WASM_EXEC="$(go env GOROOT)/misc/wasm/wasm_exec.js"
fi
cp "$WASM_EXEC" static/wasm_exec.js
echo "    → static/wasm_exec.js"

echo "==> Building server binary..."
cd server
go build -o ../server-bin ./cmd/server
cd ..
echo "    → server-bin"

echo "==> Building Docker image..."
docker build -t ghcr.io/uami-11/connect-4:latest .
echo "    → ghcr.io/uami-11/connect-4:latest"

echo "==> Pushing to GHCR..."
docker push ghcr.io/uami-11/connect-4:latest
echo "    → pushed"

echo ""
echo "Done! On your server, run:"
echo "  kubectl rollout restart deployment/connect4 -n connect4"
