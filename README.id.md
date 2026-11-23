# Obrol: klien WhatsApp native untuk Linux

**[English](README.md)** · **Bahasa Indonesia**

[![build](https://github.com/Katon17/obrol-wa/actions/workflows/build.yml/badge.svg)](https://github.com/Katon17/obrol-wa/actions/workflows/build.yml)
[![rilis](https://img.shields.io/github/v/release/Katon17/obrol-wa?display_name=tag)](https://github.com/Katon17/obrol-wa/releases/latest)
[![lisensi](https://img.shields.io/badge/lisensi-MIT-green.svg)](LICENSE)
[![go](https://img.shields.io/badge/Go-1.26-00ADD8.svg?logo=go&logoColor=white)](go.mod)

Obrol adalah aplikasi WhatsApp desktop yang **benar-benar native** untuk Linux Mint XFCE (juga jalan di Ubuntu, Debian, MATE, dan Cinnamon).
Tidak memakai browser, Electron, maupun WebView. Tampilannya murni **GTK3**, dan koneksinya langsung ke server WhatsApp lewat
[whatsmeow](https://github.com/tulir/whatsmeow), library Go yang menjalankan protokol multi-device WhatsApp.

Dibuat oleh **Katon Bagaskoro**.

![Dark mode](screenshots/gelap-grup.png)

| Mode terang | Mute (klik kanan chat) | Login |
|---|---|---|
| ![Terang](screenshots/terang.png) | ![Mute](screenshots/menu-mute.png) | ![Login](screenshots/login.png) |

**Baru di 1.2:**

| Tempel screenshot (Ctrl+V) | Edit pesan | Pesan & obrolan disematkan |
|---|---|---|
| ![Tempel](screenshots/v12-pasted.png) | ![Edit](screenshots/v12-edit.png) | ![Sematkan](screenshots/v12-pin-obrolan.png) |

**Baru di 1.3:**

| Stiker (termasuk animasi) | Panel stiker: Terbaru / Favorit / Buat | Buat stiker dari foto atau screenshot |
|---|---|---|
| ![Stiker](screenshots/v13-stiker.png) | ![Panel](screenshots/v13-panel.png) | ![Buat](screenshots/v13-buat.png) |

**Baru di 1.4:**

| Balas dengan kutipan + reaksi emoji | Teruskan pesan ke chat mana pun | Antarmuka Bahasa Indonesia / Inggris |
|---|---|---|
| ![Balas](screenshots/v14-reply.png) | ![Teruskan](screenshots/v14-forward.png) | ![English](screenshots/v14-english.png) |

**Baru di 1.5:**

| Pesan suara (rekam & putar) | Baris panggilan tak terjawab |
|---|---|
| ![VN](screenshots/v15-pesan-suara.png) | ![Panggilan](screenshots/v15-panggilan.png) |

**Baru di 1.6:**

| Baris "Diarsipkan" di daftar chat | Isi folder arsip + tombol kembali |
|---|---|
| ![Arsip](screenshots/v16-arsip.png) | ![Isi arsip](screenshots/v16-arsip-view.png) |

**Baru di 1.7:**

| Profil kontak | Profil grup | Donasi di Tentang |
|---|---|---|
| ![Profil kontak](screenshots/v17-profil-kontak.png) | ![Profil grup](screenshots/v17-profil-grup.png) | ![About](screenshots/v18-about-donasi.png) |

**Baru di 1.9 — drag & drop + presence:**

| Indikator mengetik (grup) | Online / terakhir dilihat |
|---|---|
| ![Mengetik](screenshots/v19-mengetik.png) | ![Online](screenshots/v19-online.png) |

**Baru di 1.10–1.12 — gelombang irit & manajemen:**

- 🪶 **Diet RAM**: puncak ~636 MB → **<100 MB bahkan setelah pemakaian berat**
  (muat berbasis SQL, cache berplafon, memori dikembalikan ke OS)
- 🔎 **Cari pesan** (Ctrl+F) lintas semua percakapan + halaman **Berbintang** untuk pesan
  yang kamu klik kanan → *Bintangi*
- 🧑‍🤝‍ **Info & manajemen grup**: daftar peserta berhalaman dengan badge admin, ganti
  subjek, foto & deskripsi grup, tambah/hapus anggota — semua native

**Baru di 1.12.3–1.13 — bahasa & seleksi:**

- 🌐 **Bahasa otomatis**: menu Bahasa mendapat *Otomatis (ikuti sistem)* yang mengikuti
  locale desktop (kontribusi **baduhai**, PR #7)
- ✅ **Pilih beberapa pesan**: klik kanan → *Pilih pesan…* menampilkan bulatan ceklist di
  samping bubble; centang beberapa lalu **Teruskan**, **Bintang**, **Salin teks**, atau
  **Hapus** massal (untuk saya / untuk semua orang bila memenuhi syarat); Esc keluar
- 🗑 **Hapus untuk saya** per pesan langsung dari menu pesan — tanpa mode seleksi

**Baru di 1.14 — jangkau orang yang belum kamu simpan:**

- 💬 **Balas secara pribadi**: klik kanan pesan di grup → *Balas secara pribadi* membuka
  chat 1-on-1 dengan pengirimnya; aksi yang sama ada di menu klik-kanan daftar anggota
  grup — nomornya tidak perlu tersimpan di kontak

![Balas secara pribadi](screenshots/balas-privasi.png)

## Kenapa ringan?

| | Obrol | WhatsApp Web di browser | Klien Electron |
|---|---|---|---|
| Teknologi | GTK3 + Go (1 binary) | Tab Chrome/Firefox | Chromium terbungkus |
| Ukuran | ±19 MB | – | 150–250 MB |
| RAM | ±73–99 MB (diukur) | 300–600 MB | 300–500 MB |
| Proses | 1 | Banyak | Banyak |

## Fitur

- ✅ **Drag & drop berkas** langsung ke percakapan untuk mengirim (bisa beberapa berkas sekaligus)
- ✅ **Presence**: *online*, *terakhir dilihat kemarin, 21.04*, dan *sedang mengetik…* di header
  chat — di grup beserta nama pengetiknya ("Budi sedang mengetik…"); status mengetik Anda juga dikirim ke kontak
- ✅ Login lewat **kode QR** atau **kode 8 karakter dari nomor telepon**
- ✅ Daftar chat dengan avatar inisial, pratinjau pesan, jam, dan **badge belum dibaca**
- ✅ **Sinkronisasi riwayat** chat dari ponsel saat pertama login
- ✅ Kirim dan terima teks, emoji (Ctrl+. atau ikon 😊), dan link yang bisa diklik
- ✅ **Centang** ✓ terkirim, ✓✓ diterima, ✓✓ biru dibaca. Laporan baca juga dikirim ke lawan chat
- ✅ Thumbnail **foto/video**; klik untuk mengunduh lalu membuka di aplikasi default
- ✅ Buka dokumen, pesan suara, dan audio (disimpan ke `~/Unduhan/Obrol`)
- ✅ **Stiker**: diterima dan ditampilkan persis seperti di ponsel, **termasuk stiker animasi (WebP bergerak)**.
  Bubble stiker transparan tanpa latar, seperti WhatsApp asli
- ✅ **Kirim foto dan file** (📎) dengan keterangan
- ✅ **Tempel gambar dengan Ctrl+V**: screenshot di clipboard (tombol PrtSc, xfce4-screenshooter, Flameshot) atau
  file yang di-copy dari Thunar/Nemo langsung muncul di dialog kirim, lengkap dengan pratinjau dan keterangan
- ✅ **Pesan multi-baris**: `Enter` untuk kirim, `Shift+Enter` untuk baris baru. Kotak ketik ikut membesar sampai ±7 baris, lalu bisa di-scroll
- ✅ **Edit pesan** yang sudah terkirim (maksimal 15 menit, sama seperti WhatsApp): klik kanan pesan → ✏️ Edit pesan,
  atau tekan `↑` saat kotak ketik kosong untuk mengedit pesan terakhirmu. Pesan yang diedit ditandai "diedit"
- ✅ **Hapus untuk semua orang** (klik kanan pesanmu, maksimal ±2,5 hari)
- ✅ **Sematkan pesan** selama 24 jam, 7 hari, atau 30 hari (klik kanan pesan → 📌). Banner di atas percakapan
  menampilkan pesan yang disematkan; klik banner untuk lompat ke pesannya (kalau ada beberapa, diklik bergiliran)
- ✅ **Sematkan obrolan** di urutan teratas daftar chat (maksimal 3), tersinkron dengan ponsel
- ✅ Chat **grup** dengan nama pengirim berwarna
- ✅ **Cari pesan** (Ctrl+F) di semua percakapan; hasil klik langsung lompat ke pesannya
- ✅ **Bintangi pesan** dan buka lagi lewat halaman Berbintang
- ✅ **Info & manajemen grup**: peserta (berhalaman, badge admin), ganti subjek, foto &
  deskripsi grup, tambah/hapus anggota
- ✅ **Pilih beberapa pesan** (klik kanan → *Pilih pesan…*): centang bubble lalu teruskan,
  bintang, salin, atau hapus massal; Esc keluar
- ✅ **Balas secara pribadi** dari menu pesan grup atau daftar anggota (termasuk nomor yang belum disimpan)
- ✅ **Hapus untuk saya** per pesan tanpa masuk mode seleksi
- ✅ **Bahasa UI otomatis** mengikuti locale sistem, bisa ditimpa lewat menu Bahasa
- ✅ Pesan yang **dihapus** atau **diedit** orang lain ikut ter-update
- ✅ Lokasi (dengan link Google Maps), kontak, dan polling ditampilkan sebagai teks
- ✅ **Notifikasi desktop**; klik notifikasi untuk langsung membuka chat-nya. Ada suara notifikasi juga
- ✅ **Dark mode** ala WhatsApp. Mode *Otomatis* mengikuti tema XFCE (misal Mint-Y-Dark) dan ikut berubah saat tema diganti; bisa juga dipaksa *Terang* atau *Gelap* (Menu ☰ → Tema)
- ✅ **Mute chat/grup** selama 8 jam, 1 minggu, atau selamanya, lewat klik kanan di daftar chat atau tombol 🔔 di header chat.
  Status mute **tersinkron dua arah dengan ponsel**: grup yang sudah kamu bisukan di HP otomatis ikut dibisukan di Obrol
- ✅ **Bisukan semua grup** sekaligus (Menu ☰). Kamu tetap dapat notifikasi kalau ada yang **@mention** atau **membalas pesanmu**
- ✅ Chat yang dibisukan: ikon 🔕, badge abu-abu, dan **tidak dihitung** di judul jendela maupun titik merah tray
- ✅ **Ikon system tray** dengan titik merah saat ada pesan baru. Menutup jendela = sembunyi ke tray
- ✅ **Jalankan saat login** (autostart, langsung tersembunyi di tray)
- ✅ Obrolan baru ke nomor mana pun (nomor dicek dulu apakah terdaftar di WA)
- ✅ Cari chat, separator tanggal (Hari ini/Kemarin/nama hari), riwayat lama dimuat saat scroll ke atas
- ✅ **Panel stiker** (tombol di samping emoji): tab *Terbaru* (stiker yang pernah dipakai/diterima, urut pemakaian),
  *Favorit*, dan *Buat*. Klik untuk mengirim lagi; klik kanan untuk favoritkan atau sembunyikan
- ✅ **Buat stiker sendiri** dari foto/file gambar atau screenshot di clipboard: dipotong persegi otomatis,
  sudut membulat, garis tepi putih, lalu dikirim sebagai stiker WebP ≤100 KB (format stiker WhatsApp)
- ✅ Tombol **"Jadikan stiker…"** langsung di dialog kirim gambar
- ✅ **Balas pesan** (klik kanan → Balas): kutipan pesan asli tampil di dalam bubble dengan warna nama
  pengirim; klik kutipan untuk lompat ke pesan aslinya
- ✅ **Reaksi emoji**: klik kanan → Reaksi (👍 ❤️ 😂 😮 😢 🙏); chip reaksi dengan jumlah peserta tampil
  di bawah bubble; klik chip untuk ganti/hapus reaksimu. Reaksi dari ponsel ikut tersinkron
- ✅ **Teruskan pesan** ke chat mana pun lewat dialog pencari obrolan; pesan terusan diberi tanda
  *Diteruskan*. Teks, stiker, dan media (diunduh lalu diunggah ulang) bisa diteruskan
- ✅ **Dwibahasa**: Menu ☰ → Bahasa → *Bahasa Indonesia* atau *English*; aplikasi mulai ulang otomatis
- ✅ **Pesan suara**: tombol mik merekam VN (opus lewat ffmpeg, fallback PulseAudio/ALSA), klik lagi
  untuk kirim; VN masuk diputar langsung di aplikasi
- ✅ **Notifikasi panggilan**: panggilan masuk/tak terjawab tampil sebagai baris 📞 di chat plus
  notifikasi desktop (menjawab tetap di ponsel — protokol unofficial tidak mendukung audio panggilan)
- ✅ **Arsip chat**: klik kanan chat → *Arsipkan obrolan*; semua chat tersimpan rapi di baris
  **Diarsipkan** (dengan jumlah), bisa dibuka dan dikembalikan kapan pun; status tersimpan permanen
- ✅ **Profil kontak & grup**: klik header chat untuk melihat foto profil (otomatis diunduh &
  di-cache), nomor, teks about, serta jumlah peserta + deskripsi grup; ada tombol salin nomor
- ✅ **Bahasa Inggris penuh**: seluruh antarmuka (login, tooltip, status, dialog stiker, dll.)
  diterjemahkan — tidak ada lagi campuran Indonesia saat mode English
- ☕ **Dukungan**: tautan Saweria **dan Ko-fi** di dialog *Tentang Obrol*
- ✅ **Ikon & logo buatan sendiri**: logo "Ngobrol" (dua gelembung obrolan) dan 27 ikon garis tipis yang
  digambar khusus sebagai SVG, sehingga tajam di skala berapa pun dan warnanya ikut tema terang/gelap
- ✅ Single-instance: membuka Obrol lagi cukup memunculkan jendela yang sudah ada

## Pintasan keyboard

| Tombol | Aksi |
|---|---|
| `Ctrl+F` / `Ctrl+K` | fokus ke pencarian chat |
| `Ctrl+Shift+F` | cari di dalam percakapan yang terbuka |
| `Ctrl+N` | obrolan baru ke nomor telepon |
| `Ctrl+V` | tempel gambar/screenshot ke kolom pesan |
| `Enter` / `Shift+Enter` | kirim / baris baru |
| `↑` (kolom kosong) | edit lagi pesan terkirim terakhirmu |
| `Esc` | keluar mode seleksi / cari-di-chat / balas / edit, lalu tutup percakapan |
| `Ctrl+W` / `Ctrl+Q` | sembunyikan jendela ke tray / keluar |

**Pintasan:** `Enter` kirim · `Shift+Enter` baris baru · `Ctrl+V` tempel gambar/teks · `↑` edit pesan terakhir ·
`Ctrl+F` cari · `Ctrl+N` obrolan baru · `Esc` batal edit / tutup obrolan · `Ctrl+W` sembunyikan · `Ctrl+Q` keluar

## Instalasi (Linux Mint 21/22, x86_64)

> **CPU:** prosesor x86-64 apa pun. Biner rilis menarget x86-64-baseline (v1):
> build mengunci `GOAMD64=v1` dengan flag CGO tetap, dan `build-deb.sh`
> menggagalkan rilis jika biner menuntut level ISA lebih baru — jadi CPU
> pra-AVX2 (mis. Intel Sandy Bridge / Core gen-2) tetap bisa menjalankan Obrol.
> Jika suatu saat melihat pesan glibc `CPU ISA level is lower than required`,
> biner itu bukan dari pipeline rilis kami; verifikasi dengan
> `readelf -nW obrol | grep 'ISA needed'` (harusnya `x86-64-baseline`).

### Opsi A — paket `.deb` (disarankan)

Unduh `obrol_1.14.1-1_amd64.deb` dari [Releases](https://github.com/Katon17/obrol-wa/releases/latest), lalu:

```bash
sudo apt install ./obrol_1.14.1-1_amd64.deb
```

apt otomatis membereskan semua dependensi dan memasang sistem-wide
(`/usr/bin/obrol` + entri menu + ikon). Hapus dengan `sudo apt remove obrol`
— data chat di `~/.local/share/obrol/` tetap aman.

### Opsi B — tarball (per-user, tanpa sudo)

Binary sudah jadi, jadi **tidak perlu install Go**. Unduh tarball terbaru dari
halaman [Releases](https://github.com/Katon17/obrol-wa/releases/latest), lalu:

```bash
tar xzf obrol-1.14.1-linux-x86_64.tar.gz
cd obrol
bash install.sh
```

Setelah itu buka dari **Menu → Internet → Obrol**, atau jalankan `obrol` di terminal.
Instalasinya per-user (tanpa sudo):

- Binary: `~/.local/bin/obrol`
- Menu: `~/.local/share/applications/obrol.desktop`
- Data (sesi login + riwayat chat): `~/.local/share/obrol/`

> **Flatpak:** manifest siap-Flathub ada di `packaging/flatpak/` (pengiriman
> sedang diproses, issue #5). Sampai tayang di Flathub, `.deb` dan tarball
> tetap jalur instalasi yang disarankan.

Menghapus aplikasi:

```bash
bash install.sh --uninstall   # hapus aplikasi, data chat tetap disimpan
bash install.sh --purge       # hapus aplikasi + sesi login + riwayat chat
```

> Agar perangkat juga hilang dari daftar *Perangkat tertaut* di ponsel, pilih **Menu ☰ → Keluar dari akun** sebelum uninstall.

## Build dari source (opsional)

```bash
git clone https://github.com/Katon17/obrol-wa.git
cd obrol-wa
sudo apt install build-essential pkg-config libgtk-3-dev golang-go
./build.sh        # build pertama ±3–6 menit (binding GTK-nya besar)
./install.sh
```

Go 1.22 atau lebih baru sudah cukup. Toolchain 1.26 yang dibutuhkan whatsmeow akan diunduh otomatis.

Untuk membangun paket Debian/Ubuntu tanpa memasangnya:

```bash
sudo apt install dpkg-dev binutils  # tambahan dari dependensi build di atas
./build-deb.sh
```

Paket jadi disimpan di `dist/`. Skrip membaca versi dari `main.go` dan memakai
nama/email Git repositori sebagai maintainer paket; bisa ditimpa lewat
`DEB_VERSION` dan `DEB_MAINTAINER` (lihat `./build-deb.sh --help`). Lakukan build
pada rilis Debian/Ubuntu tertua yang ingin didukung, karena kebutuhan pustaka
runtime dideteksi dari mesin build. Skrip tidak butuh root dan tidak memasang
paketnya.

Opsi baris perintah:

| Opsi | Fungsi |
|---|---|
| `--hidden` | mulai tersembunyi di tray (dipakai autostart) |
| `--demo` | mode demo dengan data contoh, tanpa koneksi |
| `--debug` | tampilkan log whatsmeow di terminal |
| `--version` | tampilkan versi |

## ⚠️ Penting: risiko dan batasan

- **Tidak resmi.** Obrol memakai protokol WhatsApp hasil reverse-engineering (whatsmeow) dan **tidak berafiliasi dengan WhatsApp/Meta**.
  Ini melanggar Ketentuan Layanan WhatsApp. Ada kemungkinan akun **dibatasi atau diblokir**, walaupun untuk pemakaian chat
  normal hal ini jarang terjadi (whatsmeow juga dipakai oleh banyak bridge seperti mautrix-whatsapp). **Jangan dipakai untuk
  spam atau pesan massal.** Kalau khawatir, pakai nomor cadangan dulu.
- **Belum ada:** panggilan suara/video, kirim pesan suara, status/story, reaksi (pesan reaksi tidak ditampilkan),
  balas/kutip pesan, foto profil, dan saluran (channel).
- Pesan "sekali lihat" tidak bisa dibuka (sama seperti WhatsApp Web).
- Kalau WhatsApp mengubah protokolnya, cukup **build ulang** dengan whatsmeow terbaru:
  `go get go.mau.fi/whatsmeow@latest && ./build.sh && ./install.sh`.
- Ponsel harus online minimal sekali setiap ±14 hari, kalau tidak perangkat tertaut akan dilepas otomatis.

## Riwayat versi

Catatan lengkap per rilis ada di [CHANGELOG.md](CHANGELOG.md).

- **1.14.1**: cakupan terjemahan Inggris penuh (menu, tanggal, tooltip, dialog, error)
- **1.14.0**: *Balas secara pribadi* dari menu pesan grup dan daftar anggota; placeholder
  kolom ketik dipersingkat
- **1.13.0**: pilih beberapa pesan dengan aksi massal; *hapus untuk saya* satuan; menu
  pesan dibuka saat lepas klik (tanpa aksi tak sengaja)
- **1.12.3**: bahasa otomatis mengikuti sistem (PR #7 oleh baduhai); perbaikan
  format-string `go vet`
- **1.12.2**: stabilisasi pustaka whatsmeow; percobaan fitur Status/Story ditarik
  (perangkat tertaut tidak menerima riwayat status dari protokol)
- **1.12.0–1.12.1**: halaman info & manajemen grup (peserta, subjek, foto, deskripsi,
  tambah/hapus anggota)
- **1.11.0–1.11.1**: cari pesan + halaman Berbintang; auto-load riwayat saat scroll ke atas
  tanpa baris sidebar yang mengganggu
- **1.10.0–1.10.1**: diet RAM — RSS tetap di bawah 100 MB bahkan setelah sesi berat
- **1.9.2**: perbaikan final drag & drop — data drop dibaca mentah berbasis panjang
  (ternyata `GetText()` mengembalikan NULL untuk tipe `text/uri-list`); keterangan kilat
  bila drop tidak berisi berkas yang bisa dikirim.
- **1.9.1**: perbaikan crash (SIGSEGV) saat drag & drop akibat regresi pustaka gotk3
  (`GetURIs()`); "kotak hitam" `crash.log` di `~/.local/share/obrol/` + pengaman per-drop
  supaya aplikasi tidak mati.
- **1.9.0**: drag & drop berkas ke percakapan; presence ala WhatsApp (online, terakhir
  dilihat, "*Budi sedang mengetik…*" lengkap dengan nama pengetik di grup; status mengetik
  Anda juga dikirim); paket `.deb` di halaman Releases.
- **1.8.0**: cakupan Bahasa Inggris penuh (±90 string) + tautan donasi Ko-fi di dialog Tentang.
- **1.7.0**: profil kontak/grup (foto, about, info grup) + link donasi Saweria
- **1.6.0**: arsip chat (folder Diarsipkan, simpan permanen)
- **1.5.0**: pesan suara (rekam/kirim/putar) + notifikasi panggilan
- **1.4.0**: balas pesan (kutipan), reaksi emoji, teruskan pesan, dan pilihan bahasa Inggris
- **1.3.0**: stiker penuh (tampil + animasi, panel Terbaru/Favorit, buat stiker dari gambar/screenshot),
  logo & ikon SVG asli buatan sendiri
- **1.2.0**: tempel screenshot/gambar dari clipboard dengan Ctrl+V, pesan multi-baris dengan Shift+Enter, edit pesan
  (klik kanan atau tombol ↑), hapus untuk semua orang, sematkan pesan (banner + lompat ke pesan), sematkan obrolan,
  pratinjau foto kiriman sendiri lebih tajam, dan kredit pembuat (Katon Bagaskoro) di menu Tentang.
- **1.1.0**: dark mode (otomatis/terang/gelap), mute per chat yang tersinkron dengan ponsel, bisukan semua grup
  kecuali @mention/balasan, badge mute, status sementara di header kembali otomatis ke status koneksi.
  Update dari 1.0.0 cukup dengan `bash install.sh`; sesi login dan riwayat chat tetap aman.
- **1.0.0**: rilis pertama.

## Pemecahan masalah

| Masalah | Solusi |
|---|---|
| Ikon tray tidak muncul | Panel XFCE perlu plugin **Status Tray** (Klik kanan panel → Panel → Add New Items → *Status Tray Plugin*). Tanpa tray, menutup jendela akan keluar dari aplikasi. |
| Tidak ada suara notifikasi | `sudo apt install gnome-session-canberra` (menyediakan `canberra-gtk-play`) |
| Emoji tampil kotak | `sudo apt install fonts-noto-color-emoji` |
| Ingin login ulang / sesi rusak | Menu ☰ → Keluar dari akun, atau hapus `~/.local/share/obrol/` |
| Muncul "Sesi dibuka di tempat lain" | Obrol sedang dijalankan di komputer lain dengan sesi yang sama |
| Aplikasi pernah crash sendiri | Kirim isi `~/.local/share/obrol/crash.log` — "kotak hitam" otomatis yang merekam jejak panic |
| Drag & drop tidak melakukan apa-apa | Pastikan versi ≥ 1.9.2 (`obrol --version`), lalu jatuhkan berkas ke area percakapan yang terbuka |
| RAM membesar setelah lama dipakai | Sejak 1.9.3 cache stiker animasi dibatasi (±4 MB/stiker, 24 stiker); restart Obrol melepaskan cache yang sudah terlanjur penuh |

## Struktur kode

```
main.go    entry point, opsi CLI, konfigurasi, data demo
wa.go      koneksi whatsmeow: login, event, riwayat, kirim/terima, media
store.go   database SQLite lokal (chat + pesan)
ui.go      seluruh antarmuka GTK3
features.go kotak ketik multi-baris, tempel gambar, edit/hapus pesan, sematkan pesan & obrolan
tray.go    binding cgo kecil untuk GtkStatusIcon (tray XFCE)
clip.go    binding cgo kecil untuk membaca file yang di-copy dari file manager
icons.go   ikon & logo SVG bawaan (widget DrawingArea yang bisa di-recolor sesuai tema)
stickers_core.go penyimpanan, unduh/kirim, dan encode WebP stiker (dlopen libwebp)
stickers.go panel stiker, pratinjau animasi, dan dialog buat stiker
webp.go    decoder/encoder WebP (termasuk WebP animasi) lewat cgo libwebp
assets/    ikon & logo SVG, stiker demo
```

Obrol © 2026 Katon Bagaskoro. Lisensi kode: MIT. whatsmeow: MPL-2.0. gotk3: ISC.
