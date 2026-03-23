#!/bin/bash
# Khởi chạy môi trường dev
# Chạy Go Engine + Vite dev server song song

set -e

cd "$(dirname "$0")/.."

# Build Go Engine
echo "Building Go Engine..."
cd go-engine
go build -o bin/go-engine-dev .
cd ..

# Chạy Go tests trước
echo "Running Go tests..."
cd go-engine && go test ./... 2>&1 | tail -5
cd ..

# Chạy TypeScript check
echo "Checking TypeScript..."
npx tsc --noEmit

echo ""
echo "Dev environment ready!"
echo "  Go Engine: go-engine/bin/go-engine-dev"
echo "  Frontend:  npm run dev"
echo "  Tauri:     npm run tauri dev"
