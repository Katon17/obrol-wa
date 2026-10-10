package main

// ui.go — antarmuka GTK3 native.

import (
	"fmt"
	"hash/fnv"
	"html"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
	qrcode "github.com/skip2/go-qrcode"
)

const pageSize = 80

type chatRow struct {
	row     *gtk.ListBoxRow
	avatar  *gtk.Label
	name    *gtk.Label
	preview *gtk.Label
	when    *gtk.Label
	badge   *gtk.Label
	muteIc  *Icon
	pinIc   *Icon
	chat    Chat
	display string
}

type metaRef struct {
	label  *gtk.Label
	ts     int64
	status int
	edited bool
}

type App struct {
	gapp *gtk.Application
	win  *gtk.ApplicationWindow
	wa   *WA
	st   *Store
	cfg  *Config
	opts Options

	header *gtk.HeaderBar
	pages  *gtk.Stack

	// login
	qrImage   *gtk.Image
	qrSpinner *gtk.Spinner
	qrRetry   *gtk.Button
	phoneEnt  *gtk.Entry
	pairLbl   *gtk.Label

	// daftar chat
	search   *gtk.SearchEntry
	chatList *gtk.ListBox
	rows     map[string]*chatRow
	filter   string

	// panel percakapan
	chatStack  *gtk.Stack
	chatTitle  *gtk.Label
	chatSub    *gtk.Label
	chatAvatar *gtk.Label
	msgList    *gtk.ListBox
	msgScroll  *gtk.ScrolledWindow

	// composer & edit
	input       *gtk.TextView
	inputBuf    *gtk.TextBuffer
	placeholder *gtk.Label
	inputSW     *gtk.ScrolledWindow
	inputW      int
	inputH      int
	editBar     *gtk.Box
	editLbl     *gtk.Label
	editID      string
	editChat    string

	// pesan tersemat
	pinBar   *gtk.EventBox
	pinTitle *gtk.Label
	pinLbl   *gtk.Label
	pins     []Pin
	pinIdx   int
	rowsByID map[string]*gtk.ListBoxRow

	current     string
	oldestTS    int64
	noMore      bool
	loading     bool
	firstDay    string
	firstSepRow *gtk.ListBoxRow
	lastDay     string
	metas       map[string]*metaRef
	stickBottom bool
	restoreDist float64

	// tray
	tray      *StatusIcon
	pixNormal *gdk.Pixbuf
	pixUnread *gdk.Pixbuf
	iconPath  string

	focused   bool
	quitting  bool
	selecting bool

	// tema
	themeCSS       *gtk.CssProvider
	settings       *gtk.Settings
	origPreferDark bool
	dark           bool

	// stiker & ikon
	stickers *stickerPanel
	msgGen   int
	muteIcon *Icon
	traySize int

	// status & mute
	muteBtn    *gtk.MenuButton
	connStatus string
	statusSeq  int
}

// ------------------------------------------------------------ bootstrap

func (a *App) Activate() {
	if a.win != nil {
		a.showWindow()
		return
	}
	a.rows = map[string]*chatRow{}
	a.metas = map[string]*metaRef{}
	a.rowsByID = map[string]*gtk.ListBoxRow{}
	a.restoreDist = -1
	a.loadCSS()
	a.pixNormal = pixbufFromBytes(iconPNG)
	a.pixUnread = pixbufFromBytes(iconUnreadPNG)

	win, _ := gtk.ApplicationWindowNew(a.gapp)
	a.win = win
	win.SetTitle("Obrol")
	win.SetDefaultSize(1000, 680)
	win.SetIcon(a.pixNormal)
	win.SetPosition(gtk.WIN_POS_CENTER)

	a.header, _ = gtk.HeaderBarNew()
	a.header.SetShowCloseButton(true)
	a.header.SetTitle("Obrol")
	a.header.SetSubtitle("Memulai…")
	win.SetTitlebar(a.header)
	a.buildHeaderButtons()

	a.pages, _ = gtk.StackNew()
	a.pages.SetTransitionType(gtk.STACK_TRANSITION_TYPE_CROSSFADE)
	a.pages.AddNamed(a.buildLogin(), "login")
	a.pages.AddNamed(a.buildMain(), "main")
	win.Add(a.pages)

	win.Connect("delete-event", func() bool {
		if a.quitting {
			return false
		}
		if a.tray != nil && a.tray.IsEmbedded() && a.cfg.CloseToTray {
			win.Hide()
			return true
		}
		a.quit()
		return true
	})
	win.Connect("notify::is-active", func() {
		a.focused = win.IsActive()
		if a.focused && a.current != "" {
			if r, ok := a.rows[a.current]; ok && r.chat.Unread > 0 {
				a.markCurrentRead()
			}
		}
	})
	win.Connect("key-press-event", a.onKey)

	a.setupTray()
	a.setupActions()

	win.ShowAll()
	a.applyTheme()
	// refresh tiap menit: mute yang kedaluwarsa, label waktu "Kemarin", dll.
	glib.TimeoutAdd(60000, func() bool { a.reloadChats(); return true })
	if a.opts.Demo {
		a.pages.SetVisibleChildName("main")
		a.connStatus = "Mode demo"
		a.header.SetSubtitle("Mode demo")
		a.reloadChats()
	} else if a.opts.DemoLogin {
		a.pages.SetVisibleChildName("login")
		a.OnQR("2@demo-obrol-qr-code-preview,abcdefghijklmnopqrstuvwxyz0123456789")
	} else if a.wa.IsPaired() {
		a.pages.SetVisibleChildName("main")
		a.reloadChats()
	} else {
		a.pages.SetVisibleChildName("login")
	}
	a.chatStack.SetVisibleChildName("empty")
	if a.opts.Hidden && a.tray != nil {
		glib.TimeoutAdd(1500, func() bool {
			if a.tray.IsEmbedded() {
				win.Hide()
			}
			return false
		})
	}
	if a.wa != nil {
		a.wa.Start()
	}
	a.updateUnread()
}

func (a *App) showWindow() {
	a.win.Show()
	a.win.Deiconify()
	a.win.PresentWithTime(uint32(gdk.CURRENT_TIME))
}

func (a *App) quit() {
	a.quitting = true
	if a.tray != nil {
		a.tray.SetVisible(false)
	}
	if a.wa != nil {
		a.wa.Close()
	}
	a.gapp.Quit()
}

func (a *App) onKey(_ *gtk.ApplicationWindow, ev *gdk.Event) bool {
	k := gdk.EventKeyNewFromEvent(ev)
	ctrl := k.State()&uint(gdk.CONTROL_MASK) != 0
	switch {
	case ctrl && (k.KeyVal() == gdk.KEY_f || k.KeyVal() == gdk.KEY_k):
		a.search.GrabFocus()
		return true
	case ctrl && k.KeyVal() == gdk.KEY_n:
		a.newChatDialog()
		return true
	case ctrl && k.KeyVal() == gdk.KEY_q:
		a.quit()
		return true
	case ctrl && k.KeyVal() == gdk.KEY_w:
		a.win.Close()
		return true
	case k.KeyVal() == gdk.KEY_Escape && a.editID != "":
		a.cancelEdit()
		return true
	case k.KeyVal() == gdk.KEY_Escape && a.current != "":
		a.closeChat()
		return true
	}
	return false
}

func (a *App) setupActions() {
	act := glib.SimpleActionNew("open-chat", glib.VARIANT_TYPE_STRING)
	act.Connect("activate", func(_ *glib.SimpleAction, p *glib.Variant) {
		a.showWindow()
		if p != nil {
			a.openChat(p.GetString())
		}
	})
	a.gapp.AddAction(act)
}

// ------------------------------------------------------------ CSS

