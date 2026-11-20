# Changelog

All notable changes to Obrol are documented in this file.
The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.12.2] — 2026-11-20

### Changed
- Perpustakaan whatsmeow diperbarui ke revisi 2026-11-16 (penanganan stanza
  status & stabilitas koneksi), tanpa perubahan perilaku aplikasi. /
  whatsmeow updated to the 2026-11-16 revision (status-stanza handling &
  connection stability) with no behaviour change in the app.

### Removed
- Percobaan fitur Status/Story ditarik kembali: perangkat tertaut
  (non-primér) tidak menerima riwayat status yang sudah lewat maupun
  gelontoran offline, sehingga halaman Status tidak dapat diisi secara
  andal. Lebih baik tidak ada daripada setengah jalan. /
  The Status/Story attempt is withdrawn: linked (non-primary) devices
  receive neither past status history nor an offline flush, so the Status
  page could not be filled reliably. Better absent than half-working.

## [1.12.1] — 2026-11-20

### Fixed
- **Deskripsi grup sangat panjang** tidak lagi mendorong daftar peserta keluar layar:
  deskripsi kini dikurung guliran setinggi ±4 baris (dan disembunyikan bila kosong). /
  Very long group descriptions no longer push the participant list off-screen: the
  description is capped in a ±4-line scroll area (hidden when empty).

### Added
- **Foto grup tampil di halaman info** (diunduh preview via URL profil, dilepas lagi
  saat halaman ditutup). / **Group photo shown on the info page** (preview downloaded
  via profile URL, released again on leave).

### Changed
- Kepala halaman info lebih padat (kembali | foto 84px | judul ter-elipsize) dan aksi
  manajemen kini tombol **berlabel** (Subjek · Foto · Deskripsi · Tambah peserta) yang
  lega diklik, bukan ikon mungil. / Compact info header (back | 84px photo | ellipsized
  title) and management actions are now roomy **labeled buttons** instead of tiny icons.

## [1.12.0] — 2026-11-20

### Added
- **Info & manajemen grup**: klik header grup membuka halaman info — daftar peserta
  (dimuat per 50 via gulir, dibongkar saat keluar), ganti subjek/foto/deskripsi grup,
  tambah peserta (validasi nomor), dan klik kanan peserta → «Keluarkan dari grup»
  bagi admin. / **Group info & management**: clicking a group header opens the info
  page — participant list (paginated 50 via scroll, torn down on leave), change group
  subject/photo/description, add participants (number validation), and right-click a
  participant → "Remove from group" for admins.

## [1.11.1] — 2026-11-20

### Changed
- Baris «Tampilkan percakapan lain (N lagi)» dihapus: batch sidebar berikutnya (+60)
  kini dimuat otomatis saat guliran menyentuh dasar daftar — tidak ada lagi baris
  menetap yang mengganggu di daftar chat. / The "Show more chats (N more)" row is gone:
  the next sidebar batch (+60) now loads automatically when scrolling reaches the
  bottom — no more permanent row cluttering the chat list.

## [1.11.0] — 2026-11-20

### Added
- **Pencarian pesan**: baris «Cari pesan: …» muncul di sidebar saat kolom cari terisi
  (hasil global lintas chat), dan **Ctrl+Shift+F** membuka baris cari-di-chat untuk
  menyaring percakapan yang sedang terbuka (Esc kembali). /
  **Message search**: a "Search messages: …" row appears in the sidebar while the filter
  box has text (global cross-chat results), and **Ctrl+Shift+F** opens an in-chat find bar
  (Esc returns).
- **Pesan berbintang**: klik kanan pesan → «Bintangi pesan» (★ muncul di meta pesan);
  menu utama → «Pesan berbintang» membuka daftarnya. / **Starred messages**: right-click a
  message → "Star message" (★ shown in message meta); main menu → "Starred messages" opens
  the list.
- Ikon baru `search.svg` sesuai gaya garis tipis. / New thin-stroke `search.svg` icon.

### Changed
- Halaman hasil & berbintang dibangun malas dan **dibongkar saat ditinggalkan**
  (baris chat tersembunyi ikut dilepas) sehingga fitur baru tidak menambah RSS menetap. /
  Results & starred pages are built lazily and **torn down on leave** (hidden chat rows are
  released too) so the new features add no permanent RSS.

