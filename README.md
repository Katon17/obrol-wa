# Obrol — a truly native WhatsApp client for Linux

**English** · **[Bahasa Indonesia](README.id.md)**

[![build](https://github.com/Katon17/obrol-wa/actions/workflows/build.yml/badge.svg)](https://github.com/Katon17/obrol-wa/actions/workflows/build.yml)
[![release](https://img.shields.io/github/v/release/Katon17/obrol-wa?display_name=tag)](https://github.com/Katon17/obrol-wa/releases/latest)
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

**New in 1.4 — replies, reactions, forwarding & English:**

| Reply with quoted bubble + emoji reactions | Forward messages to any chat | Bahasa Indonesia / English UI |
|---|---|---|
| ![Reply](screenshots/v14-reply.png) | ![Forward](screenshots/v14-forward.png) | ![English](screenshots/v14-english.png) |

**New in 1.5 — voice messages & call events:**

| Record & play voice messages | Missed-call rows in chat |
|---|---|
| ![VN](screenshots/v15-pesan-suara.png) | ![Calls](screenshots/v15-panggilan.png) |

**New in 1.6 — chat archive:**

| "Archived" row in the chat list | Archive contents + back row |
|---|---|
| ![Archive](screenshots/v16-arsip.png) | ![Archive view](screenshots/v16-arsip-view.png) |

**New in 1.7 — profiles & donation link:**

| Contact profile | Group profile | Donation in About |
|---|---|---|
| ![Contact](screenshots/v17-profil-kontak.png) | ![Group](screenshots/v17-profil-grup.png) | ![About](screenshots/v18-about-donasi.png) |

**New in 1.9 — drag & drop + presence:**

| Group typing indicator | Online / last seen |
|---|---|
| ![Typing](screenshots/v19-mengetik.png) | ![Online](screenshots/v19-online.png) |

**New in 1.10–1.12 — the leanness & management wave:**

- 🪶 **RAM diet**: peak usage went from ~636 MB to **under 100 MB even after heavy use**
  (SQL-first loading, capped caches, memory handed back to the OS)
- 🔎 **Message search** (Ctrl+F) across all conversations, plus a **Starred** page for
  messages you right-click → *Star*
- 🧑‍🤝‍🧑 **Group info & management**: paged participant list with admin badges, rename the
  group, change its photo & description, add or remove members — all native

**New in 1.12.3–1.13 — language & selection:**

- 🌐 **Automatic language**: the Language menu gains *Automatic (follow system)*,
  tracking your desktop locale (contributed by **baduhai**, PR #7)
- ✅ **Multi-select messages**: right-click → *Select messages…* puts check circles
  beside bubbles; tick several and bulk-**Forward**, **Star**, **Copy text** or
  **Delete** (for me, or for everyone when eligible); Esc leaves the mode
- 🗑 **Delete for me** on a single message, straight from the message menu — no
  selection mode needed

**New in 1.15 — voice notes, two steps:**

- 🎤 **Two-step voice recording**: the second mic click finishes the take and shows a
  preview bar (duration, play, discard, send); send with the send button, another mic
  click, or Enter — discard with the trash or Esc

![Voice note preview](screenshots/v115-vn-preview-en.png)

**New in 1.14 — reach people you never saved:**

- 💬 **Message privately**: right-click a group message → *Message privately* opens a
  1-on-1 chat with its sender; the same action sits in the right-click menu of the
  group member list — their number never has to be in your contacts

![Message privately](screenshots/message-privately.png)

## Why is it so light?

| | Obrol | WhatsApp Web in a browser | Electron clients |
|---|---|---|---|
| Technology | GTK3 + Go (single binary) | Chrome/Firefox tab | wrapped Chromium |
| Size | ±19 MB | – | 150–250 MB |
| RAM | ±73–99 MB (measured) | 300–600 MB | 300–500 MB |
| Processes | 1 | many | many |

## Features

- ✅ **Drag & drop files** straight into a conversation to send them (multiple files at once)
- ✅ **Presence**: *online*, *last seen yesterday, 21:04* and *typing…* in the chat header —
  in groups with the sender's name ("Budi is typing…"); your own typing state is broadcast too
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
- ✅ **Reply to messages** (right-click → Reply): the quoted message appears inside your bubble with the
  sender's color; click the quote to jump to the original
- ✅ **Emoji reactions**: right-click → React (👍 ❤️ 😂 😮 😢 🙏); reaction chips with participant counts
  render under the bubble; click a chip to change/remove yours. Reactions made on the phone sync too
- ✅ **Forward messages** to any chat through a searchable picker; forwarded messages are labelled
  *Forwarded*. Text, stickers and media (download + re-upload) can all be forwarded
- ✅ **Bilingual UI**: Menu ☰ → Language → *Bahasa Indonesia* or *English*; the app restarts to apply
- ✅ **Voice messages**: the mic button records a VN (opus via ffmpeg, PulseAudio/ALSA fallback),
  click again to send; incoming VNs play right inside the app
- ✅ **Call notifications**: incoming/missed calls appear as 📞 rows in the chat plus a desktop
  notification (answering stays on the phone — the unofficial protocol carries no call audio)
- ✅ **Chat archive**: right-click a chat → *Archive chat*; archived conversations collect under an
  **Archived** row (with count) and can be restored any time; the state persists across restarts
- ✅ **Contact & group profile**: click the chat header to see the profile picture (auto-downloaded
  & cached), phone number, about text, and for groups the participant count + description
- ✅ **Full English coverage**: the entire UI (login page, tooltips, connection statuses,
  sticker dialogs, …) is translated — no more Indonesian leftovers in English mode
- ☕ **Support**: both Saweria **and Ko-fi** links live in the *About* dialog
- ✅ **Send photos & files** with a caption; **paste images with Ctrl+V** (PrtSc screenshots,
  Flameshot, or files copied from Thunar/Nemo) with a preview dialog
- ✅ **Multi-line messages**: `Enter` sends, `Shift+Enter` newline; the input box grows to ±7 lines
- ✅ **Edit sent messages** (15-minute window, like WhatsApp); edited messages are labelled
- ✅ **Delete for everyone** (±2.5 days)
- ✅ **Pin messages** for 24 h / 7 days / 30 days with a banner above the conversation
- ✅ **Pin up to 3 chats** to the top of the list, synced with the phone
- ✅ **Group chats** with colored sender names
- ✅ **Search messages** (Ctrl+F) in every conversation; results jump straight to the message
- ✅ **Star messages** and revisit them in a dedicated Starred page
- ✅ **Group info & management**: participants (paged, admin badges), rename, group photo &
  description, add/remove members
- ✅ **Multi-select messages** (right-click → *Select messages…*): tick bubbles and bulk-forward,
  star, copy or delete them; Esc exits
- ✅ **Message privately** from a group message menu or the member list (unsaved numbers included)
- ✅ **Delete for me** per message without entering selection mode
- ✅ **Automatic UI language** following the system locale, overridable in the Language menu
- ✅ Messages deleted/edited by others update live

## Keyboard shortcuts

| Keys | Action |
|---|---|
| `Ctrl+F` / `Ctrl+K` | focus the chat search |
| `Ctrl+Shift+F` | search inside the open conversation |
| `Ctrl+N` | new chat by phone number |
| `Ctrl+V` | paste an image or screenshot into the composer |
| `Enter` / `Shift+Enter` | send / newline |
| `↑` (empty composer) | re-edit your last sent message |
| `Esc` | leave multi-select / in-chat search / reply / edit, then close the conversation |
| `Ctrl+W` / `Ctrl+Q` | hide window to tray / quit |
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

> **CPU:** any x86-64 processor. Release binaries target x86-64-baseline (v1):
> builds pin `GOAMD64=v1` with fixed CGO flags, and `build-deb.sh` fails the
> release if the binary ever demands a newer ISA level — so even pre-AVX2 CPUs
> (e.g. Intel Sandy Bridge / 2nd-gen Core) run Obrol. If you ever see glibc's
> `CPU ISA level is lower than required`, that binary did not come from our
> release pipeline; verify with `readelf -nW obrol | grep 'ISA needed'`
> (expect `x86-64-baseline`).

### Option A — `.deb` package (recommended)

Download `obrol_1.15.0-1_amd64.deb` from [Releases](https://github.com/Katon17/obrol-wa/releases/latest), then:

```bash
sudo apt install ./obrol_1.15.0-1_amd64.deb
```

apt resolves all dependencies automatically and installs system-wide
(`/usr/bin/obrol` + menu entry + icons). Uninstall with `sudo apt remove obrol`
— chat data in `~/.local/share/obrol/` is kept.

### Option B — tarball (per-user, no sudo)

A prebuilt binary is shipped, so **you don't need Go**. Download the latest tarball from
[Releases](https://github.com/Katon17/obrol-wa/releases/latest), then:

```bash
tar xzf obrol-1.15.0-linux-x86_64.tar.gz
cd obrol
bash install.sh
```

Then launch it from **Menu → Internet → Obrol**, or run `obrol` in a terminal.
The install is per-user (no sudo needed):

- Binary: `~/.local/bin/obrol`
- Menu entry: `~/.local/share/applications/obrol.desktop`
- Data (login session + chat history): `~/.local/share/obrol/`

> **Flatpak:** a Flathub-ready manifest lives in `packaging/flatpak/`
> (submission in progress, issue #5). Until it lands on Flathub, the `.deb`
> and tarball remain the recommended install paths.

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

To build a Debian/Ubuntu package without installing it:

```bash
sudo apt install dpkg-dev binutils  # in addition to the build dependencies above
./build-deb.sh
```

Packages are written to `dist/`. The script reads the version from `main.go` and
uses your repository Git name/email as the package maintainer. Override these
with `DEB_VERSION` and `DEB_MAINTAINER` if needed; see `./build-deb.sh --help`.
Build on the oldest Debian/Ubuntu release you intend to support, since runtime
library requirements are detected from the build machine. The script requires
no root privileges and does not install the package.

Command-line options:

| Option | Effect |
|---|---|
| `--demo` | demo mode with sample data, no connection |
| `--demo-login` | show the sample login page |
| `--debug` | log whatsmeow traffic to the terminal |
| `--hidden` | start hidden in the system tray |
| `--version` | print the version |

## Release history

Full notes per release live in [CHANGELOG.md](CHANGELOG.md).

- **1.0.0**: first release.
- **1.1.0**: dark mode (auto/light/dark), per-chat mute synced with the phone, mute all groups
  except @mention/reply, mute badge, temporary header status falls back to connection status.
- **1.2.0**: paste screenshots/images from the clipboard with Ctrl+V, multi-line messages with
  Shift+Enter, edit message (right-click or ↑), delete for everyone, pin message (banner + jump),
  pin chat, sharper preview for own photos, creator credits (Katon Bagaskoro) in About.
- **1.3.0**: full stickers (render + animated, Recent/Favorites panel, create stickers from
  images/screenshots), original self-made SVG logo & icons
- **1.4.0**: message replies (quotes), emoji reactions, forwarding, and the English language option
- **1.5.0**: voice messages (record/send/play) + call notifications
- **1.6.0**: chat archive (Archived folder, kept permanently)
- **1.7.0**: contact & group profiles (photo, about, group info) + Saweria donation link
- **1.8.0**: full English coverage (~90 strings) + Ko-fi donation link in the About dialog.
- **1.9.0**: drag & drop files into conversations; WhatsApp-style presence (online, last
  seen, "*Budi is typing…*" including the sender name in groups; your own typing state is
  broadcast too); `.deb` package on the Releases page.
- **1.9.1**: fix for the drag & drop SIGSEGV caused by an upstream gotk3 regression
  (`GetURIs()`); panic "black box" at `~/.local/share/obrol/crash.log` + a per-drop guard
  that keeps the app alive.
- **1.9.2**: final drag & drop fix — the raw drop payload is now read by length
  (`GetText()` turned out to return NULL for `text/uri-list`); flash feedback when a drop
  carries no sendable file.
- **1.10.0–1.10.1**: the memory diet — RSS stays below 100 MB even after heavy sessions
- **1.11.0–1.11.1**: message search + Starred page; silent scroll-to-top auto-load for older
  history (no sidebar row)
- **1.12.0–1.12.1**: group info & management page (participants, subject, photo, description,
  add/remove members)
- **1.12.2**: whatsmeow library stabilisation; the Status/Story experiment was withdrawn
  (linked devices receive no status history from the protocol)
- **1.12.3**: automatic system language (PR #7 by baduhai); `go vet` format-string fixes
- **1.13.0**: multi-select messages with bulk actions; single-message *delete for me*;
  message menu now opens on button-release (no accidental activation)
- **1.14.0**: *Message privately* from the group message menu and member list;
  composer placeholder shortened
- **1.14.1**: full English translation coverage (menus, dates, tooltips, dialogs, errors)
- **1.14.2**: stuck typing indicator fixed (cleared on offline/message/timeout); last-seen persists across restarts; stray untranslated strings (logout dialog, "(Anda)", deleted-message preview) fixed
- **1.15.0**: two-step voice recording with preview bar (play/discard/send); emoji picker gains a flags tab and fuller gestures/hugs; tray right-click menu now shows Quit on SNI hosts (with automatic XEmbed fallback on panels that cannot render SNI menus); sticker panel empty states translated

## Troubleshooting

| Problem | Fix |
|---|---|
| Tray icon missing | The XFCE panel needs the **Status Tray** plugin (right-click panel → Panel → Add New Items → *Status Tray Plugin*). Without a tray, closing the window quits the app. |
| No notification sound | `sudo apt install gnome-session-canberra` (provides `canberra-gtk-play`) |
| Emoji shown as boxes | `sudo apt install fonts-noto-color-emoji` |
| Want to re-login / broken session | Menu ☰ → Log out, or delete `~/.local/share/obrol/` |
| "Session opened elsewhere" | Obrol is running on another computer with the same session |
| App crashed on its own | Send the contents of `~/.local/share/obrol/crash.log` — the automatic black box that records panic traces |
| Drag & drop does nothing | Make sure you run ≥ 1.9.2 (`obrol --version`), then drop files onto an open conversation |
| RAM grows after long use | Since 1.9.3 the animated-sticker cache is capped (±4 MB/sticker, 24 stickers); restarting Obrol releases an already-full cache |

## Project layout

```
main.go    entry point, CLI options, config, demo data
wa.go      whatsmeow connection: login, events, history, send/receive, media
store.go   local SQLite database (chats + messages)
ui.go      the whole GTK3 interface
features.go multi-line input, image paste, edit/delete, pin message & chat
icons.go   bundled SVG icons & logo (a recolorable DrawingArea widget)
i18n.go    UI translations (Bahasa Indonesia / English)
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