const baseCSS = `
.bubble { border-radius: 10px; padding: 5px 9px 4px 9px; }
.bubble label.meta { font-size: 0.75em; }
.bubble label.sender { font-weight: bold; font-size: 0.86em; }
.bubble label.deleted { font-style: italic; }
.bubble button { padding: 2px 8px; background-image: none; box-shadow: none; text-shadow: none; -gtk-icon-shadow: none; }
.thumb { border-radius: 6px; }
.datesep { border-radius: 8px; padding: 3px 12px; font-size: 0.8em; }
.avatar { border-radius: 999px; color: #ffffff; font-weight: bold; min-width: 44px; min-height: 44px; }
.avatar.small { min-width: 36px; min-height: 36px; font-size: 0.9em; }
.av0 { background-color: #00a884; } .av1 { background-color: #53bdeb; } .av2 { background-color: #ff7eb6; }
.av3 { background-color: #8a6fdf; } .av4 { background-color: #f5a623; } .av5 { background-color: #e5533d; }
.av6 { background-color: #3a8bd6; } .av7 { background-color: #5d9c59; }
.chatname { font-weight: bold; }
.preview { opacity: 0.68; font-size: 0.92em; }
.when { opacity: 0.6; font-size: 0.8em; }
.when.unread { color: #1fa855; opacity: 1; font-weight: bold; }
.mutedicon { opacity: 0.55; font-size: 0.85em; }
.badge { background-color: #25d366; color: #ffffff; border-radius: 999px; padding: 0 6px; font-size: 0.78em; font-weight: bold; min-width: 10px; }
.chatlist row { padding: 7px 10px; }
.chathdr { padding: 6px 12px; }
.chathdr button { padding: 2px 8px; }
.composer { padding: 6px 8px; }
.qrframe { background-color: #ffffff; padding: 14px; border-radius: 12px; }
.bigtitle { font-size: 1.5em; font-weight: bold; }
.warn { color: #b26a00; font-size: 0.85em; }
.emptyhint { opacity: 0.6; }
.credit { opacity: 0.55; font-size: 0.85em; }
.inputbox { background-color: @theme_base_color; border: 1px solid @borders; border-radius: 18px; padding: 7px 12px; }
.inputbox textview, .inputbox textview text { background-color: transparent; }
.placeholder { opacity: 0.45; }
.editbar { padding: 4px 8px 4px 12px; border-left: 4px solid #00a884; background-color: alpha(@theme_fg_color, 0.05); }
.pinbar { padding: 5px 12px; background-color: alpha(@theme_fg_color, 0.04); border-bottom: 1px solid alpha(@theme_fg_color, 0.1); }
.pintitle { color: #00a884; font-size: 0.82em; font-weight: bold; }
row.flash, row.flash:hover { background-color: alpha(#00a884, 0.28); }
.accent { color: #00a884; }
.mutedicon { opacity: 0.6; }
.iconbtn { padding: 4px 6px; }
.bubble.sticker.in, .bubble.sticker.out { background-color: transparent; box-shadow: none; padding: 0; }
.bubble.sticker label.meta { background-color: rgba(0,0,0,0.42); color: #ffffff; border-radius: 8px; padding: 1px 7px; margin-top: 2px; }
.stickerph { opacity: 0.35; }
.stickerpanel { padding: 8px; }
.stickercell { padding: 3px; border-radius: 10px; }
`

const lightCSS = `
.msgbg { background-color: #efeae2; }
.bubble { box-shadow: 0 1px 0.5px rgba(0,0,0,0.13); }
.bubble.in  { background-color: #ffffff; }
.bubble.sticker.in, .bubble.sticker.out { background-color: transparent; box-shadow: none; padding: 0; }
.bubble.sticker label.meta { background-color: rgba(0,0,0,0.42); color: #ffffff; border-radius: 8px; padding: 1px 7px; margin-top: 2px; }
.bubble.out { background-color: #d9fdd3; }
.bubble label { color: #111b21; }
.bubble label.meta, .bubble label.deleted { color: #667781; }
.bubble label link:link, .bubble label link:visited { color: #027eb5; }
.bubble button { background-color: rgba(0,0,0,0.05); color: #111b21; border: 1px solid rgba(0,0,0,0.12); }
.bubble button:hover { background-color: rgba(0,0,0,0.10); }
.bubble button label { color: #111b21; }
.bubble .icon { color: #667781; }
.bubble button .icon { color: #00a884; }
.datesep { background-color: rgba(255,255,255,0.95); color: #54656f; }
.badge.muted { background-color: #aebac1; }
.chatlist row { border-bottom: 1px solid rgba(0,0,0,0.05); }
.chathdr { border-bottom: 1px solid rgba(0,0,0,0.1); }
.composer { border-top: 1px solid rgba(0,0,0,0.1); }
`

const darkCSS = `
.msgbg { background-color: #0b141a; }
.bubble { box-shadow: 0 1px 0.5px rgba(0,0,0,0.4); }
.bubble.in  { background-color: #202c33; }
.bubble.sticker.in, .bubble.sticker.out { background-color: transparent; box-shadow: none; padding: 0; }
.bubble.sticker label.meta { background-color: rgba(0,0,0,0.42); color: #ffffff; border-radius: 8px; padding: 1px 7px; margin-top: 2px; }
.bubble.out { background-color: #005c4b; }
.bubble label { color: #e9edef; }
.bubble label.meta, .bubble label.deleted { color: #8696a0; }
.bubble.out label.meta { color: rgba(233,237,239,0.6); }
.bubble label link:link, .bubble label link:visited { color: #53bdeb; }
.bubble button { background-color: rgba(255,255,255,0.07); color: #e9edef; border: 1px solid rgba(255,255,255,0.12); }
.bubble button:hover { background-color: rgba(255,255,255,0.14); }
.bubble button label { color: #e9edef; }
.bubble .icon { color: #8696a0; }
.bubble button .icon { color: #00a884; }
.stickerph { color: #e9edef; }
.datesep { background-color: #182229; color: #8696a0; }
.emptyhint { color: #d1d7db; }
.badge.muted { background-color: #54656f; }
.when.unread { color: #00a884; }
.chatlist row { border-bottom: 1px solid rgba(255,255,255,0.05); }
.chathdr { border-bottom: 1px solid rgba(255,255,255,0.08); }
.composer { border-top: 1px solid rgba(255,255,255,0.08); }
`

func (a *App) loadCSS() {
	screen, _ := gdk.ScreenGetDefault()
	base, _ := gtk.CssProviderNew()
	base.LoadFromData(baseCSS)
	gtk.AddProviderForScreen(screen, base, gtk.STYLE_PROVIDER_PRIORITY_APPLICATION)
	a.themeCSS, _ = gtk.CssProviderNew()
	gtk.AddProviderForScreen(screen, a.themeCSS, gtk.STYLE_PROVIDER_PRIORITY_APPLICATION+1)

	if set, err := gtk.SettingsGetDefault(); err == nil {
		a.settings = set
		if v, err := set.GetProperty("gtk-application-prefer-dark-theme"); err == nil {
			a.origPreferDark, _ = v.(bool)
		}
		set.Connect("notify::gtk-theme-name", func() { glib.IdleAdd(a.applyTheme) })
	}
}

// systemDark mendeteksi apakah tema GTK yang aktif gelap.
func (a *App) systemDark() bool {
	if a.settings != nil {
		if v, err := a.settings.GetProperty("gtk-theme-name"); err == nil {
			if n, ok := v.(string); ok && strings.Contains(strings.ToLower(n), "dark") {
				return true
			}
		}
		if a.origPreferDark {
			return true
		}
	}
	// cadangan: hitung kecerahan warna latar tema
	if a.win != nil {
		if sc, err := a.win.GetStyleContext(); err == nil {
			if c, ok := sc.LookupColor("theme_bg_color"); ok && c != nil {
				lum := 0.2126*c.GetRed() + 0.7152*c.GetGreen() + 0.0722*c.GetBlue()
				return lum < 0.5
			}
		}
	}
	return false
}

func (a *App) applyTheme() {
	if a.settings != nil {
		// paksa varian gelap tema GTK bila user memilih "Gelap"
		a.settings.SetProperty("gtk-application-prefer-dark-theme", a.cfg.Theme == "dark" || a.origPreferDark)
	}
	switch a.cfg.Theme {
	case "dark":
		a.dark = true
	case "light":
		a.dark = false
	default:
		a.dark = a.systemDark()
	}
	if a.dark {
		a.themeCSS.LoadFromData(darkCSS)
	} else {
		a.themeCSS.LoadFromData(lightCSS)
	}
	// warna nama pengirim grup bergantung tema → render ulang chat terbuka
	if a.current != "" && a.msgList != nil {
		a.reopenKeepScroll(a.current)
	}
}

func addClass(w interface {
	GetStyleContext() (*gtk.StyleContext, error)
}, classes ...string) {
	sc, err := w.GetStyleContext()
	if err != nil {
		return
	}
	for _, c := range classes {
		sc.AddClass(c)
	}
}

// ------------------------------------------------------------ header

