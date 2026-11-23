#!/usr/bin/env bash
# Installer Obrol untuk Linux Mint / Ubuntu / Debian (per-user, tanpa sudo).
#   bash install.sh             → pasang
#   bash install.sh --uninstall → hapus aplikasi (data chat tetap ada)
#   bash install.sh --purge     → hapus aplikasi + sesi login + riwayat chat
set -e
cd "$(dirname "$0")"
BIN="$HOME/.local/bin"
APPS="$HOME/.local/share/applications"
ICONS="$HOME/.local/share/icons/hicolor"
DATA="${XDG_DATA_HOME:-$HOME/.local/share}/obrol"

if [ "$1" = "--uninstall" ] || [ "$1" = "--purge" ]; then
  pkill -x obrol 2>/dev/null || true
  rm -f "$BIN/obrol" "$APPS/obrol.desktop" "$HOME/.config/autostart/obrol.desktop"
  rm -f "$ICONS"/*/apps/obrol.png
  [ "$1" = "--purge" ] && rm -rf "$DATA" && echo "✔ Data & sesi dihapus."
  command -v update-desktop-database >/dev/null && update-desktop-database "$APPS" 2>/dev/null || true
  echo "✔ Obrol sudah dihapus."
  exit 0
fi

# File hasil unduhan browser sering kehilangan izin eksekusi → pulihkan.
chmod +x obrol build.sh 2>/dev/null || true

if [ ! -f ./obrol ]; then
  echo "Binary 'obrol' tidak ditemukan di folder ini, build dari source…"
  bash ./build.sh
fi
if ! ./obrol --version >/dev/null 2>&1; then
  echo "✘ Binary tidak bisa dijalankan di sistem ini. Detail:"
  ./obrol --version || true
  echo "  → Pastikan file 'obrol' terunduh utuh (±19 MB), atau build dari source: bash build.sh"
  exit 1
fi

pkill -x obrol 2>/dev/null && sleep 1 || true   # tutup versi lama bila sedang jalan
mkdir -p "$BIN" "$APPS"
install -m 755 obrol "$BIN/obrol"

ICON_SRC="assets/obrol.png"
if [ ! -f "$ICON_SRC" ]; then
  ICON_SRC="$(mktemp --suffix=.png)"
  ./obrol --export-icon "$ICON_SRC"
fi
for s in 32 48 64 128 256; do
  mkdir -p "$ICONS/${s}x${s}/apps"
  if command -v convert >/dev/null 2>&1; then
    convert "$ICON_SRC" -resize ${s}x${s} "$ICONS/${s}x${s}/apps/obrol.png"
  else
    cp "$ICON_SRC" "$ICONS/${s}x${s}/apps/obrol.png"
  fi
done

mkdir -p "$ICONS/scalable/apps"
if [ -f assets/obrol.svg ]; then
  cp assets/obrol.svg "$ICONS/scalable/apps/obrol.svg"
fi

cat > "$APPS/obrol.desktop" <<DESK
[Desktop Entry]
Type=Application
Name=Obrol
GenericName=Klien WhatsApp
Comment=Klien WhatsApp native & super ringan
Exec=$BIN/obrol
Icon=obrol
Terminal=false
Categories=Network;InstantMessaging;Chat;GTK;
Keywords=whatsapp;wa;chat;pesan;obrol;
StartupNotify=true
StartupWMClass=obrol
DESK
command -v gtk-update-icon-cache >/dev/null && gtk-update-icon-cache -f -t "$ICONS" 2>/dev/null || true
command -v update-desktop-database >/dev/null && update-desktop-database "$APPS" 2>/dev/null || true

echo ""
echo "✔ Obrol $(./obrol --version | cut -d' ' -f2) terpasang!"
echo "  • Buka dari Menu → Internet → Obrol, atau ketik:  obrol"
echo "  • Data & sesi tersimpan di: $DATA"
case ":$PATH:" in *":$BIN:"*) ;; *) echo "  • (Catatan: $BIN belum ada di PATH — logout/login sekali agar perintah 'obrol' dikenali)";; esac