## [1.10.1] — 2026-11-20

### Changed
- **Plafon sidebar 60 percakapan** (+ baris «Tampilkan percakapan lain (N lagi)»):
  ratusan chat tidak lagi menjadi ratusan widget permanen — potongan terbesar
  baseline RSS selama ini. Chat yang sedang terbuka selalu punya baris.
  / **Sidebar capped at 60 chats** (+ a "Show more chats (N more)" row): hundreds of
  chats no longer mean hundreds of permanent widgets — the largest baseline RSS
  chunk all along. The open chat always keeps its row.
- Anggaran stiker animasi 1,5 MB → 1 MB per stiker (cache 5 → plafon ±5 MB). /
  Animated-sticker budget 1.5 MB → 1 MB per sticker (cache 5 → ±5 MB cap).
- Mode demo kini menyertakan 300 chat uji untuk memperlihatkan plafon sidebar. /
  Demo mode now seeds 300 test chats to showcase the sidebar cap.

## [1.10.0] — 2026-11-20

### Changed
- **Kalibrasi akhir berbasis log lapangan**: cache stiker animasi 8 → 5 (log menunjukkan
  cache selalu penuh; plafon baru ±7,5 MB), halaman riwayat 30 → 24 baris dengan plafon
  baris hidup 72, dan pemilih emoji kini **berhalaman 64 sel** (+ tombol «⋯») sehingga
  menggulir kategori tidak memetakan ratusan glyph font emoji berwarna ke RSS sekaligus.
  / **Final log-driven calibration**: sticker cache 8 → 5 (field logs showed it always
  full; new cap ±7.5 MB), history page 30 → 24 rows with a 72 live-row cap, and the emoji
  picker is now **paginated at 64 cells** (+ «⋯» button) so browsing a category no longer
  maps hundreds of color-emoji glyphs into RSS at once.

## [1.9.9] — 2026-11-20

### Changed
- Threshold mmap/trim malloc diturunkan 128 KB → 64 KB: permukaan glyph font emoji
  berwarna kini ikut jalur mmap dan langsung kembali ke OS, mengurangi fragmentasi
  arena setelah sesi berat. / malloc mmap/trim thresholds lowered 128 KB → 64 KB so
  color-emoji glyph surfaces take the mmap path and return straight to the OS,
  cutting arena fragmentation after heavy sessions.
- mem.log kini ikut mencatat `rows=` (jumlah baris pesan hidup) untuk diagnostik
  sesi berat. / mem.log now also records `rows=` (live message-row count) for
  heavy-session diagnostics.

## [1.9.8] — 2026-11-19

### Changed
- **Pemilih emoji baru milik obrol** menggantikan pemilih bawaan GTK: kategori dimuat
  malas dan semua sel dihancurkan saat popover tertutup. Pemilih GTK membangun ribu-an
  widget emoji permanen + cache glyph font emoji berwarna saat digulir — penyumbang
  besar lonjakan RSS pemakaian berat. /
  **New built-in emoji picker** replaces the GTK chooser: categories load lazily and all
  cells are destroyed when the popover closes. The GTK chooser built thousands of
  permanent emoji widgets plus a color-emoji glyph cache while scrolling — a major
  contributor to heavy-use RSS spikes.
- **Panel stiker dibongkar saat tertutup**: sel, pixbuf, dan memori C-nya dikembalikan ke
  OS; dibangun ulang murah saat dibuka lagi. / **Sticker panel tears down on close**:
  cells, pixbufs and their C memory return to the OS; cheaply rebuilt on reopen.
- **Plafon 120 baris hidup per chat**: gulir mendalam tidak menumpuk halaman tanpa batas;
  pemuatan dijeda dengan pemberitahuan status. / **120 live-row cap per chat**: deep
  scrolling no longer stacks pages unbounded; loading pauses with a status notice.

## [1.9.7] — 2026-11-19

### Changed
- **RSS tidak lagi menempel di puncak**: strategi arena malloc disetel saat start
  (`M_ARENA_MAX=2`, threshold mmap/trim tetap 128 KB) sehingga memori C yang dilepas
  (pixbuf, cache font) benar-benar kembali ke OS, bukan tertahan sebagai high-water mark. /
  **RSS no longer sticks at its peak**: malloc arena strategy set at startup
  (`M_ARENA_MAX=2`, fixed 128 KB mmap/trim thresholds) so freed C memory (pixbufs, font
  caches) is returned to the OS instead of being retained as a high-water mark.