func (a *App) buildHeaderButtons() {
	newBtn := iconButton("chat-new", "Obrolan baru (Ctrl+N)", 18)
	newBtn.SetRelief(gtk.RELIEF_NORMAL)
	newBtn.Connect("clicked", a.newChatDialog)
	a.header.PackStart(newBtn)

	menu, _ := gtk.MenuNew()
	add := func(label string, fn func()) {
		it, _ := gtk.MenuItemNewWithLabel(label)
		it.Connect("activate", fn)
		menu.Append(it)
	}
	check := func(label string, val bool, fn func(bool)) {
		it, _ := gtk.CheckMenuItemNewWithLabel(label)
		it.SetActive(val)
		it.Connect("toggled", func() { fn(it.GetActive()) })
		menu.Append(it)
	}
	sep := func() { s, _ := gtk.SeparatorMenuItemNew(); menu.Append(s) }

	add("Obrolan baru…", a.newChatDialog)
	add("Buka folder unduhan", func() {
		d := downloadDir()
		os.MkdirAll(d, 0o755)
		exec.Command("xdg-open", d).Start()
	})
	sep()
	check("Notifikasi desktop", a.cfg.Notify, func(v bool) { a.cfg.Notify = v; a.cfg.Save() })
	check("Bisukan semua grup (kecuali @mention/balasan)", a.cfg.MuteGroups, func(v bool) {
		a.cfg.MuteGroups = v
		a.cfg.Save()
		a.reloadChats()
	})
	check("Suara notifikasi", a.cfg.Sound, func(v bool) { a.cfg.Sound = v; a.cfg.Save() })
	check("Tutup = sembunyikan ke tray", a.cfg.CloseToTray, func(v bool) { a.cfg.CloseToTray = v; a.cfg.Save() })
	check("Jalankan saat login", autostartEnabled(), func(v bool) { setAutostart(v) })
	sep()
	// submenu tema
	themeItem, _ := gtk.MenuItemNewWithLabel("Tema")
	themeMenu, _ := gtk.MenuNew()
	var first *gtk.RadioMenuItem
	for _, t := range []struct{ id, label string }{{"auto", "Otomatis (ikuti sistem)"}, {"light", "Terang"}, {"dark", "Gelap"}} {
		var it *gtk.RadioMenuItem
		if first == nil {
			it, _ = gtk.RadioMenuItemNewWithLabel(nil, t.label)
			first = it
		} else {
			it, _ = gtk.RadioMenuItemNewWithLabelFromWidget(first, t.label)
		}
		it.SetActive(a.cfg.Theme == t.id)
		id := t.id
		it.Connect("toggled", func() {
			if it.GetActive() {
				a.cfg.Theme = id
				a.cfg.Save()
				a.applyTheme()
			}
		})
		themeMenu.Append(it)
	}
	themeItem.SetSubmenu(themeMenu)
	menu.Append(themeItem)
	sep()
	add("Keluar dari akun (logout)…", a.confirmLogout)
	add("Tentang Obrol", a.about)
	sep()
	add("Tutup aplikasi (Ctrl+Q)", a.quit)
	menu.ShowAll()

	mb, _ := gtk.MenuButtonNew()
	if old, err := mb.GetChild(); err == nil && old != nil {
		mb.Remove(old)
	}
	mb.Add(newIcon("menu", 18))
	mb.SetTooltipText("Menu")
	mb.SetPopup(menu)
	a.header.PackEnd(mb)
}

// ------------------------------------------------------------ login page

func (a *App) buildLogin() gtk.IWidget {
	outer, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	outer.SetHAlign(gtk.ALIGN_CENTER)
	outer.SetVAlign(gtk.ALIGN_CENTER)

	box, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 36)
	outer.PackStart(box, false, false, 0)

	left, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 12)
	left.SetVAlign(gtk.ALIGN_CENTER)
	title, _ := gtk.LabelNew("Hubungkan Obrol ke WhatsApp")
	addClass(title, "bigtitle")
	title.SetXAlign(0)
	steps, _ := gtk.LabelNew("")
	steps.SetMarkup("1.  Buka <b>WhatsApp</b> di ponsel\n" +
		"2.  Ketuk <b>⋮ Menu</b> (Android) atau <b>Pengaturan</b> (iPhone)\n" +
		"3.  Pilih <b>Perangkat tertaut</b> → <b>Tautkan perangkat</b>\n" +
		"4.  Arahkan kamera ponsel ke kode QR di samping")
	steps.SetXAlign(0)
	steps.SetLineWrap(true)

	sepr, _ := gtk.SeparatorNew(gtk.ORIENTATION_HORIZONTAL)
	alt, _ := gtk.LabelNew("")
	alt.SetMarkup("<b>Atau masuk dengan nomor telepon</b>")
	alt.SetXAlign(0)
	prow, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 6)
	a.phoneEnt, _ = gtk.EntryNew()
	a.phoneEnt.SetPlaceholderText("contoh: 0812xxxxxxx")
	a.phoneEnt.SetInputPurpose(gtk.INPUT_PURPOSE_PHONE)
	pbtn, _ := gtk.ButtonNewWithLabel("Dapatkan kode")
	prow.PackStart(a.phoneEnt, true, true, 0)
	prow.PackStart(pbtn, false, false, 0)
	a.pairLbl, _ = gtk.LabelNew("")
	a.pairLbl.SetXAlign(0)
	a.pairLbl.SetSelectable(true)
	a.pairLbl.SetLineWrap(true)
	doPair := func() {
		ph := a.phoneEnt.GetChars(0, -1)
		a.pairLbl.SetText("Meminta kode…")
		go func() {
			code, err := a.wa.PairPhone(ph)
			glib.IdleAdd(func() {
				if err != nil {
					a.pairLbl.SetText("Gagal: " + err.Error())
					return
				}
				a.pairLbl.SetMarkup("<span size='large'>Masukkan kode ini di ponsel:</span>\n<span font_family='monospace' size='xx-large' weight='bold'>" + html.EscapeString(code) + "</span>\n<small>Ponsel: Perangkat tertaut → Tautkan perangkat → Tautkan dengan nomor telepon</small>")
			})
		}()
	}
	pbtn.Connect("clicked", doPair)
	a.phoneEnt.Connect("activate", doPair)

	warn, _ := gtk.LabelNew("⚠ Obrol adalah klien tidak resmi (protokol whatsmeow). Ada risiko akun\ndibatasi/diblokir oleh WhatsApp. Gunakan dengan bijak & risiko sendiri.")
	addClass(warn, "warn")
	warn.SetXAlign(0)

	left.PackStart(title, false, false, 0)
	left.PackStart(steps, false, false, 4)
	left.PackStart(sepr, false, false, 8)
	left.PackStart(alt, false, false, 0)
	left.PackStart(prow, false, false, 0)
	left.PackStart(a.pairLbl, false, false, 0)
	left.PackStart(warn, false, false, 12)
	credit, _ := gtk.LabelNew("Obrol " + version + " · dibuat oleh " + author)
	addClass(credit, "credit")
	credit.SetXAlign(0)
	left.PackStart(credit, false, false, 0)
	left.SetSizeRequest(400, -1)

	right, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 10)
	right.SetVAlign(gtk.ALIGN_CENTER)
	frame, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	addClass(frame, "qrframe")
	ov, _ := gtk.OverlayNew()
	a.qrImage, _ = gtk.ImageNew()
	a.qrImage.SetSizeRequest(264, 264)
	a.qrSpinner, _ = gtk.SpinnerNew()
	a.qrSpinner.SetSizeRequest(48, 48)
	a.qrSpinner.SetHAlign(gtk.ALIGN_CENTER)
	a.qrSpinner.SetVAlign(gtk.ALIGN_CENTER)
	a.qrSpinner.Start()
	ov.Add(a.qrImage)
	ov.AddOverlay(a.qrSpinner)
	frame.PackStart(ov, false, false, 0)
	a.qrRetry, _ = gtk.ButtonNewWithLabel("Kode kedaluwarsa — muat ulang QR")
	a.qrRetry.SetNoShowAll(true)
	a.qrRetry.Connect("clicked", func() {
		a.qrRetry.Hide()
		a.qrSpinner.Show()
		a.qrSpinner.Start()
		a.qrImage.Clear()
		if a.wa != nil {
			a.wa.RetryQR()
		}
	})
	right.PackStart(frame, false, false, 0)
	right.PackStart(a.qrRetry, false, false, 0)

	box.PackStart(left, false, false, 0)
	box.PackStart(right, false, false, 0)
	return outer
}

// ------------------------------------------------------------ main page

