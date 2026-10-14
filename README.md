# Obrol — a truly native WhatsApp client for Linux

**English** · **[Bahasa Indonesia](README.id.md)**

[![build](https://github.com/GANTI-USERNAME/obrol-wa/actions/workflows/build.yml/badge.svg)](https://github.com/GANTI-USERNAME/obrol-wa/actions/workflows/build.yml)
[![release](https://img.shields.io/github/v/release/GANTI-USERNAME/obrol-wa?display_name=tag)](https://github.com/GANTI-USERNAME/obrol-wa/releases/latest)
[![license](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![go](https://img.shields.io/badge/Go-1.26-00ADD8.svg?logo=go&logoColor=white)](go.mod)

Obrol is a WhatsApp desktop client that is **truly native** on Linux (built for Linux Mint XFCE, but also runs on
Ubuntu, Debian, MATE, Cinnamon, …). No browser, no Electron, no WebView: the UI is pure **GTK3**, and the connection
talks directly to WhatsApp's servers through [whatsmeow](https://github.com/tulir/whatsmeow), a Go implementation of
the multi-device WhatsApp protocol.

Made by **Katon Bagaskoro**.

![Dark mode](screenshots/gelap-grup.png)

| Light mode | Mute (right-click a chat) | Login |
|---|---|---|
| ![Light](screenshots/terang.png) | ![Mute](screenshots/menu-mute.png) | ![Login](screenshots/login.png) |

**New in 1.3 — stickers & original artwork:**

| Stickers (animated ones too) | Sticker panel: Recent / Favorites / Create | Make stickers from photos or screenshots |
|---|---|---|
| ![Stickers](screenshots/v13-stiker.png) | ![Panel](screenshots/v13-panel.png) | ![Create](screenshots/v13-buat.png) |

## Why is it so light?

| | Obrol | WhatsApp Web in a browser | Electron clients |
|---|---|---|---|
| Technology | GTK3 + Go (single binary) | Chrome/Firefox tab | wrapped Chromium |
| Size | ±19 MB | – | 150–250 MB |
| RAM | ±60–105 MB (measured) | 300–600 MB | 300–500 MB |
| Processes | 1 | many | many |

## Features

- ✅ Login with a **QR code** or an **8-character pairing code** from your phone
- ✅ Chat list with initial avatars, message previews, timestamps and **unread badges**
- ✅ **History sync** from the phone on first login
- ✅ Send & receive text, emoji, and clickable links
- ✅ **Ticks**: ✓ sent, ✓✓ delivered, blue ✓✓ read — read receipts are sent back too
- ✅ Photo/video thumbnails; click to download and open in your default app
- ✅ Open documents, voice notes and audio (saved to `~/Downloads/Obrol`)
- ✅ **Stickers**, rendered exactly like on the phone — including **animated WebP stickers**
  (custom animated-WebP decoder via cgo libwebp), with transparent bubbles like real WhatsApp
- ✅ **Sticker panel** next to the emoji button: *Recent* (last used), *Favorites* and *Create* tabs.
  Click to resend; right-click to favorite or hide
- ✅ **Create your own stickers** from any image or a clipboard screenshot: automatic square crop,
  rounded corners, white outline, encoded as WebP ≤100 KB (WhatsApp's sticker format)
- ✅ **Send photos & files** with a caption; **paste images with Ctrl+V** (PrtSc screenshots,
  Flameshot, or files copied from Thunar/Nemo) with a preview dialog
- ✅ **Multi-line messages**: `Enter` sends, `Shift+Enter` newline; the input box grows to ±7 lines
- ✅ **Edit sent messages** (15-minute window, like WhatsApp); edited messages are labelled
- ✅ **Delete for everyone** (±2.5 days)
- ✅ **Pin messages** for 24 h / 7 days / 30 days with a banner above the conversation
- ✅ **Pin up to 3 chats** to the top of the list, synced with the phone
- ✅ **Group chats** with colored sender names
- ✅ Messages deleted/edited by others update live
- ✅ Locations (with Google Maps link), contacts and polls shown as text
- ✅ **Desktop notifications** (click opens the chat) with optional sound
- ✅ WhatsApp-style **dark mode**: *Automatic* follows the XFCE theme, or force *Light*/*Dark*
- ✅ **Mute** chats/groups for 8 h / 1 week / forever, **synced two-way with the phone**;
  mentions and replies still notify; "mute all groups" option
- ✅ **System tray icon** with a red dot for unread messages; closing the window hides it to the tray
- ✅ **Autostart** on login (hidden in the tray)
- ✅ Start a chat with any number (checked against WhatsApp first)
- ✅ Search, date separators (Today/Yesterday/weekday), older history loads on scroll-up
- ✅ **Original hand-drawn SVG artwork**: the "Ngobrol" logo and 27 thin-line icons that recolor
  themselves to match the theme
- ✅ Single instance: launching Obrol again just raises the existing window

**Shortcuts:** `Enter` send · `Shift+Enter` newline · `Ctrl+V` paste image/text · `↑` edit last message ·
`Ctrl+F` search · `Ctrl+N` new chat · `Esc` cancel edit / close chat · `Ctrl+W` hide · `Ctrl+Q` quit

## Installation (Linux Mint 21/22, x86_64)

A prebuilt binary is shipped, so **you don't need Go**. Download the latest tarball from
[Releases](https://github.com/GANTI-USERNAME/obrol-wa/releases/latest), then:

```bash
tar xzf obrol-1.3.0-linux-x86_64.tar.gz
cd obrol
bash install.sh
```

Then launch it from **Menu → Internet → Obrol**, or run `obrol` in a terminal.
The install is per-user (no sudo needed):

- Binary: `~/.local/bin/obrol`
- Menu entry: `~/.local/share/applications/obrol.desktop`
- Data (login session + chat history): `~/.local/share/obrol/`

Uninstall:

```bash
bash install.sh --uninstall   # removes the app, keeps chat data
bash install.sh --purge       # removes the app + login session + chat history
```

> To also remove the device from your phone's *Linked devices* list, choose **Menu ☰ → Log out** before uninstalling.

## Building from source (optional)

```bash
git clone https://github.com/Katon17/obrol-wa.git
cd obrol-wa
sudo apt install build-essential pkg-config libgtk-3-dev golang-go
./build.sh        # first build ±3–6 minutes (the GTK bindings are big)
./install.sh
```

Go 1.22+ is enough; the 1.26 toolchain required by whatsmeow is downloaded automatically.

Command-line options:

| Option | Effect |
|---|---|
| `--demo` | demo mode with sample data, no connection |
| `--demo-login` | show the sample login page |
| `--debug` | log whatsmeow traffic to the terminal |
| `--hidden` | start hidden in the system tray |
| `--version` | print the version |

## Project layout

```
main.go    entry point, CLI options, config, demo data
wa.go      whatsmeow connection: login, events, history, send/receive, media
store.go   local SQLite database (chats + messages)
ui.go      the whole GTK3 interface
features.go multi-line input, image paste, edit/delete, pin message & chat
icons.go   bundled SVG icons & logo (a recolorable DrawingArea widget)
stickers_core.go sticker storage, download/send, WebP encoding (dlopen libwebp)
stickers.go sticker panel, animation preview, sticker creation dialog
webp.go    WebP decoder/encoder (incl. animated WebP) via cgo libwebp
tray.go    tiny cgo binding for GtkStatusIcon (XFCE tray)
clip.go    tiny cgo binding to read files copied from the file manager
assets/    app icon & logo SVGs, icon set, demo stickers
```

## Contributing

Bug reports, feature ideas and pull requests are welcome — see [CONTRIBUTING.md](CONTRIBUTING.md).
Please follow the [Code of Conduct](CODE_OF_CONDUCT.md). Full version history: [CHANGELOG.md](CHANGELOG.md).

## Disclaimer

Obrol is an independent open-source project and is **not affiliated with or endorsed by
WhatsApp or Meta**. It uses the unofficial multi-device protocol implementation from whatsmeow.

## License

MIT — see [LICENSE](LICENSE). © 2026 Katon Bagaskoro.
whatsmeow: MPL-2.0 · gotk3: ISC.
