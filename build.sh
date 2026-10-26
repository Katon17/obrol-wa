#!/usr/bin/env bash
# Build Obrol dari source. Butuh: Go >= 1.22 (toolchain 1.26 diunduh otomatis), gcc, libgtk-3-dev.
set -e
cd "$(dirname "$0")"

# Import multi-lingual script support
# By having msg() use english as the key, this provides a default fallback language
source ./i18n.sh


if ! command -v go >/dev/null 2>&1; then
  msg 'Go is not installed. Install it first: sudo apt install golang-go'; exit 1
fi
if ! pkg-config --exists gtk+-3.0 2>/dev/null; then
  msg 'GTK3 headers are missing. Install them first: sudo apt install build-essential pkg-config libgtk-3-dev'; exit 1
fi
export GOTOOLCHAIN=auto CGO_ENABLED=1 CGO_CFLAGS="-O2 -Wno-deprecated-declarations"
msg '» Building Obrol (the first build takes about 3–6 minutes due to the large GTK bindings)…'
go build -trimpath -ldflags="-s -w" -o obrol . 2>&1 | grep -v "warning:\|note:\|cgo-gcc-prolog\|In file included\|^\s*from \|^ *|\|^\s*[0-9]* |\|^#" || true
[ -x obrol ] && msg '✔ Done: ./obrol (%s)' "$(du -h obrol | cut -f1)"