func (a *App) buildMain() gtk.IWidget {
	paned, _ := gtk.PanedNew(gtk.ORIENTATION_HORIZONTAL)
	paned.SetPosition(340)

	// kiri
	left, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	a.search, _ = gtk.SearchEntryNew()
	a.search.SetPlaceholderText("Cari obrolan (Ctrl+F)")
	a.search.SetMarginStart(8)
	a.search.SetMarginEnd(8)
	a.search.SetMarginTop(8)
	a.search.SetMarginBottom(8)
	a.search.Connect("search-changed", func() {
		a.filter = strings.ToLower(strings.TrimSpace(a.search.GetChars(0, -1)))
		a.chatList.InvalidateFilter()
	})
	a.search.Connect("activate", func() {
		// Enter di kolom cari → buka hasil pertama yang terlihat
		for i := 0; ; i++ {
			r := a.chatList.GetRowAtIndex(i)
			if r == nil {
				break
			}
			if a.filterRow(r) {
				a.chatList.SelectRow(r)
				a.input.GrabFocus()
				break
			}
		}
	})
	left.PackStart(a.search, false, false, 0)

	sw, _ := gtk.ScrolledWindowNew(nil, nil)
	sw.SetPolicy(gtk.POLICY_NEVER, gtk.POLICY_AUTOMATIC)
	a.chatList, _ = gtk.ListBoxNew()
	addClass(a.chatList, "chatlist")
	a.chatList.SetSelectionMode(gtk.SELECTION_SINGLE)
	a.chatList.SetSortFunc(func(r1, r2 *gtk.ListBoxRow) int {
		c1, c2 := a.rowChat(r1), a.rowChat(r2)
		if c1 == nil || c2 == nil {
			return 0
		}
		switch {
		case c1.chat.Pinned > c2.chat.Pinned:
			return -1
		case c1.chat.Pinned < c2.chat.Pinned:
			return 1
		}
		switch {
		case c1.chat.LastTS > c2.chat.LastTS:
			return -1
		case c1.chat.LastTS < c2.chat.LastTS:
			return 1
		}
		return strings.Compare(c1.chat.JID, c2.chat.JID)
	})
	a.chatList.SetFilterFunc(a.filterRow)
	a.chatList.Connect("row-selected", func(_ *gtk.ListBox, r *gtk.ListBoxRow) {
		if a.selecting || r == nil {
			return
		}
		if c := a.rowChat(r); c != nil && c.chat.JID != a.current {
			a.openChat(c.chat.JID)
		}
	})
	a.chatList.Connect("button-press-event", func(lb *gtk.ListBox, ev *gdk.Event) bool {
		b := gdk.EventButtonNewFromEvent(ev)
		if b.Button() != gdk.BUTTON_SECONDARY {
			return false
		}
		c := a.rowChat(lb.GetRowAtY(int(b.Y())))
		if c == nil {
			return false
		}
		m := a.muteMenu(c.chat.JID)
		m.PopupAtPointer(ev)
		return true
	})
	sw.Add(a.chatList)
	left.PackStart(sw, true, true, 0)
	paned.Pack1(left, false, false)

	// kanan
	a.chatStack, _ = gtk.StackNew()
	emptyBg, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	addClass(emptyBg, "msgbg")
	empty, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 10)
	empty.SetVAlign(gtk.ALIGN_CENTER)
	emptyBg.PackStart(empty, true, true, 0)
	logo, _ := gtk.ImageNewFromPixbuf(svgPixbuf(logoSVG, 112))
	eh, _ := gtk.LabelNew("")
	eh.SetMarkup("<span size='x-large' weight='bold'>Obrol</span>\nPilih obrolan di sebelah kiri untuk mulai mengirim pesan.\n<small>Ctrl+F cari · Ctrl+N obrolan baru · Esc tutup obrolan</small>\n\n<small>Obrol " + version + " · dibuat oleh " + html.EscapeString(author) + "</small>")
	eh.SetJustify(gtk.JUSTIFY_CENTER)
	addClass(eh, "emptyhint")
	empty.PackStart(logo, false, false, 0)
	empty.PackStart(eh, false, false, 0)
	a.chatStack.AddNamed(emptyBg, "empty")
	a.chatStack.AddNamed(a.buildChatView(), "chat")
	paned.Pack2(a.chatStack, true, false)
	return paned
}

func (a *App) buildChatView() gtk.IWidget {
	v, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)

	hdr, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 10)
	addClass(hdr, "chathdr")
	a.chatAvatar, _ = gtk.LabelNew("")
	addClass(a.chatAvatar, "avatar", "small")
	tb, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	tb.SetVAlign(gtk.ALIGN_CENTER)
	a.chatTitle, _ = gtk.LabelNew("")
	addClass(a.chatTitle, "chatname")
	a.chatTitle.SetXAlign(0)
	a.chatTitle.SetEllipsize(pango.ELLIPSIZE_END)
	a.chatSub, _ = gtk.LabelNew("")
	addClass(a.chatSub, "when")
	a.chatSub.SetXAlign(0)
	a.chatSub.SetSelectable(true)
	tb.PackStart(a.chatTitle, false, false, 0)
	tb.PackStart(a.chatSub, false, false, 0)
	hdr.PackStart(a.chatAvatar, false, false, 0)
	hdr.PackStart(tb, true, true, 0)
	a.muteBtn, _ = gtk.MenuButtonNew()
	a.muteBtn.SetRelief(gtk.RELIEF_NONE)
	a.muteBtn.SetVAlign(gtk.ALIGN_CENTER)
	if old, err := a.muteBtn.GetChild(); err == nil && old != nil {
		a.muteBtn.Remove(old)
	}
	a.muteIcon = newIcon("bell", 18)
	a.muteBtn.Add(a.muteIcon)
	hdr.PackEnd(a.muteBtn, false, false, 0)
	v.PackStart(hdr, false, false, 0)
	v.PackStart(a.buildPinBar(), false, false, 0)

	a.msgScroll, _ = gtk.ScrolledWindowNew(nil, nil)
	a.msgScroll.SetPolicy(gtk.POLICY_NEVER, gtk.POLICY_AUTOMATIC)
	addClass(a.msgScroll, "msgbg")
	a.msgList, _ = gtk.ListBoxNew()
	a.msgList.SetSelectionMode(gtk.SELECTION_NONE)
	addClass(a.msgList, "msgbg")
	a.msgScroll.Add(a.msgList)
	adj := a.msgScroll.GetVAdjustment()
	adj.Connect("changed", func() {
		if a.stickBottom {
			adj.SetValue(adj.GetUpper() - adj.GetPageSize())
		} else if a.restoreDist >= 0 {
			adj.SetValue(adj.GetUpper() - a.restoreDist)
			a.restoreDist = -1
		}
	})
	adj.Connect("value-changed", func() {
		a.stickBottom = adj.GetValue() >= adj.GetUpper()-adj.GetPageSize()-60
		if adj.GetValue() < 80 && adj.GetUpper() > adj.GetPageSize() && !a.loading && !a.noMore && a.current != "" {
			a.loading = true
			glib.IdleAdd(a.loadOlder)
		}
	})
	v.PackStart(a.msgScroll, true, true, 0)

	v.PackStart(a.buildComposer(), false, false, 0)
	return v
}

// ------------------------------------------------------------ chat list

func (a *App) rowChat(r *gtk.ListBoxRow) *chatRow {
	if r == nil {
		return nil
	}
	n, err := r.GetName()
	if err != nil {
		return nil
	}
	return a.rows[n]
}

func (a *App) filterRow(r *gtk.ListBoxRow) bool {
	if a.filter == "" {
		return true
	}
	c := a.rowChat(r)
	if c == nil {
		return false
	}
	return strings.Contains(strings.ToLower(c.display), a.filter) || strings.Contains(c.chat.JID, a.filter)
}

func (a *App) displayName(c Chat) string {
	if a.wa == nil {
		if c.Name != "" {
			return c.Name
		}
		return strings.SplitN(c.JID, "@", 2)[0]
	}
	return a.wa.DisplayName(c.JID, c.Name)
}

func (a *App) reloadChats() {
	chats, err := a.st.ListChats()
	if err != nil {
		return
	}
	seen := map[string]bool{}
	for _, c := range chats {
		seen[c.JID] = true
		r, ok := a.rows[c.JID]
		if !ok {
			r = a.newChatRow(c)
			a.rows[c.JID] = r
			a.chatList.Add(r.row)
			r.row.ShowAll()
		}
		a.updateChatRow(r, c)
	}
	for jid, r := range a.rows {
		if !seen[jid] {
			r.row.Destroy()
			delete(a.rows, jid)
		}
	}
	a.selecting = true
	a.chatList.InvalidateSort()
	if r, ok := a.rows[a.current]; ok {
		a.chatList.SelectRow(r.row)
	}
	a.selecting = false
	a.updateUnread()
}

func (a *App) newChatRow(c Chat) *chatRow {
	r := &chatRow{}
	r.row, _ = gtk.ListBoxRowNew()
	r.row.SetName(c.JID)
	h, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 10)
	r.avatar, _ = gtk.LabelNew("")
	addClass(r.avatar, "avatar")
	r.avatar.SetVAlign(gtk.ALIGN_CENTER)

	v, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 2)
	v.SetVAlign(gtk.ALIGN_CENTER)
	top, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 6)
	r.name, _ = gtk.LabelNew("")
	addClass(r.name, "chatname")
	r.name.SetXAlign(0)
	r.name.SetEllipsize(pango.ELLIPSIZE_END)
	r.when, _ = gtk.LabelNew("")
	addClass(r.when, "when")
	r.muteIc = newIcon("bell-off", 14)
	addClass(r.muteIc, "mutedicon")
	r.muteIc.SetNoShowAll(true)
	r.muteIc.SetTooltipText("Notifikasi dibisukan")
	top.PackStart(r.name, true, true, 0)
	top.PackEnd(r.when, false, false, 0)
	top.PackEnd(r.muteIc, false, false, 0)
	r.pinIc = newIcon("pin", 14)
	addClass(r.pinIc, "mutedicon")
	r.pinIc.SetNoShowAll(true)
	r.pinIc.SetTooltipText("Obrolan disematkan")
	top.PackEnd(r.pinIc, false, false, 0)

	bot, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 6)
	r.preview, _ = gtk.LabelNew("")
	addClass(r.preview, "preview")
	r.preview.SetXAlign(0)
	r.preview.SetEllipsize(pango.ELLIPSIZE_END)
	r.preview.SetWidthChars(10)
	r.badge, _ = gtk.LabelNew("")
	addClass(r.badge, "badge")
	r.badge.SetNoShowAll(true)
	bot.PackStart(r.preview, true, true, 0)
	bot.PackEnd(r.badge, false, false, 0)

	v.PackStart(top, false, false, 0)
	v.PackStart(bot, false, false, 0)
	h.PackStart(r.avatar, false, false, 0)
	h.PackStart(v, true, true, 0)
	r.row.Add(h)
	return r
}

