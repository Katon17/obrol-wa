# Changelog

All notable changes to Obrol are documented in this file.
The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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

[1.4.0]: https://github.com/Katon17/obrol-wa/releases/tag/v1.4.0
[1.3.0]: https://github.com/Katon17/obrol-wa/releases/tag/v1.3.0
[1.2.0]: https://github.com/Katon17/obrol-wa/releases/tag/v1.2.0
[1.1.0]: https://github.com/Katon17/obrol-wa/releases/tag/v1.1.0
[1.0.0]: https://github.com/Katon17/obrol-wa/releases/tag/v1.0.0
