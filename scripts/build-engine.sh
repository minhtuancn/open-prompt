#!/usr/bin/env bash
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT="$SCRIPT_DIR/.."

cd "$ROOT/go-engine"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

if [ "$ARCH" = "x86_64" ]; then
    ARCH="amd64"
elif [ "$ARCH" = "aarch64" ]; then
    ARCH="arm64"
fi

echo "Building Go engine for $OS-$ARCH..."
go build -o "bin/go-engine-$OS-$ARCH" .

SIDECAR_DIR="$ROOT/src-tauri/binaries"
mkdir -p "$SIDECAR_DIR"

TARGET_TRIPLE=""
case "$OS-$ARCH" in
    linux-amd64)   TARGET_TRIPLE="x86_64-unknown-linux-gnu" ;;
    darwin-amd64)  TARGET_TRIPLE="x86_64-apple-darwin" ;;
    darwin-arm64)  TARGET_TRIPLE="aarch64-apple-darwin" ;;
    windows-amd64) TARGET_TRIPLE="x86_64-pc-windows-msvc" ;;
esac

cp "bin/go-engine-$OS-$ARCH" "$SIDECAR_DIR/go-engine-$TARGET_TRIPLE"
echo "Copied to $SIDECAR_DIR/go-engine-$TARGET_TRIPLE"