func (a *App) updateChatRow(r *chatRow, c Chat) {
	disp := a.displayName(c)
	if disp != r.display {
		r.display = disp
		r.name.SetText(disp)
		r.avatar.SetText(initials(disp, c.IsGroup))
		addClass(r.avatar, avatarClass(c.JID))
	}
	if c.LastText != r.chat.LastText || r.chat.JID == "" {
		r.preview.SetText(c.LastText)
	}
	r.when.SetText(shortTime(c.LastTS))
	sc, _ := r.when.GetStyleContext()
	bsc, _ := r.badge.GetStyleContext()
	muted := a.isMuted(c)
	if c.Muted() {
		r.muteIc.Show()
	} else {
		r.muteIc.Hide()
	}
	r.pinIc.SetVisible(c.Pinned > 0)
	if muted {
		bsc.AddClass("muted")
	} else {
		bsc.RemoveClass("muted")
	}
	if c.Unread > 0 {
		r.badge.SetText(fmt.Sprint(c.Unread))
		r.badge.Show()
		if muted {
			sc.RemoveClass("unread")
		} else {
			sc.AddClass("unread")
		}
	} else {
		r.badge.Hide()
		sc.RemoveClass("unread")
	}
	r.chat = c
}

func (a *App) updateUnread() {
	total := a.st.TotalUnread(a.cfg.MuteGroups)
	title := "Obrol"
	if total > 0 {
		title = fmt.Sprintf("(%d) Obrol", total)
	}
	if a.win != nil {
		a.win.SetTitle(title)
		a.header.SetTitle(title)
	}
	if a.tray != nil {
		if total > 0 {
			a.tray.SetFromPixbuf(a.trayPixbuf(true))
			a.tray.SetTooltipText(fmt.Sprintf("Obrol — %d pesan belum dibaca", total))
		} else {
			a.tray.SetFromPixbuf(a.trayPixbuf(false))
			a.tray.SetTooltipText("Obrol")
		}
	}
}

// ------------------------------------------------------------ percakapan

func (a *App) openChat(jid string) {
	if jid == "" {
		return
	}
	if a.editID != "" && a.editChat != jid {
		a.cancelEdit()
	}
	if jid != a.current {
		a.pinIdx = 0
	}
	a.current = jid
	c, ok := a.st.GetChat(jid)
	if !ok {
		c = Chat{JID: jid, IsGroup: isGroupJID(jid)}
	}
	disp := a.displayName(c)
	a.chatTitle.SetText(disp)
	a.chatAvatar.SetText(initials(disp, c.IsGroup))
	sc, _ := a.chatAvatar.GetStyleContext()
	for i := 0; i < 8; i++ {
		sc.RemoveClass(fmt.Sprintf("av%d", i))
	}
	sc.AddClass(avatarClass(jid))
	if c.IsGroup {
		a.chatSub.SetText("Grup")
	} else {
		user := strings.SplitN(jid, "@", 2)[0]
		if strings.HasSuffix(jid, "@s.whatsapp.net") {
			a.chatSub.SetText(formatPhone(user))
		} else {
			a.chatSub.SetText(user)
		}
	}
	a.selecting = true
	if r, ok := a.rows[jid]; ok {
		a.chatList.SelectRow(r.row)
	} else {
		a.chatList.UnselectAll()
	}
	a.selecting = false

	a.clearMessages()
	msgs, _ := a.st.LoadMessages(jid, 0, pageSize)
	a.noMore = len(msgs) < pageSize
	a.stickBottom = true
	a.prependBatch(msgs)
	a.chatStack.SetVisibleChildName("chat")
	a.input.GrabFocus()
	if a.focused || a.opts.Demo {
		a.markCurrentRead()
	}
	c, _ = a.st.GetChat(jid)
	if c.JID == "" {
		c = Chat{JID: jid, IsGroup: isGroupJID(jid)}
	}
	a.refreshMuteBtn(c)
	a.refreshPins()
}

func (a *App) closeChat() {
	a.cancelEdit()
	a.current = ""
	a.pinBar.Hide()
	a.clearMessages()
	a.chatStack.SetVisibleChildName("empty")
	a.selecting = true
	a.chatList.UnselectAll()
	a.selecting = false
}

func (a *App) markCurrentRead() {
	if a.current == "" {
		return
	}
	if a.wa != nil {
		a.wa.MarkRead(a.current)
	} else {
		a.st.MarkChatRead(a.current)
	}
	if r, ok := a.rows[a.current]; ok {
		c := r.chat
		c.Unread = 0
		a.updateChatRow(r, c)
		a.refreshMuteBtn(c)
	}
	a.updateUnread()
}

func (a *App) clearMessages() {
	a.msgGen++ // hentikan animasi stiker dari percakapan sebelumnya
	a.msgList.GetChildren().Foreach(func(item interface{}) {
		if w, ok := item.(*gtk.Widget); ok {
			w.Destroy()
		}
	})
	a.metas = map[string]*metaRef{}
	a.rowsByID = map[string]*gtk.ListBoxRow{}
	a.oldestTS, a.firstDay, a.lastDay, a.firstSepRow = 0, "", "", nil
	a.loading = false
}

func (a *App) loadOlder() bool {
	defer func() { a.loading = false }()
	if a.current == "" || a.noMore || a.oldestTS == 0 {
		return false
	}
	msgs, _ := a.st.LoadMessages(a.current, a.oldestTS, pageSize)
	if len(msgs) < pageSize {
		a.noMore = true
	}
	if len(msgs) == 0 {
		return false
	}
	adj := a.msgScroll.GetVAdjustment()
	a.restoreDist = adj.GetUpper() - adj.GetValue()
	a.stickBottom = false
	a.prependBatch(msgs)
	return false
}

// prependBatch menyisipkan pesan (urut lama→baru) di atas daftar.
func (a *App) prependBatch(msgs []Msg) {
	if len(msgs) == 0 {
		return
	}
	pos := 0
	prev := ""
	var firstSep *gtk.ListBoxRow
	for i := range msgs {
		m := &msgs[i]
		d := dayKey(m.TS)
		if d != prev {
			sep := a.dateRow(m.TS)
			a.msgList.Insert(sep, pos)
			pos++
			if firstSep == nil {
				firstSep = sep
			}
			prev = d
		}
		a.msgList.Insert(a.bubbleRow(m), pos)
		pos++
	}
	// hapus separator lama bila harinya sama dengan akhir batch
	if a.firstSepRow != nil && prev == a.firstDay {
		a.firstSepRow.Destroy()
	}
	if a.lastDay == "" {
		a.lastDay = prev
	}
	a.firstSepRow = firstSep
	a.firstDay = dayKey(msgs[0].TS)
	a.oldestTS = msgs[0].TS
	a.msgList.ShowAll()
}

func (a *App) appendMessage(m *Msg) {
	d := dayKey(m.TS)
	if d != a.lastDay {
		sep := a.dateRow(m.TS)
		a.msgList.Add(sep)
		if a.firstSepRow == nil {
			a.firstSepRow, a.firstDay = sep, d
		}
		a.lastDay = d
	}
	if a.oldestTS == 0 {
		a.oldestTS = m.TS
	}
	r := a.bubbleRow(m)
	a.msgList.Add(r)
	a.msgList.ShowAll()
}

func (a *App) dateRow(ts int64) *gtk.ListBoxRow {
	row, _ := gtk.ListBoxRowNew()
	row.SetActivatable(false)
	row.SetSelectable(false)
	l, _ := gtk.LabelNew(dayLabel(ts))
	addClass(l, "datesep")
	l.SetHAlign(gtk.ALIGN_CENTER)
	l.SetMarginTop(10)
	l.SetMarginBottom(6)
	row.Add(l)
	return row
}

var urlRe = regexp.MustCompile(`https?://[^\s<>"]+`)

func linkify(s string) string {
	esc := html.EscapeString(s)
	return urlRe.ReplaceAllStringFunc(esc, func(u string) string {
		return `<a href="` + u + `">` + u + `</a>`
	})
}

