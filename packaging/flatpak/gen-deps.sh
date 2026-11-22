#!/usr/bin/env bash
# gen-deps.sh — menghasilkan go-deps-module.yaml untuk manifest Flatpak.
# Isi cache module Go (zip/mod/info) dipetakkan ke sumber proxy.golang.org
# supaya build Flathub berjalan offline. Jalankan dari root repo:
#   bash packaging/flatpak/gen-deps.sh
set -e
cd "$(dirname "$0")/../.."
OUT=packaging/flatpak/go-deps-module.yaml
JSON=$(mktemp)
PY=$(mktemp /tmp/gendeps.XXXXXX.py)
command -v python3 >/dev/null || { echo "butuh python3"; exit 1; }
go mod download all >/dev/null 2>&1 || true
go mod download -json all 2>/dev/null > "$JSON" || true
cat > "$PY" <<'PY'
import json, sys, hashlib, os

inp, out = sys.argv[1], sys.argv[2]
mods = {}
dec = json.JSONDecoder()
raw = open(inp, encoding="utf-8", errors="replace").read()
i = 0
while i < len(raw):
    while i < len(raw) and raw[i] in " \t\r\n":
        i += 1
    if i >= len(raw):
        break
    try:
        d, j = dec.raw_decode(raw, i)
    except Exception:
        break
    i = j
    if isinstance(d, dict) and "Path" in d and "Version" in d:
        mods[(d["Path"], d["Version"])] = d

def esc(p):
    return "".join("!" + c.lower() if c.isupper() else c for c in p)

entries = []
for (path, ver), d in sorted(mods.items()):
    if path.startswith("golang.org/toolchain"):
        continue
    base = f"https://proxy.golang.org/{esc(path)}/@v/{ver}"
    cache = os.path.join(os.environ.get("GOMODCACHE", os.path.expanduser("~/go/pkg/mod")),
                         "cache/download", esc(path), "@v")
    for ext in ("mod", "info", "zip"):
        f = os.path.join(cache, f"{ver}.{ext}")
        if not os.path.exists(f):
            continue
        sha = hashlib.sha256(open(f, "rb").read()).hexdigest()
        entries.append(
            "      - type: file\n"
            f"        url: {base}.{ext}\n"
            f"        sha256: {sha}\n"
            f"        dest: gopath/pkg/mod/cache/download/{esc(path)}/@v\n")

with open(out, "w") as fh:
    fh.write("# DIHASILKAN OLEH gen-deps.sh — jangan diedit manual.\n")
    fh.write("name: obrol-go-deps\nbuildsystem: simple\nbuild-commands:\n")
    fh.write("  - mkdir -p /app/gopath && cp -a gopath/. /app/gopath/\nsources:\n")
    fh.writelines(entries)
print(f"{out}: {len(entries)} sumber")
PY
python3 "$PY" "$JSON" "$OUT"
rm -f "$JSON" "$PY"