- Thumbnail baris didekode langsung pada ukuran tampil (260px, sekali scale; sebelumnya
  dua langkah 340→260px) dan satu halaman riwayat 30 baris. /
  Row thumbnails decode straight at display size (260px, single scale; previously two
  steps 340→260px) and a history page is 30 rows.
- Log diagnostik `mem.log` (satu baris/menit: RSS, heap Go, jumlah GC, cache stiker) di
  folder data untuk pelacakan lapangan. / Diagnostic `mem.log` (one line/min: RSS, Go heap,
  GC count, sticker cache) in the data folder for field tracking.

## [1.9.6] — 2026-11-19

### Changed
- **Mode hemat memori**: thumbnail media di baris pesan kini didekode seperlunya saja
  (≤340px, sebelumnya ukuran tersimpan 520px ≈ 0,8 MB/baris); satu halaman riwayat 40 baris
  (sebelumnya 80); cache stiker animasi 8 stiker × ±1,5 MB; dan trim otomatis tiap 60 detik.
  GC juga dibuat lebih rajin (GOGC 50). Kombinasi ini menahan RSS pemakaian wajar di bawah ±100 MB pada sesi besar. /
  **Memory-saver mode**: row thumbnails decode at display size (≤340px instead of the stored
  520px ≈ 0.8 MB/row); history page 40 rows (was 80); sticker cache 8 × ±1.5 MB; automatic
  trim every 60 s and a more eager GC (GOGC 50) — keeping typical RSS under ±100 MB on large sessions.

## [1.9.5] — 2026-11-18

### Changed
- **RSS benar-benar turun setelah ganti chat**: setelah GC, `malloc_trim(0)` memaksa glibc
  mengembalikan memori C yang sudah bebas ke OS (tanpa ini RSS menetap tinggi walau objek
  sudah dilepas). Cache stiker animasi juga diperketat: 12 stiker × anggaran ±2 MB
  (sebelumnya 24 × 4 MB). / **RSS now actually drops after switching chats**: `malloc_trim(0)`
  returns freed C memory to the OS after GC; animated-sticker cache tightened to 12 stickers
  × ±2 MB budget.

## [1.9.4] — 2026-11-18

### Fixed
- **RAM naik terus tiap ganti chat**: pixbuf thumbnail/stiker baris pesan hidup di memori C
  dan hanya dibebaskan finalizer saat GC Go berjalan — padahal GC tidak terpicu oleh alokasi
  C, sehingga bekas percakapan lama menumpuk. Kini satu siklus GC dipaksa saat daftar pesan
  dibongkar (ganti chat) dan saat cache stiker mengeviksi entri; RSS kembali turun setelah
  pindah chat. / **RAM kept growing per chat switch**: message-row pixbufs live in C memory and
  are only freed by finalizers when the Go GC runs — which C allocations never trigger. A GC
  cycle is now forced when the message list is torn down and when the sticker cache evicts.

## [1.9.3] — 2026-11-18

### Changed
- **RAM stiker animasi dibatasi**: dahulu satu stiker animasi bisa menyimpan sampai 150
  frame penuh di memori dan cache menampung 60 stiker — pada sesi ramai stiker ini bisa
  memakan ratusan MB (teramati ±636 MB). Kini ada anggaran ±4 MB per stiker (frame berlebih
  dipotong) dan cache cukup 24 stiker; pemakaian RAM kembali wajar. / **Animated-sticker RAM
  cap**: one animated sticker could keep up to 150 full frames in memory and the cache held
  60 stickers — hundreds of MB on sticker-heavy sessions (observed ±636 MB). Now each sticker
  gets a ±4 MB frame budget and the cache holds 24 stickers.

## [1.9.2] — 2026-11-17

### Fixed
- **Drag & drop tidak mengirim apa pun di 1.9.1**: perbaikan sebelumnya memakai
  `GetText()`, yang ternyata mengembalikan NULL untuk tipe drop `text/uri-list`.
  Kini data drop dibaca mentah berbasis panjang (`GetData`) lalu diurai sendiri —
  bebas dari dua fungsi gotk3 yang bermasalah. / **Drag & drop sent nothing in 1.9.1**:
  the previous fix used `GetText()`, which returns NULL for `text/uri-list` drops;
  the raw drop data is now read by length (`GetData`) and parsed locally.