func (a *App) bubbleRow(m *Msg) *gtk.ListBoxRow {
	row, _ := gtk.ListBoxRowNew()
	row.SetActivatable(false)
	row.SetSelectable(false)

	b, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 3)
	addClass(b, "bubble")
	chatID, msgID := m.Chat, m.ID
	a.rowsByID[m.ID] = row
	if m.FromMe {
		addClass(b, "out")
		b.SetHAlign(gtk.ALIGN_END)
		b.SetMarginStart(80)
		b.SetMarginEnd(14)
	} else {
		addClass(b, "in")
		b.SetHAlign(gtk.ALIGN_START)
		b.SetMarginStart(14)
		b.SetMarginEnd(80)
	}
	b.SetMarginTop(2)
	b.SetMarginBottom(2)

	if !m.FromMe && isGroupJID(m.Chat) {
		name := m.SenderName
		if name == "" {
			name = strings.SplitN(m.Sender, "@", 2)[0]
		}
		nl, _ := gtk.LabelNew("")
		nl.SetMarkup(fmt.Sprintf(`<span foreground="%s">%s</span>`, senderColor(m.Sender, a.dark), html.EscapeString(name)))
		addClass(nl, "sender")
		nl.SetXAlign(0)
		b.PackStart(nl, false, false, 0)
	}

	text := m.Text
	if m.Kind == "deleted" {
		box, l := iconLabel("ban", "Pesan ini dihapus", 15)
		addClass(l, "deleted")
		b.PackStart(box, false, false, 0)
		text = ""
	} else if m.Kind == "sticker" {
		addClass(b, "sticker")
		b.PackStart(a.stickerView(m), false, false, 0)
		text = ""
	} else if len(m.Thumb) > 0 && (m.Kind == "image" || m.Kind == "video" || m.Kind == "gif" || m.Kind == "location") {
		if pb := pixbufFromBytes(m.Thumb); pb != nil {
			pb = scaleToWidth(pb, 260)
			img, _ := gtk.ImageNewFromPixbuf(pb)
			addClass(img, "thumb")
			eb, _ := gtk.EventBoxNew()
			eb.Add(img)
			if isMediaKind(m.Kind) {
				eb.SetTooltipText("Klik untuk membuka " + strings.TrimSpace(strings.TrimLeftFunc(kindLabel(m.Kind), func(r rune) bool { return !unicode.IsLetter(r) })))
				chat, id := m.Chat, m.ID
				eb.Connect("button-press-event", func() { a.openMedia(chat, id) })
			}
			b.PackStart(eb, false, false, 0)
			if m.Kind == "video" || m.Kind == "gif" {
				l, _ := gtk.LabelNew(kindLabel(m.Kind) + " — klik gambar untuk memutar")
				addClass(l, "meta")
				l.SetXAlign(0)
				b.PackStart(l, false, false, 0)
			}
		}
	} else if isMediaKind(m.Kind) {
		label := plainKind(m.Kind)
		icon := "download"
		switch m.Kind {
		case "video", "gif", "audio", "ptt":
			icon = "play"
		case "image":
			icon = "image"
		}
		switch m.Kind {
		case "document":
			icon = "file"
			parts := strings.SplitN(text, "\n", 2)
			label = parts[0]
			text = ""
			if len(parts) > 1 {
				text = parts[1]
			}
		case "ptt", "audio":
			label += "  " + text
			text = ""
		}
		btn, _ := gtk.ButtonNew()
		bb, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 8)
		bb.PackStart(newIcon(icon, 18), false, false, 0)
		bl, _ := gtk.LabelNew(label + "  ·  Buka")
		bl.SetEllipsize(pango.ELLIPSIZE_MIDDLE)
		bl.SetMaxWidthChars(40)
		bb.PackStart(bl, false, false, 0)
		btn.Add(bb)
		btn.SetHAlign(gtk.ALIGN_START)
		chat, id := m.Chat, m.ID
		btn.Connect("clicked", func() { a.openMedia(chat, id) })
		b.PackStart(btn, false, false, 0)
	} else if lbl := kindLabel(m.Kind); lbl != "" {
		l, _ := gtk.LabelNew(lbl)
		addClass(l, "meta")
		l.SetXAlign(0)
		b.PackStart(l, false, false, 0)
	}

	if text != "" {
		l, _ := gtk.LabelNew("")
		l.SetMarkup(linkify(text))
		l.SetXAlign(0)
		l.SetLineWrap(true)
		l.SetLineWrapMode(pango.WRAP_WORD_CHAR)
		l.SetMaxWidthChars(60)
		l.SetSelectable(true)
		l.SetCanFocus(false)
		// klik kanan pada teks: menu bawaan (Salin/Pilih semua) + aksi pesan
		l.Connect("populate-popup", func(_ *gtk.Label, menu *gtk.Menu) {
			sep, _ := gtk.SeparatorMenuItemNew()
			menu.Append(sep)
			a.fillMsgMenu(menu, chatID, msgID, false)
			menu.ShowAll()
		})
		b.PackStart(l, false, false, 0)
	}

	meta, _ := gtk.LabelNew("")
	addClass(meta, "meta")
	meta.SetHAlign(gtk.ALIGN_END)
	a.setMeta(meta, m.TS, m.FromMe, m.Status, m.Edited)
	b.PackStart(meta, false, false, 0)
	if m.FromMe {
		a.metas[m.ID] = &metaRef{label: meta, ts: m.TS, status: m.Status, edited: m.Edited}
	}
	// klik kanan di area bubble → menu pesan
	eb, _ := gtk.EventBoxNew()
	eb.SetHAlign(b.GetHAlign())
	eb.Add(b)
	eb.Connect("button-press-event", func(_ *gtk.EventBox, ev *gdk.Event) bool {
		if gdk.EventButtonNewFromEvent(ev).Button() == gdk.BUTTON_SECONDARY {
			a.popupMsgMenu(ev, chatID, msgID)
			return true
		}
		return false
	})
	row.Add(eb)
	return row
}

func (a *App) setMeta(l *gtk.Label, ts int64, fromMe bool, st int, edited bool) {
	t := time.Unix(ts, 0).Format("15:04")
	if edited {
		t = "diedit · " + t
	}
	if !fromMe {
		l.SetText(t)
		return
	}
	tick := ""
	switch st {
	case StPending:
		tick = "◷"
	case StSent:
		tick = "✓"
	case StDelivered:
		tick = "✓✓"
	case StRead:
		tick = `<span foreground="#34b7f1" weight="bold">✓✓</span>`
	case StFailed:
		tick = `<span foreground="#e53935">⚠ gagal</span>`
	}
	l.SetMarkup(t + "  " + tick)
}

func (a *App) openMedia(chat, id string) {
	if a.wa == nil {
		return
	}
	a.flashStatus("Mengunduh media…")
	go func() {
		err := a.wa.OpenMedia(chat, id)
		glib.IdleAdd(func() {
			if err != nil {
				a.flashStatus("⚠ " + err.Error())
			} else {
				a.flashStatus("Media disimpan di " + downloadDir())
			}
		})
	}()
}

func (a *App) sendCurrent() {
	text := strings.TrimSpace(a.inputText())
	if a.editID != "" {
		a.finishEdit(text)
		return
	}
	if text == "" || a.current == "" {
		return
	}
	a.setInputText("")
	var m *Msg
	if a.wa != nil {
		m = a.wa.SendText(a.current, text)
	} else {
		m = &Msg{Chat: a.current, ID: fmt.Sprint(time.Now().UnixNano()), FromMe: true, TS: time.Now().Unix(), Kind: "text", Text: text, Status: StRead, IsRead: true}
		a.st.SaveMessage(m, false)
	}
	a.stickBottom = true
	a.appendMessage(m)
	a.reloadChats()
}

func (a *App) attachFile() {
	if a.current == "" {
		return
	}
	dlg, _ := gtk.FileChooserDialogNewWith2Buttons("Kirim foto / file", a.win, gtk.FILE_CHOOSER_ACTION_OPEN,
		"Batal", gtk.RESPONSE_CANCEL, "Kirim", gtk.RESPONSE_ACCEPT)
	ff, _ := gtk.FileFilterNew()
	ff.SetName("Semua file")
	ff.AddPattern("*")
	fi, _ := gtk.FileFilterNew()
	fi.SetName("Gambar (JPG, PNG)")
	fi.AddMimeType("image/jpeg")
	fi.AddMimeType("image/png")
	dlg.AddFilter(ff)
	dlg.AddFilter(fi)
	capEnt, _ := gtk.EntryNew()
	capEnt.SetPlaceholderText("Keterangan (opsional)")
	dlg.SetExtraWidget(capEnt)
	resp := dlg.Run()
	path := dlg.GetFilename()
	caption := strings.TrimSpace(capEnt.GetChars(0, -1))
	dlg.Destroy()
	if resp != gtk.RESPONSE_ACCEPT || path == "" {
		return
	}
	a.sendFileTo(a.current, path, caption)
}

// ------------------------------------------------------------ dialog

func (a *App) newChatDialog() {
	if a.wa == nil || a.pages.GetVisibleChildName() != "main" {
		return
	}
	dlg, _ := gtk.DialogNew()
	dlg.SetTransientFor(a.win)
	dlg.SetModal(true)
	dlg.SetTitle("Obrolan baru")
	dlg.AddButton("Batal", gtk.RESPONSE_CANCEL)
	ok, _ := dlg.AddButton("Mulai obrolan", gtk.RESPONSE_OK)
	addClass(ok, "suggested-action")
	dlg.SetDefaultResponse(gtk.RESPONSE_OK)
	area, _ := dlg.GetContentArea()
	area.SetSpacing(8)
	area.SetMarginStart(14)
	area.SetMarginEnd(14)
	area.SetMarginTop(10)
	lbl, _ := gtk.LabelNew("Masukkan nomor WhatsApp (boleh diawali 0 atau 62):")
	lbl.SetXAlign(0)
	ent, _ := gtk.EntryNew()
	ent.SetPlaceholderText("0812 3456 7890")
	ent.SetActivatesDefault(true)
	info, _ := gtk.LabelNew("")
	info.SetXAlign(0)
	area.PackStart(lbl, false, false, 0)
	area.PackStart(ent, false, false, 0)
	area.PackStart(info, false, false, 0)
	dlg.ShowAll()
	dlg.Connect("response", func(_ *gtk.Dialog, resp gtk.ResponseType) {
		if resp != gtk.RESPONSE_OK {
			dlg.Destroy()
			return
		}
		num := ent.GetChars(0, -1)
		info.SetText("Memeriksa nomor…")
		go func() {
			jid, err := a.wa.CheckNumber(num)
			glib.IdleAdd(func() {
				if err != nil {
					info.SetMarkup(`<span foreground="#e53935">` + html.EscapeString(err.Error()) + `</span>`)
					return
				}
				dlg.Destroy()
				a.st.EnsureChat(jid, "")
				a.openChat(jid)
			})
		}()
	})
}

