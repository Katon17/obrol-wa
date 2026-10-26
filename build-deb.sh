#!/usr/bin/env bash
# Build a native Debian binary package without installing it or requiring root.
set -euo pipefail
# Package data must remain readable even with a restrictive caller umask.
umask 022

declare -A usage_text=()
obrol_locale="${OBROL_LANG:-${LC_ALL:-${LC_MESSAGES:-${LANG:-en}}}}"

usage_text[en]=$(cat <<'EOF'
Usage: ./build-deb.sh [--output DIR] [--help]

Always build a fresh native binary from source before packaging.
Output defaults to ./dist. Requires dpkg-dev and binutils; building also requires
Go, gcc, pkg-config, and libgtk-3-dev.

Environment overrides:
  DEB_VERSION     Package version (default: version in main.go, with -1 revision)
  DEB_MAINTAINER  Name <email> (default: repository Git identity)

Build on the oldest Debian/Ubuntu release you intend to support: detected library
requirements reflect the build machine. Cross-compilation is not supported.
EOF
)

usage_text[id]=$(cat <<'EOF'
Penggunaan: ./build-deb.sh [--output DIR] [--help]

Selalu buat biner natif yang baru dari kode sumber sebelum melakukan pengemasan.
Hasil keluaran secara default ditempatkan di ./dist. Memerlukan dpkg-dev dan binutils;
proses pembuatan juga memerlukan Go, gcc, pkg-config, dan libgtk-3-dev.

Pengaturan lingkungan (environment overrides):
DEB_VERSION     Versi paket (default: versi yang tercantum di main.go, dengan revisi -1)
DEB_MAINTAINER  Nama <email> (default: identitas Git dari repositori)

Lakukan proses pembuatan pada rilis Debian/Ubuntu tertua yang ingin Anda dukung:
persyaratan pustaka yang terdeteksi akan bergantung pada mesin tempat proses
pembuatan dilakukan. Kompilasi silang (cross-compilation) tidak didukung.
EOF
)

project_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
cd "$project_dir"

usage() {
  local language="${obrol_locale%%[_.@]*}"
  printf '%s\n' "${usage_text[$language]:-${usage_text[en]}}"
}

fail() { printf 'Error: %s\n' "$*" >&2; exit 1; }

output_dir="$project_dir/dist"

