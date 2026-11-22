# Kemasan Flatpak / Flatpak packaging

Status: manifest siap-submit ke Flathub; pengiriman dipimpin kontributor
(baduhai, see issue #5) dengan persetujuan maintainer.

Isi folder ini:

- `io.github.Katon17.Obrol.yml` — manifest Flathub (runtime freedesktop 24.08).
- `io.github.Katon17.Obrol.metainfo.xml` — AppStream metainfo (wajib Flathub).
- `gen-deps.sh` — menghasilkan `go-deps-module.yaml` (cache module Go agar
  build Flathub offline). Jalankan tiap kali go.mod/go.sum berubah:
  `bash packaging/flatpak/gen-deps.sh`
- `go-deps-module.yaml` — hasil generator, JANGAN diedit manual.

Catatan teknis penting:

1. **libwebp dibangun sebagai module**: pembuat stiker mem-`dlopen`
   `libwebp.so.7/.so.6/.so` saat runtime; runtime freedesktop tidak menjamin
   soname itu ada di sandbox.
2. **Tray = StatusNotifierItem** (`sni.go`, own-name
   `io.github.Katon17.Obrol.SNI`); fallback XEmbed untuk panel tua di luar
   Flatpak.
3. **ISA baseline**: langkah `objcopy --remove-section=.note.gnu.property`
   di manifest = kebijakan yang sama dengan release.yml (CPU tua).
4. Sebelum submit: isi `commit:` pada source git di manifest dengan SHA tag
   v1.13.2, lalu uji lokal:
   `flatpak-builder --user --install-deps-from=flathub --install builddir packaging/flatpak/io.github.Katon17.Obrol.yml`
5. Hal yang diverifikasi bersama saat uji sandbox: ikon tray (SNI), notifikasi
   GApplication, unduhan media ke ~/Downloads/Obrol, dan tema GTK3 (dokumentasikan
   batasannya di halaman Flathub bila extension tema tidak tersedia).