func (a *App) confirmLogout() {
	d := gtk.MessageDialogNew(a.win, gtk.DIALOG_MODAL, gtk.MESSAGE_QUESTION, gtk.BUTTONS_YES_NO,
		"Keluar dari akun WhatsApp di Obrol?")
	d.FormatSecondaryText("Perangkat ini akan dilepas dari WhatsApp dan semua riwayat chat lokal dihapus.")
	resp := d.Run()
	d.Destroy()
	if resp == gtk.RESPONSE_YES && a.wa != nil {
		a.header.SetSubtitle("Keluar…")
		go a.wa.Logout()
	}
}

func (a *App) errorDialog(title, msg string) {
	d := gtk.MessageDialogNew(a.win, gtk.DIALOG_MODAL, gtk.MESSAGE_ERROR, gtk.BUTTONS_OK, "%s", title)
	d.FormatSecondaryText("%s", msg)
	d.Run()
	d.Destroy()
}

func (a *App) about() {
	d, _ := gtk.AboutDialogNew()
	d.SetTransientFor(a.win)
	d.SetProgramName("Obrol")
	d.SetVersion(version)
	d.SetLogo(svgPixbuf(logoSVG, 128))
	d.SetAuthors([]string{author})
	d.SetComments("Klien WhatsApp native & super ringan untuk Linux (GTK3 + whatsmeow).\n\nDibuat oleh " + author + ".")
	d.SetLicense("Kode aplikasi: MIT. whatsmeow: MPL-2.0. gotk3: ISC.")
	cr := "© 2026 " + author
	if a.wa != nil {
		cr += "\nMasuk sebagai: " + a.wa.MyName()
	}
	d.SetCopyright(cr)
	d.Run()
	d.Destroy()
}

// ------------------------------------------------------------ mute

// isMuted: chat dibisukan sendiri, atau grup saat "bisukan semua grup" aktif.
func (a *App) isMuted(c Chat) bool {
	return c.Muted() || (a.cfg.MuteGroups && isGroupJID(c.JID))
}

func muteDesc(c Chat) string {
	switch {
	case c.MutedUntil == -1:
		return "Dibisukan selamanya"
	case c.Muted():
		t := time.Unix(c.MutedUntil, 0)
		if dayKey(c.MutedUntil) == dayKey(time.Now().Unix()) {
			return "Dibisukan sampai " + t.Format("15:04")
		}
		return fmt.Sprintf("Dibisukan sampai %s, %s", dayLabelShort(t), t.Format("15:04"))
	}
	return ""
}

func dayLabelShort(t time.Time) string {
	return fmt.Sprintf("%s %d %s", hari[t.Weekday()], t.Day(), bulan[t.Month()][:3])
}

func (a *App) muteMenu(jid string) *gtk.Menu {
	c, ok := a.st.GetChat(jid)
	if !ok {
		c = Chat{JID: jid, IsGroup: isGroupJID(jid)}
	}
	menu, _ := gtk.MenuNew()
	add := func(label string, fn func()) {
		it, _ := gtk.MenuItemNewWithLabel(label)
		if fn == nil {
			it.SetSensitive(false)
		} else {
			it.Connect("activate", fn)
		}
		menu.Append(it)
	}
	addI := func(icon, label string, fn func()) { menu.Append(menuItem(icon, label, fn)) }
	if c.Pinned > 0 {
		addI("pin-off", "Lepas sematan obrolan", func() { a.pinChat(jid, false) })
	} else {
		addI("pin", "Sematkan obrolan", func() { a.pinChat(jid, true) })
	}
	if ps := a.st.Pins(jid); len(ps) > 0 {
		add(fmt.Sprintf("    (%d pesan tersemat di obrolan ini)", len(ps)), nil)
	}
	sp, _ := gtk.SeparatorMenuItemNew()
	menu.Append(sp)
	if c.Muted() {
		addI("bell-off", muteDesc(c), nil)
		addI("bell", "Bunyikan notifikasi", func() { a.setMute(jid, false, 0) })
	} else {
		addI("bell-off", "Bisukan 8 jam", func() { a.setMute(jid, true, 8*time.Hour) })
		addI("bell-off", "Bisukan 1 minggu", func() { a.setMute(jid, true, 7*24*time.Hour) })
		addI("bell-off", "Bisukan selamanya", func() { a.setMute(jid, true, 0) })
		if a.cfg.MuteGroups && c.IsGroup {
			add("(semua grup sedang dibisukan lewat menu ☰)", nil)
		}
	}
	if c.Unread > 0 {
		s, _ := gtk.SeparatorMenuItemNew()
		menu.Append(s)
		addI("check-all", "Tandai sudah dibaca", func() {
			if jid == a.current {
				a.markCurrentRead()
				return
			}
			if a.wa != nil {
				a.wa.MarkRead(jid)
			} else {
				a.st.MarkChatRead(jid)
			}
			a.reloadChats()
		})
	}
	menu.ShowAll()
	return menu
}

func (a *App) setMute(jid string, mute bool, dur time.Duration) {
	if a.wa != nil {
		a.wa.SetMute(jid, mute, dur)
	} else {
		var until int64
		if mute {
			until = -1
			if dur > 0 {
				until = time.Now().Add(dur).Unix()
			}
		}
		a.st.SetMute(jid, until)
	}
	if mute {
		a.flashStatus("Notifikasi dibisukan")
	} else {
		a.flashStatus("Notifikasi dibunyikan kembali")
	}
	a.reloadChats()
	if jid == a.current {
		c, _ := a.st.GetChat(jid)
		a.refreshMuteBtn(c)
	}
}

func (a *App) refreshMuteBtn(c Chat) {
	if a.muteBtn == nil {
		return
	}
	if c.Muted() {
		a.muteIcon.SetName("bell-off")
		a.muteBtn.SetTooltipText(muteDesc(c) + " — klik untuk mengubah")
	} else {
		a.muteIcon.SetName("bell")
		a.muteBtn.SetTooltipText("Sematkan / bisukan obrolan ini")
	}
	a.muteBtn.SetPopup(a.muteMenu(c.JID))
}

// reopenKeepScroll merender ulang chat terbuka tanpa melompatkan posisi scroll.
func (a *App) reopenKeepScroll(chat string) {
	adj := a.msgScroll.GetVAdjustment()
	dist := adj.GetUpper() - adj.GetValue()
	stick := a.stickBottom
	a.openChat(chat)
	if !stick {
		a.stickBottom = false
		a.restoreDist = dist
	}
}

// ------------------------------------------------------------ tray

func (a *App) setupTray() {
	t := NewStatusIcon(a.pixNormal)
	if t == nil {
		return
	}
	a.tray = t
	t.SetTooltipText("Obrol")
	t.SetTitle("Obrol")
	menu, _ := gtk.MenuNew()
	show, _ := gtk.MenuItemNewWithLabel("Tampilkan Obrol")
	show.Connect("activate", a.showWindow)
	q, _ := gtk.MenuItemNewWithLabel("Keluar")
	q.Connect("activate", a.quit)
	menu.Append(show)
	s, _ := gtk.SeparatorMenuItemNew()
	menu.Append(s)
	menu.Append(q)
	menu.ShowAll()
	t.Connect("activate", func() {
		if a.win.IsVisible() && a.win.IsActive() {
			a.win.Hide()
		} else {
			a.showWindow()
		}
	})
	t.Connect("popup-menu", func() { menu.PopupAtPointer(nil) })
	// render ulang logo SVG pas di ukuran panel (tajam di 16–48 px)
	t.Connect("size-changed", func(_ interface{}, size int) bool {
		if size > 0 && size != a.traySize {
			a.traySize = size
			a.updateUnread()
		}
		return true
	})
}

func (a *App) trayPixbuf(unread bool) *gdk.Pixbuf {
	svg, fallback := logoSVG, a.pixNormal
	if unread {
		svg, fallback = logoUnreadSVG, a.pixUnread
	}
	if a.traySize > 0 {
		if pb := svgPixbuf(svg, a.traySize); pb != nil {
			return pb
		}
	}
	return fallback
}

// ------------------------------------------------------------ hooks dari WA

func (a *App) OnQR(code string) {
	glib.IdleAdd(func() {
		png, err := qrcode.Encode(code, qrcode.Medium, 264)
		if err != nil {
			return
		}
		a.qrImage.SetFromPixbuf(pixbufFromBytes(png))
		a.qrSpinner.Stop()
		a.qrSpinner.Hide()
		a.qrRetry.Hide()
		a.header.SetSubtitle("Pindai kode QR dengan ponsel")
	})
}