## [1.9.1] — 2026-11-17

### Fixed
- **Crash saat drag & drop (SIGSEGV)**: penyebabnya regresi di pustaka gotk3 —
  `SelectionData.GetURIs()` membaca larik pointer dengan aritmetika +1 byte sehingga
  membaca memori tak selaras dan segfault. Obrol kini mengurai teks URI sendiri
  (`selectionURIs`), tanpa menyentuh fungsi yang cacat itu. / **Drag & drop crash
  (SIGSEGV)**: caused by an upstream gotk3 regression (`GetURIs` mis-parses the pointer
  array); Obrol now parses the URI text itself.
- **Kotak hitam crash**: bila panic terjadi, jejak lengkap disimpan ke
  `~/.local/share/obrol/crash.log` untuk memudahkan pelaporan; panic saat memproses
  drop juga tidak lagi mematikan aplikasi (dialog error muncul). / **Crash black box**:
  panics are now logged to `~/.local/share/obrol/crash.log`; a panic while handling a
  drop shows an error dialog instead of killing the app.

## [1.9.0] — 2026-11-17

### Added
- **Drag & drop berkas**: jatuhkan satu atau beberapa berkas (foto, dokumen, video…) dari
  pengelola berkas langsung ke area percakapan untuk mengirimnya. / **Drag & drop files**:
  drop one or several files (photos, documents, videos…) from your file manager straight
  into a conversation to send them.
- **Paket `.deb`**: rilis kini menyediakan paket Debian/Ubuntu
  (`obrol_1.9.0-1_amd64.deb`) — instal cukup dengan `sudo apt install ./obrol_….deb`;
  bisa juga dibangun lokal lewat `build-deb.sh` (kontribusi @SkoZombie) dan workflow
  rilis otomatis melampirkan `.deb` pada setiap tag. / **Debian package**: releases now
  ship `obrol_1.9.0-1_amd64.deb` — install with a single `sudo apt install ./obrol_….deb`;
  or build one locally via `build-deb.sh` (contributed by @SkoZombie); the release
  workflow attaches a `.deb` to every tag automatically.
- **Presence ala WhatsApp**: header chat kini menampilkan *online*, *terakhir dilihat
  hari kemarin, 21.04*, serta indikator *sedang mengetik…* — di grup lengkap dengan nama
  pengetiknya (mis. "Budi sedang mengetik…"). Aplikasi juga mengirim status "sedang
  mengetik" ke kontak saat Anda mengetik. / **WhatsApp-style presence**: the chat header
  now shows *online*, *last seen yesterday, 21:04*, and a *typing…* indicator — in groups
  including the sender's name ("Budi is typing…"). The app also broadcasts your own
  typing status to the contact while you type.

### Fixed
- String "sampai …" pada banner pesan tersemat kini ikut diterjemahkan. /
  The "until …" suffix on the pinned-message banner is now translatable.

## [1.8.0] — 2026-11-03

### Changed
- **Cakupan Bahasa Inggris penuh**: ±90 string yang dulu lolos (halaman login/pairing,
  petunjuk kosong, tooltip, status koneksi, menu dialog stiker, tombol dialog, deskripsi
  bisu/arsip, dll.) kini ikut diterjemahkan; audit otomatis menjamin setiap `T()` punya
  padanan EN. / **Full English coverage**: ~90 leftover strings (login/pairing page, empty
  hint, tooltips, connection statuses, sticker dialog, dialog buttons, mute/pin texts…)
  are now translated; an automated audit guarantees every `T()` key has an EN counterpart.