# Process CLI args
while (($#)); do
  case "$1" in
    --output)
      (($# >= 2)) || fail "$1 requires a value"
      output_dir="$(realpath -m -- "$2")"
      shift 2
      ;;
    -h|--help) usage; exit 0 ;;
    *) fail "Unknown argument: $1 (see --help)" ;;
  esac
done

# Check we have the packaging tools we need
for tool in dpkg dpkg-deb dpkg-architecture dpkg-shlibdeps readelf strip install gzip; do
  command -v "$tool" >/dev/null || fail "Missing tool: $tool (install dpkg-dev and binutils)"
done

# Check build tools
for tool in go gcc pkg-config; do
  command -v "$tool" >/dev/null || fail "Missing build tool: $tool"
done

# Check we have the GTK libs installed
pkg-config --exists gtk+-3.0 || fail 'Install libgtk-3-dev before building'


# Get the version from code with package version from environment if set
upstream_version="$(sed -nE 's/^const version = "([^"]+)".*/\1/p' main.go)"
package_version="${DEB_VERSION:-${upstream_version:+$upstream_version-1}}"
[[ -n "$package_version" ]] || fail 'Cannot determine version; set DEB_VERSION'
dpkg --validate-version "$package_version"

# Set the maintainer from environment
maintainer="${DEB_MAINTAINER:-}"

# If we don't have a maintainer set in code, load it from git if possible
if [[ -z "$maintainer" ]]; then
  maintainer_name="$(git config user.name || true)"
  maintainer_email="$(git config user.email || true)"
  [[ -n "$maintainer_name" && -n "$maintainer_email" ]] || fail 'Set DEB_MAINTAINER="Name <email>"'
  maintainer="$maintainer_name <$maintainer_email>"
fi

# Validate maintainer
[[ "$maintainer" != *$'\n'* && "$maintainer" == *'<'*'>' ]] || fail 'Invalid DEB_MAINTAINER; use Name <email>'

architecture="$(dpkg --print-architecture)"
build_dir="$(mktemp -d)"
trap 'rm -rf -- "$build_dir"' EXIT
package_root="$build_dir/package"

# Create directories for package
install -d "$package_root/DEBIAN" "$package_root/usr/bin" \
  "$package_root/usr/share/applications" \
  "$package_root/usr/share/icons/hicolor/256x256/apps" \
  "$package_root/usr/share/icons/hicolor/scalable/apps" \
  "$package_root/usr/share/doc/obrol"

# Build directly: build.sh currently suppresses build failures with || true.
# A fresh temporary output prevents accidentally packaging a stale executable.
printf 'Building Obrol from source…\n'
GOTOOLCHAIN=auto CGO_ENABLED=1 GOOS=linux GOARCH="$(go env GOHOSTARCH)" \
  CGO_CFLAGS="${CGO_CFLAGS:--O2 -Wno-deprecated-declarations}" \
  go build -trimpath -ldflags='-s -w' -o "$package_root/usr/bin/obrol" .

# Verify that the compiled ELF architecture matches the package.
elf_machine="$(LC_ALL=C readelf -h "$package_root/usr/bin/obrol" | sed -n 's/^ *Machine: *//p')"
expected_machine="$(dpkg-architecture -qDEB_HOST_GNU_CPU)"
case "$expected_machine:$elf_machine" in
  x86_64:*X86-64*|aarch64:AArch64|arm:ARM|i686:*80386*|riscv64:*RISC-V*|powerpc64le:PowerPC64|s390x:*S/390*) ;;
  *) fail "Binary architecture ($elf_machine) does not match native host ($expected_machine)" ;;
esac

# Strip the staged binary.
strip --strip-unneeded "$package_root/usr/bin/obrol"

# Copy across support files
install -m 644 obrol.desktop "$package_root/usr/share/applications/obrol.desktop"
# Ensure a final newline even if the source desktop entry lacks one.
sed -i -e '$a\' "$package_root/usr/share/applications/obrol.desktop"
install -m 644 assets/obrol.png "$package_root/usr/share/icons/hicolor/256x256/apps/obrol.png"
install -m 644 assets/obrol.svg "$package_root/usr/share/icons/hicolor/scalable/apps/obrol.svg"
install -m 644 LICENSE "$package_root/usr/share/doc/obrol/copyright"
for doc in README.md README.id.md; do
  gzip -n -9 -c "$doc" > "$package_root/usr/share/doc/obrol/$doc.gz"
done
if command -v desktop-file-validate >/dev/null; then
  desktop-file-validate "$package_root/usr/share/applications/obrol.desktop"
fi

# dpkg-shlibdeps expects Debian source metadata in its working directory.
install -d "$build_dir/debian"
cat > "$build_dir/debian/control" <<CONTROL
Source: obrol
Section: net
Priority: optional
Maintainer: $maintainer

Package: obrol
Architecture: any
Description: lightweight native WhatsApp desktop client
CONTROL
shared_deps="$(cd "$build_dir" && dpkg-shlibdeps -O -e"$package_root/usr/bin/obrol")"
shared_deps="$(printf '%s\n' "$shared_deps" | sed -n 's/^shlibs:Depends=//p')"
[[ -n "$shared_deps" ]] || fail 'No shared-library dependencies were detected'
# Non-native Debian packages include a packaging changelog.
cat > "$build_dir/changelog.Debian" <<CHANGELOG
obrol ($package_version) unstable; urgency=medium

  * Build binary package from the Obrol project sources.

 -- $maintainer  $(LC_ALL=C date -R)
CHANGELOG
gzip -n -9 -c "$build_dir/changelog.Debian" > "$package_root/usr/share/doc/obrol/changelog.Debian.gz"
installed_size="$(du -sk "$package_root/usr" | cut -f1)"
cat > "$package_root/DEBIAN/control" <<CONTROL
Package: obrol
Version: $package_version
Section: net
Priority: optional
Architecture: $architecture
Maintainer: $maintainer
Installed-Size: $installed_size
Depends: $shared_deps, xdg-utils, xdg-user-dirs, libwebp7
Recommends: ffmpeg, libcanberra-gtk3-module, libcanberra-gtk3-0
Homepage: https://github.com/Katon17/obrol-wa
Description: lightweight native WhatsApp desktop client
 Native GTK3 WhatsApp client using whatsmeow, with messaging, media,
 stickers, desktop notifications, and voice messages.
 This is an unofficial WhatsApp client.
CONTROL

mkdir -p -- "$output_dir"
# An epoch is valid metadata but need not appear in the filename.
package_file="$output_dir/obrol_${package_version#*:}_${architecture}.deb"
dpkg-deb --root-owner-group -Zxz --build "$package_root" "$package_file"
printf 'Created %s\n' "$package_file"
