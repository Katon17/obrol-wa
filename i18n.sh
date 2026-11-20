#!/usr/bin/env bash
#
# Script internationalisation (i18n) support.
#
# Centralises non-english languages into one place. By using english as the key for the translation array, messages
# will default back to english when they are missing. Using bash arrays avoids using gnu-gettext or other dependencies
#

declare -A translations=()

obrol_locale="${OBROL_LANG:-${LC_ALL:-${LC_MESSAGES:-${LANG:-en}}}}"

case "$obrol_locale" in
  id|id_*|id.*|id@*)
    translations['Building Obrol…']='Meng-compile Obrol…'
    translations['Obrol %s installed!']='Obrol %s terpasang!'
    translations['Data and session stored in: %s']='Data & sesi tersimpan di: %s'
    translations['✔ Data and session deleted.']='✔ Data & sesi dihapus.'
    translations['✔ Obrol uninstalled.']='✔ Obrol sudah dihapus.'

    translations["Binary 'obrol' not found in this directory; building from source…"]="Binary 'obrol' tidak ditemukan di folder ini, build dari source…"
    translations['✘ The binary cannot run on this system. Details:']='✘ Binary tidak bisa dijalankan di sistem ini. Detail:'
    translations["  → Make sure 'obrol' was fully downloaded (±19 MB), or build from source: bash build.sh"]="  → Pastikan file 'obrol' terunduh utuh (±19 MB), atau build dari source: bash build.sh"

    translations['✔ Obrol %s installed!']='✔ Obrol %s terpasang!'
    translations['  • Open Menu → Internet → Obrol, or run: obrol']='  • Buka dari Menu → Internet → Obrol, atau ketik:  obrol'
    translations['  • Data and session stored in: %s']='  • Data & sesi tersimpan di: %s'
    translations["  • (Note: %s is not in PATH — log out and back in so the 'obrol' command is available)"]="  • (Catatan: %s belum ada di PATH — logout/login sekali agar perintah 'obrol' dikenali)"
    translations['Go is not installed. Install it first: sudo apt install golang-go']='Go belum terpasang. Pasang dulu:  sudo apt install golang-go'
    translations['GTK3 headers are missing. Install them first: sudo apt install build-essential pkg-config libgtk-3-dev']='Header GTK3 belum ada. Pasang dulu:  sudo apt install build-essential pkg-config libgtk-3-dev'
    translations['» Building Obrol (the first build takes about 3–6 minutes due to the large GTK bindings)…']='» Meng-compile Obrol (build pertama ±3–6 menit karena binding GTK besar)…'
    translations['✔ Done: ./obrol (%s)']='✔ Selesai: ./obrol (%s)'
    ;;
esac

msg() {
  local original="$1"
  shift
  # Get the translation, defaulting back to the original key (english)
  local format="${translations[$original]-$original}"
  printf "$format\n" "$@"
}