- **Donasi**: dialog *Tentang Obrol* kini memuat dua tautan — Saweria dan
  [ko-fi.com/katonbagaskoro](https://ko-fi.com/katonbagaskoro).

## [1.7.0] — 2026-11-02

### Added
- **Profil kontak & grup**: klik header chat (atau klik kanan chat → *Lihat profil*) membuka
  dialog profil: foto profil (diunduh & di-cache otomatis), nama, nomor telepon, teks
  *about/status* kontak, serta untuk grup: jumlah peserta dan deskripsi grup.
  Tombol *Salin nomor* untuk kontak. / **Contact & group profile**: click the chat header
  (or right-click → View profile) for photo (auto-cached), phone, about text, and for
  groups participant count + description; Copy-number button included.
- **Tautan donasi**: dialog *Tentang Obrol* kini memuat tautan
  [saweria.co/Kabako](https://saweria.co/Kabako) ("Dukung pengembangan via Saweria").

## [1.6.0] — 2026-10-18

### Added
- **Arsip chat**: klik kanan chat → *Arsipkan obrolan*; chat pindah ke folder **Diarsipkan**
  (baris khusus di puncak daftar dengan jumlah chat). Di dalamnya ada baris
  *Kembali ke obrolan aktif* dan menu *Kembalikan dari arsip*. Pesan baru di chat
  yang diarsipkan tidak menghitung badge judul; arsip tersimpan permanen di basis data.
  / **Chat archive**: right-click → Archive; archived chats live under a collapsible
  **Archived** row with a back row and Unarchive menu; state persists across restarts.

## [1.5.0] — 2026-10-17

### Added
- **Pesan suara (VN)**: tombol mik di composer merekam dari mikrofon (ffmpeg → opus/ogg,
  fallback pulseaudio/ALSA), klik lagi untuk berhenti & kirim; bubble VN punya tombol putar
  dalam aplikasi (mpv/ffplay/paplay/vlc). VN dari kontak diputar tanpa meninggalkan aplikasi.
  / **Voice messages**: mic button records (ffmpeg opus, pulse/ALSA fallback), click again to
  stop & send; incoming VNs play in-app.
- **Notifikasi panggilan**: peristiwa panggilan WhatsApp (masuk/tak terjawab/berakhir) tampil
  sebagai baris 📞 di chat, notifikasi desktop + bunyi, dan status header saat berdering.
  Menjawab tetap di ponsel — protokol unofficial tidak mendukung audio panggilan.
  / **Call events**: incoming/missed calls show as 📞 rows with desktop notifications.

## [1.4.1] — 2026-10-16

### Fixed
- Menu klik kanan: item **Reaksi** tampil abu-abu dan submenu emoji tidak bisa dibuka
  (item dibuat dengan penangan kosong sehingga dianggap nonaktif). Sekarang submenu
  reaksi terbuka normal. / Right-click menu: the **React** item was greyed out and its
  emoji submenu could not open; fixed.

## [1.4.0] — 2026-10-16

### Added
- **Reply**: right-click → Reply shows a quote bar above the composer; sent/received replies render a
  quoted bubble (sender color + snippet); clicking the quote jumps to the original message.
- **Reactions**: right-click → React with six quick emojis; reaction chips (emoji + count, highlighted
  when yours) render under bubbles; click a chip to toggle your reaction. Incoming reactions and
  reactions made on the phone are applied live.
- **Forward**: right-click → Forward opens a searchable chat picker; text, stickers and media
  (download + re-upload) are forwarded and labelled *Forwarded* / *Diteruskan*.
- **Bilingual UI**: Menu → Language switches between Bahasa Indonesia and English (app auto-restarts);
  ~200 strings translated, including menus, dialogs, tooltips, dates and kind labels.

## [1.3.0] — 2026-10-11

### Added
- **Stickers**: receive and render stickers exactly like on the phone, including **animated WebP stickers**
  (custom animated-WebP decoder via cgo libwebp). Sticker bubbles are transparent, like real WhatsApp.
- **Sticker panel** next to the emoji button with three tabs: *Recent* (last used), *Favorites*, and *Create*.
  Click to resend; right-click to favorite or hide.
- **Create your own stickers** from any image file or a screenshot in the clipboard: automatic square crop,
  rounded corners, white outline, live preview on a checkerboard, encoded as WebP ≤100 KB (WhatsApp's sticker format).
- **"Make sticker…"** button directly inside the image-send dialog.
- **Original artwork**: the "Ngobrol" logo (two chat bubbles) and 27 thin-line icons, all hand-drawn SVGs that
  stay sharp at any scale and recolor themselves to match the light/dark theme.
- Six demo stickers (including an animated waving mascot) embedded for `--demo` mode.

### Changed
- All in-app emoji-as-icons replaced with the new SVG icon set (header, composer, menus, chat list, tray).
- About dialog simplified.

## [1.2.0]

### Added
- Paste screenshots/images from the clipboard with `Ctrl+V` (PrtSc, xfce4-screenshooter, Flameshot,
  or files copied from Thunar/Nemo), with preview and caption dialog.
- Multi-line messages: `Enter` sends, `Shift+Enter` inserts a newline; the input box grows up to ±7 lines.
- Edit sent messages (15-minute window, like WhatsApp): right-click → Edit, or `↑` on an empty input box.
- Pin messages for 24 h / 7 days / 30 days, with a banner above the conversation (click to jump).
- Pin up to 3 chats to the top of the chat list (synced with the phone).

## [1.1.0]

### Added
- WhatsApp-style **dark mode**: *Automatic* follows the XFCE theme, or force *Light*/*Dark*.
- **Mute** chats/groups for 8 h / 1 week / forever, synced two-way with the phone;
  mentions and replies still notify. "Mute all groups" option.
- Muted chats: bell-off icon, grey badge, excluded from the window title and tray dot.

## [1.0.0]

### Added
- First public release: truly native GTK3 WhatsApp client (no Electron, no browser, no WebView),
  powered by [whatsmeow](https://github.com/tulir/whatsmeow).
- QR or pairing-code login, chat list with unread badges, history sync, text/emoji/link messages,
  read receipts, photo/video thumbnails, documents, voice notes, locations, group chats,
  desktop notifications with sound, system tray with unread dot, autostart, single instance.

[1.12.2]: https://github.com/Katon17/obrol-wa/releases/tag/v1.12.2
[1.12.1]: https://github.com/Katon17/obrol-wa/releases/tag/v1.12.1
[1.12.0]: https://github.com/Katon17/obrol-wa/releases/tag/v1.12.0
[1.11.1]: https://github.com/Katon17/obrol-wa/releases/tag/v1.11.1
[1.11.0]: https://github.com/Katon17/obrol-wa/releases/tag/v1.11.0
[1.10.1]: https://github.com/Katon17/obrol-wa/releases/tag/v1.10.1
[1.10.0]: https://github.com/Katon17/obrol-wa/releases/tag/v1.10.0
[1.9.9]: https://github.com/Katon17/obrol-wa/releases/tag/v1.9.9
[1.9.8]: https://github.com/Katon17/obrol-wa/releases/tag/v1.9.8
[1.9.7]: https://github.com/Katon17/obrol-wa/releases/tag/v1.9.7
[1.9.6]: https://github.com/Katon17/obrol-wa/releases/tag/v1.9.6
[1.9.5]: https://github.com/Katon17/obrol-wa/releases/tag/v1.9.5
[1.9.4]: https://github.com/Katon17/obrol-wa/releases/tag/v1.9.4
[1.9.3]: https://github.com/Katon17/obrol-wa/releases/tag/v1.9.3
[1.9.2]: https://github.com/Katon17/obrol-wa/releases/tag/v1.9.2
[1.9.1]: https://github.com/Katon17/obrol-wa/releases/tag/v1.9.1
[1.9.0]: https://github.com/Katon17/obrol-wa/releases/tag/v1.9.0
[1.8.0]: https://github.com/Katon17/obrol-wa/releases/tag/v1.8.0
[1.7.0]: https://github.com/Katon17/obrol-wa/releases/tag/v1.7.0
[1.6.0]: https://github.com/Katon17/obrol-wa/releases/tag/v1.6.0
[1.5.0]: https://github.com/Katon17/obrol-wa/releases/tag/v1.5.0
[1.4.1]: https://github.com/Katon17/obrol-wa/releases/tag/v1.4.1
[1.4.0]: https://github.com/Katon17/obrol-wa/releases/tag/v1.4.0
[1.3.0]: https://github.com/Katon17/obrol-wa/releases/tag/v1.3.0
[1.2.0]: https://github.com/Katon17/obrol-wa/releases/tag/v1.2.0
[1.1.0]: https://github.com/Katon17/obrol-wa/releases/tag/v1.1.0
[1.0.0]: https://github.com/Katon17/obrol-wa/releases/tag/v1.0.0
