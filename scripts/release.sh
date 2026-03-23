#!/bin/bash
# Tạo release build cho tất cả platforms
# Usage: ./scripts/release.sh v0.2.0

set -e

VERSION="${1:-v0.0.0}"
cd "$(dirname "$0")/.."

echo "Building Open Prompt $VERSION"
echo "================================"

# 1. Chạy tests
echo "[1/4] Running tests..."
cd go-engine && go test ./... -count=1 || { echo "Go tests failed!"; exit 1; }
cd ..
npx tsc --noEmit || { echo "TypeScript check failed!"; exit 1; }

# 2. Build Go Engine cho 3 platforms
echo "[2/4] Building Go Engine..."
./scripts/build-engine.sh all

# 3. Copy binaries vào Tauri sidecar
echo "[3/4] Copying binaries to Tauri..."
mkdir -p src-tauri/binaries
cp go-engine/bin/go-engine-linux-amd64 src-tauri/binaries/go-engine-x86_64-unknown-linux-gnu 2>/dev/null || true
cp go-engine/bin/go-engine-darwin-amd64 src-tauri/binaries/go-engine-x86_64-apple-darwin 2>/dev/null || true
cp go-engine/bin/go-engine-darwin-arm64 src-tauri/binaries/go-engine-aarch64-apple-darwin 2>/dev/null || true
cp go-engine/bin/go-engine-windows-amd64.exe src-tauri/binaries/go-engine-x86_64-pc-windows-msvc.exe 2>/dev/null || true

# 4. Build frontend
echo "[4/4] Building frontend..."
npm run build

echo ""
echo "Release $VERSION build complete!"
echo ""
echo "Next steps:"
echo "  npm run tauri build"
echo ""
echo "Code signing (tùy chọn):"
echo "  macOS:   export APPLE_SIGNING_IDENTITY='Developer ID Application: ...'"
echo "           export APPLE_ID=your@apple.id"
echo "           export APPLE_PASSWORD=app-specific-password"
echo "           export APPLE_TEAM_ID=XXXXXXXXXX"
echo "  Windows: export TAURI_SIGNING_PRIVATE_KEY_PASSWORD=..."
echo "           Đặt certificateThumbprint trong tauri.conf.json"
echo ""
echo "Auto-updater signing:"
echo "  export TAURI_SIGNING_PRIVATE_KEY=path/to/key"
echo "  npm run tauri build"
