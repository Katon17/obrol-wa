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
# GOAMD64=v1 keeps the binary runnable on any x86-64 CPU (baseline ISA level).
# -march=x86-64 di CGO_CFLAGS: kode C cgo (sqlite3) tidak boleh terkompilasi
# dengan AVX dari -march default host/runner — SIGILL di CPU tua (issue #14).
export GOTOOLCHAIN=auto CGO_ENABLED=1 GOAMD64=v1 CGO_CFLAGS="-O2 -Wno-deprecated-declarations -march=x86-64 -mtune=generic"
echo "» Meng-compile Obrol (build pertama ±3–6 menit karena binding GTK besar)…"
go build -trimpath -ldflags="-s -w" -o obrol . 2>&1 | grep -v "warning:\|note:\|cgo-gcc-prolog\|In file included\|^\s*from \|^ *|\|^\s*[0-9]* |\|^#" || true
[ -x obrol ] && echo "✔ Selesai: ./obrol ($(du -h obrol | cut -f1))"