func (a *App) OnQRTimeout() {
	glib.IdleAdd(func() {
		a.qrImage.Clear()
		a.qrSpinner.Stop()
		a.qrSpinner.Hide()
		a.qrRetry.Show()
	})
}

func (a *App) OnLoggedIn() {
	glib.IdleAdd(func() {
		if a.pages.GetVisibleChildName() != "main" {
			a.pages.SetVisibleChildName("main")
			a.reloadChats()
		}
	})
}

func (a *App) OnLoggedOut() {
	glib.IdleAdd(func() {
		a.closeChat()
		a.reloadChats()
		a.pairLbl.SetText("")
		a.qrSpinner.Show()
		a.qrSpinner.Start()
		a.pages.SetVisibleChildName("login")
		a.showWindow()
	})
}

func (a *App) OnStatus(text string) {
	glib.IdleAdd(func() {
		a.connStatus = text
		a.statusSeq++
		a.header.SetSubtitle(text)
	})
}

// flashStatus menampilkan pesan sementara lalu kembali ke status koneksi.
func (a *App) flashStatus(text string) {
	a.statusSeq++
	seq := a.statusSeq
	a.header.SetSubtitle(text)
	glib.TimeoutAdd(5000, func() bool {
		if seq == a.statusSeq {
			a.header.SetSubtitle(a.connStatus)
		}
		return false
	})
}

func (a *App) OnNewMessage(m *Msg, live bool) {
	glib.IdleAdd(func() {
		if m.Chat == a.current {
			a.appendMessage(m)
			if !m.FromMe && a.focused {
				a.markCurrentRead()
			}
		}
		a.reloadChats()
		if live && !m.FromMe && !(a.focused && m.Chat == a.current) {
			a.notify(m)
		}
	})
}

func (a *App) OnMessageChanged(chat string) {
	glib.IdleAdd(func() {
		if chat == a.current {
			a.reopenKeepScroll(chat)
		}
		a.reloadChats()
	})
}

func (a *App) OnChatsChanged() {
	glib.IdleAdd(func() {
		a.reloadChats()
		// muat ulang percakapan terbuka bila kosong (misal riwayat baru datang)
		if a.current != "" && len(a.metas) == 0 && a.oldestTS == 0 {
			a.openChat(a.current)
		}
	})
}

func (a *App) OnReceipt(ids []string, st int) {
	glib.IdleAdd(func() {
		for _, id := range ids {
			if r, ok := a.metas[id]; ok && (st > r.status || st == StFailed) {
				r.status = st
				a.setMeta(r.label, r.ts, true, st, r.edited)
			}
		}
	})
}

func (a *App) notify(m *Msg) {
	if !a.cfg.Notify {
		return
	}
	c, _ := a.st.GetChat(m.Chat)
	if a.isMuted(c) && !m.Mention {
		return // chat dibisukan: diam, kecuali @mention / balasan ke pesan saya
	}
	title := a.displayName(c)
	if m.Mention && isGroupJID(m.Chat) {
		title = "@ " + title
	}
	body := m.Text
	if lbl := kindLabel(m.Kind); lbl != "" {
		if body != "" && m.Kind != "ptt" && m.Kind != "audio" {
			body = lbl + " · " + body
		} else {
			body = lbl
		}
	}
	if isGroupJID(m.Chat) && m.SenderName != "" {
		body = m.SenderName + ": " + body
	}
	if len([]rune(body)) > 200 {
		body = string([]rune(body)[:200]) + "…"
	}
	n := glib.NotificationNew(title)
	n.SetBody(body)
	if a.iconPath != "" {
		n.SetIcon(a.iconPath)
	}
	n.SetDefaultAction("app.open-chat::" + m.Chat)
	a.gapp.SendNotification(m.Chat, n)
	if a.cfg.Sound {
		go exec.Command("canberra-gtk-play", "-i", "message-new-instant", "-d", "Obrol").Run()
	}
}

// ------------------------------------------------------------ helper

func pixbufFromBytes(data []byte) *gdk.Pixbuf {
	l, err := gdk.PixbufLoaderNew()
	if err != nil {
		return nil
	}
	if _, err := l.Write(data); err != nil {
		l.Close()
		return nil
	}
	if err := l.Close(); err != nil {
		return nil
	}
	pb, err := l.GetPixbuf()
	if err != nil {
		return nil
	}
	return pb
}

func scalePix(pb *gdk.Pixbuf, size int) *gdk.Pixbuf {
	if pb == nil {
		return nil
	}
	s, err := pb.ScaleSimple(size, size, gdk.INTERP_BILINEAR)
	if err != nil {
		return pb
	}
	return s
}

func scaleToWidth(pb *gdk.Pixbuf, w int) *gdk.Pixbuf {
	if pb.GetWidth() == 0 {
		return pb
	}
	h := pb.GetHeight() * w / pb.GetWidth()
	if h > 320 {
		h = 320
		w = pb.GetWidth() * h / pb.GetHeight()
	}
	s, err := pb.ScaleSimple(w, h, gdk.INTERP_BILINEAR)
	if err != nil {
		return pb
	}
	return s
}

func initials(name string, group bool) string {
	var out []rune
	for _, w := range strings.Fields(name) {
		for _, r := range w {
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				out = append(out, unicode.ToUpper(r))
				break
			}
		}
		if len(out) == 2 {
			break
		}
	}
	if len(out) == 0 {
		if group {
			return "👥"
		}
		return "?"
	}
	return string(out)
}

func hashN(s string, n uint32) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32() % n
}

func avatarClass(jid string) string { return fmt.Sprintf("av%d", hashN(jid, 8)) }

var senderColors = []string{"#1f7aec", "#d43c86", "#0b8f6a", "#c4590c", "#7a4fd1", "#b8862b", "#d1393b", "#2e8b9a"}
var senderColorsDark = []string{"#53bdeb", "#ff72a1", "#06cf9c", "#fc9775", "#a791ff", "#ffd279", "#ff6b6b", "#71ebff"}

func senderColor(jid string, dark bool) string {
	if dark {
		return senderColorsDark[hashN(jid, uint32(len(senderColorsDark)))]
	}
	return senderColors[hashN(jid, uint32(len(senderColors)))]
}

var hari = []string{"Minggu", "Senin", "Selasa", "Rabu", "Kamis", "Jumat", "Sabtu"}
var bulan = []string{"", "Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"}

func dayKey(ts int64) string { return time.Unix(ts, 0).Format("2006-01-02") }

func dayLabel(ts int64) string {
	t := time.Unix(ts, 0)
	now := time.Now()
	y, mo, d := now.Date()
	today := time.Date(y, mo, d, 0, 0, 0, 0, now.Location())
	switch {
	case !t.Before(today):
		return "Hari ini"
	case !t.Before(today.AddDate(0, 0, -1)):
		return "Kemarin"
	case !t.Before(today.AddDate(0, 0, -6)):
		return hari[t.Weekday()]
	}
	return fmt.Sprintf("%d %s %d", t.Day(), bulan[t.Month()], t.Year())
}

func shortTime(ts int64) string {
	if ts == 0 {
		return ""
	}
	t := time.Unix(ts, 0)
	now := time.Now()
	y, mo, d := now.Date()
	today := time.Date(y, mo, d, 0, 0, 0, 0, now.Location())
	switch {
	case !t.Before(today):
		return t.Format("15:04")
	case !t.Before(today.AddDate(0, 0, -1)):
		return "Kemarin"
	case !t.Before(today.AddDate(0, 0, -6)):
		return hari[t.Weekday()]
	}
	return t.Format("02/01/06")
}

func formatPhone(user string) string {
	if strings.HasPrefix(user, "62") && len(user) > 8 {
		rest := user[2:]
		var parts []string
		for len(rest) > 4 {
			parts = append(parts, rest[:4])
			rest = rest[4:]
		}
		parts = append(parts, rest)
		if len(parts) >= 2 && len(parts[0]) == 4 && len(parts[len(parts)-1]) < 3 {
			// rapikan sisa pendek
			last := parts[len(parts)-2] + parts[len(parts)-1]
			parts = append(parts[:len(parts)-2], last)
		}
		return "+62 " + strings.Join(parts, "-")
	}
	return "+" + user
}

// ------------------------------------------------------------ autostart

func autostartPath() string {
	cfg, _ := os.UserConfigDir()
	return filepath.Join(cfg, "autostart", "obrol.desktop")
}

func autostartEnabled() bool {
	_, err := os.Stat(autostartPath())
	return err == nil
}

func setAutostart(on bool) {
	p := autostartPath()
	if !on {
		os.Remove(p)
		return
	}
	exe, _ := os.Executable()
	os.MkdirAll(filepath.Dir(p), 0o755)
	os.WriteFile(p, []byte("[Desktop Entry]\nType=Application\nName=Obrol\nComment=Klien WhatsApp native\nExec="+exe+" --hidden\nIcon=obrol\nX-GNOME-Autostart-enabled=true\nHidden=false\n"), 0o644)
}

// plainKind: label jenis media tanpa emoji di depannya.
func plainKind(kind string) string {
	return strings.TrimSpace(strings.TrimLeftFunc(kindLabel(kind), func(r rune) bool { return !unicode.IsLetter(r) }))
}
