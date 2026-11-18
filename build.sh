#!/usr/bin/env bash
# Build Obrol dari source. Butuh: Go >= 1.22 (toolchain 1.26 diunduh otomatis), gcc, libgtk-3-dev.
set -e
cd "$(dirname "$0")"
if ! command -v go >/dev/null 2>&1; then
  echo "Go belum terpasang. Pasang dulu:  sudo apt install golang-go"; exit 1
fi
if ! pkg-config --exists gtk+-3.0 2>/dev/null; then
  echo "Header GTK3 belum ada. Pasang dulu:  sudo apt install build-essential pkg-config libgtk-3-dev"; exit 1
fi
export GOTOOLCHAIN=auto CGO_ENABLED=1 CGO_CFLAGS="-O2 -Wno-deprecated-declarations"
echo "» Meng-compile Obrol (build pertama ±3–6 menit karena binding GTK besar)…"
go build -trimpath -ldflags="-s -w" -o obrol . 2>&1 | grep -v "warning:\|note:\|cgo-gcc-prolog\|In file included\|^\s*from \|^ *|\|^\s*[0-9]* |\|^#" || true
[ -x obrol ] && echo "✔ Selesai: ./obrol ($(du -h obrol | cut -f1))"
