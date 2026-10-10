# Contributing to Obrol / Kontribusi ke Obrol

Thanks for considering a contribution! / Terima kasih sudah mau berkontribusi!

Obrol is a solo-maintained open-source project (*Katon Bagaskoro*). Every help counts —
reporting bugs, suggesting features, fixing typos, translating, or sending pull requests.

Obrol adalah project open-source yang dirawat sendiri (*Katon Bagaskoro*). Semua bantuan berarti —
melaporkan bug, mengusulkan fitur, memperbaiki typo, menerjemahkan, atau mengirim pull request.

## Reporting bugs / Lapor bug

1. Check [existing issues](../../issues) first so we don't duplicate.
   Cek [issue yang sudah ada](../../issues) dulu supaya tidak duplikat.
2. Open a new issue using the **Bug report** template and include:
   Buka issue baru pakai template **Bug report** dan sertakan:
   - Obrol version (`obrol --version`) and Linux distro/desktop (e.g. Linux Mint 22 XFCE),
   - steps to reproduce, what you expected, what actually happened,
   - terminal output if the app printed errors (`obrol --debug` helps for connection issues),
   - screenshot if it is a visual problem.

## Suggesting features / Usulan fitur

Open an issue with the **Feature request** template. Keep in mind Obrol's goal:
**truly native, super lightweight**. A feature that needs a browser engine or adds
hundreds of MB of dependencies will probably be declined.

Buka issue dengan template **Feature request**. Ingat tujuan Obrol:
**benar-benar native, super ringan**. Fitur yang butuh browser engine atau menambah
dependensi ratusan MB kemungkinan besar akan ditolak.

## Building from source / Build dari source

```bash
sudo apt install build-essential pkg-config libgtk-3-dev golang-go   # Debian/Ubuntu/Mint
go build -trimpath -ldflags="-s -w" -o obrol .
./obrol --demo        # run without a WhatsApp account, with sample data
```

Useful flags: `--demo` (sample data, no connection), `--demo-login` (sample login page),
`--debug` (whatsmeow logs to terminal).

## Pull requests

1. Fork the repo and create a branch from `main`: `git checkout -b fix/short-name`.
2. Keep the code style: `gofmt` your changes (`gofmt -w *.go`), keep comments in Bahasa Indonesia
   (the codebase's comment language) while user-visible strings stay translatable-friendly plain text.
3. One logical change per PR; describe what and why in the PR description (template provided).
4. Test on a real session or with `--demo`, and mention how you tested.
5. UI text must stay in **Bahasa Indonesia** for now; app language packs are not implemented yet.

## Code of conduct

Be kind. All participants are expected to follow our [Code of Conduct](CODE_OF_CONDUCT.md).
Jaga sikap. Semua partisipan diharapkan mengikuti [Code of Conduct](CODE_OF_CONDUCT.md) kami.
