#!/usr/bin/env bash
# build.sh — local development build
# Run from the project root (connect4/)
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

echo "==> Building server..."
cd server
go build -o ../server-bin ./cmd/server
cd ..
echo "    → server-bin"

echo ""
echo "Done. Start with:"
echo "  DATABASE_URL=postgres://... JWT_SECRET=... ./server-bin"
