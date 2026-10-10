package main

// features.go — fitur v1.2: input multi-baris, tempel gambar (Ctrl+V),
// edit/hapus pesan, sematkan pesan & obrolan.

import (
	"fmt"
	"html"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
)

const author = "Katon Bagaskoro"

// ------------------------------------------------------------ composer

func (a *App) buildComposer() gtk.IWidget {
	outer, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)

	// bar "sedang mengedit"
	a.editBar, _ = gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 8)
	addClass(a.editBar, "editbar")
	a.editBar.SetNoShowAll(true)
	ic := newIcon("edit", 16)
	addClass(ic, "accent")
	a.editLbl, _ = gtk.LabelNew("")
	a.editLbl.SetXAlign(0)
	a.editLbl.SetEllipsize(pango.ELLIPSIZE_END)
	x := iconButton("close", "Batal edit (Esc)", 16)
	x.Connect("clicked", a.cancelEdit)
	a.editBar.PackStart(ic, false, false, 0)
	a.editBar.PackStart(a.editLbl, true, true, 0)
	a.editBar.PackEnd(x, false, false, 0)
	ic.Show()
	a.editLbl.Show()
	x.Show()
	outer.PackStart(a.editBar, false, false, 0)

	comp, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 4)
	addClass(comp, "composer")

	attach := iconButton("attach", "Kirim foto / file", 20)
	attach.SetVAlign(gtk.ALIGN_END)
	attach.Connect("clicked", a.attachFile)

	emoji := iconButton("emoji", "Emoji (Ctrl+.)", 20)
	emoji.SetVAlign(gtk.ALIGN_END)
	emoji.Connect("clicked", func() {
		a.input.GrabFocus()
		a.input.Emit("insert-emoji", glib.TYPE_NONE)
	})

	box, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	addClass(box, "inputbox")
	ov, _ := gtk.OverlayNew()
	sw, _ := gtk.ScrolledWindowNew(nil, nil)
	sw.SetPolicy(gtk.POLICY_NEVER, gtk.POLICY_EXTERNAL)
	sw.SetMinContentHeight(18)
	a.inputSW = sw
	a.input, _ = gtk.TextViewNew()
	a.input.SetWrapMode(gtk.WRAP_WORD_CHAR)
	a.input.SetAcceptsTab(false)
	a.inputBuf, _ = a.input.GetBuffer()
	sw.Add(a.input)
	a.placeholder, _ = gtk.LabelNew("Ketik pesan…   Enter kirim · Shift+Enter baris baru · Ctrl+V tempel gambar")
	addClass(a.placeholder, "placeholder")
	a.placeholder.SetHAlign(gtk.ALIGN_START)
	a.placeholder.SetVAlign(gtk.ALIGN_START)
	a.placeholder.SetEllipsize(pango.ELLIPSIZE_END)
	ov.Add(sw)
	ov.AddOverlay(a.placeholder)
	ov.SetOverlayPassThrough(a.placeholder, true)
	box.PackStart(ov, true, true, 0)
	a.inputBuf.Connect("changed", func() {
		a.placeholder.SetVisible(a.inputBuf.GetCharCount() == 0)
		glib.IdleAddPriority(glib.PRIORITY_LOW, a.fitInput)
	})
	a.input.Connect("size-allocate", func() {
		if w := a.input.GetAllocatedWidth(); w != a.inputW {
			a.inputW = w
			glib.IdleAddPriority(glib.PRIORITY_LOW, a.fitInput)
		}
	})
	a.input.Connect("key-press-event", a.onInputKey)

	send := iconButton("send", "Kirim (Enter)", 18)
	send.SetRelief(gtk.RELIEF_NORMAL)
	addClass(send, "sendbtn")
	send.SetVAlign(gtk.ALIGN_END)
	send.Connect("clicked", a.sendCurrent)
	addClass(send, "suggested-action")

	comp.PackStart(attach, false, false, 0)
	comp.PackStart(emoji, false, false, 0)
	comp.PackStart(a.buildStickerButton(), false, false, 0)
	comp.PackStart(box, true, true, 2)
	comp.PackStart(send, false, false, 0)
	outer.PackStart(comp, false, false, 0)
	return outer
}

const inputMaxH = 150 // ±7 baris, sesudahnya scroll

// fitInput menyesuaikan tinggi kotak ketik dengan isi (1 s/d ±7 baris).
func (a *App) fitInput() bool {
	h := 0
	iter := a.inputBuf.GetStartIter()
	for {
		y, lh := a.input.GetLineYrange(iter)
		h = y + lh
		if !a.input.ForwardDisplayLine(iter) {
			// baris terakhir yang kosong (setelah Shift+Enter) ikut dihitung
			if y2, lh2 := a.input.GetLineYrange(iter); y2+lh2 > h {
				h = y2 + lh2
			}
			break
		}
	}
	if h < 18 {
		h = 18
	}
	if h > inputMaxH {
		h = inputMaxH
	}
	if h != a.inputH {
		a.inputH = h
		if h >= inputMaxH {
			a.inputSW.SetPolicy(gtk.POLICY_NEVER, gtk.POLICY_AUTOMATIC)
		} else {
			a.inputSW.SetPolicy(gtk.POLICY_NEVER, gtk.POLICY_EXTERNAL)
		}
		a.inputSW.SetMinContentHeight(h)
	}
	if h < inputMaxH {
		a.inputSW.GetVAdjustment().SetValue(0)
	}
	return false
}

func (a *App) inputText() string {
	start, end := a.inputBuf.GetBounds()
	s, _ := a.inputBuf.GetText(start, end, false)
	return s
}

func (a *App) setInputText(s string) {
	a.inputBuf.SetText(s)
	a.inputBuf.PlaceCursor(a.inputBuf.GetEndIter())
}

func (a *App) onInputKey(_ *gtk.TextView, ev *gdk.Event) bool {
	k := gdk.EventKeyNewFromEvent(ev)
	kv := k.KeyVal()
	shift := k.State()&uint(gdk.SHIFT_MASK) != 0
	ctrl := k.State()&uint(gdk.CONTROL_MASK) != 0
	switch {
	case (kv == gdk.KEY_Return || kv == gdk.KEY_KP_Enter) && !shift && !ctrl:
		a.sendCurrent()
		return true
	case ctrl && (kv == gdk.KEY_v || kv == gdk.KEY_V):
		return a.tryPaste()
	case kv == gdk.KEY_Up && a.editID == "" && strings.TrimSpace(a.inputText()) == "":
		return a.editLast()
	case kv == gdk.KEY_Escape && a.editID != "":
		a.cancelEdit()
		return true
	}
	return false // Shift+Enter → baris baru (perilaku bawaan TextView)
}

// ------------------------------------------------------------ tempel (Ctrl+V)

// tryPaste: bila clipboard berisi gambar (mis. hasil screenshot) atau file
// yang di-copy dari Thunar, tampilkan dialog kirim. Selain itu → tempel teks biasa.
func (a *App) tryPaste() bool {
	if a.current == "" {
		return false
	}
	cb, err := gtk.ClipboardGet(gdk.SELECTION_CLIPBOARD)
	if err != nil {
		return false
	}
	if cb.WaitIsImageAvailable() {
		if pb, err := cb.WaitForImage(); err == nil && pb != nil {
			a.confirmImage(pb, "")
			return true
		}
	}
	// file yang di-copy dari Thunar/Nemo/Nautilus (Ctrl+C di file manager)
	for _, t := range []string{"x-special/gnome-copied-files", "text/uri-list"} {
		if data := clipboardTarget(t); len(data) > 0 {
			if files := parseFileList(string(data)); len(files) > 0 {
				a.confirmFiles(files)
				return true
			}
		}
	}
	if txt, err := cb.WaitForText(); err == nil {
		if files := parseFileList(txt); len(files) > 0 {
			a.confirmFiles(files)
			return true
		}
	}
	return false
}

func parseFileList(txt string) []string {
	var out []string
	for _, ln := range strings.Split(strings.TrimSpace(txt), "\n") {
		ln = strings.TrimSpace(ln)
		if ln == "" || strings.HasPrefix(ln, "#") || ln == "copy" || ln == "cut" {
			continue
		}
		p := ln
		if strings.HasPrefix(ln, "file://") {
			u, err := url.Parse(ln)
			if err != nil {
				return nil
			}
			p = u.Path
		}
		if !strings.HasPrefix(p, "/") {
			return nil
		}
		st, err := os.Stat(p)
		if err != nil || !st.Mode().IsRegular() {
			return nil
		}
		out = append(out, p)
	}
	return out
}

func isImagePath(p string) bool {
	switch strings.ToLower(filepath.Ext(p)) {
	case ".jpg", ".jpeg", ".png":
		return true
	}
	return false
}

// confirmImage: dialog pratinjau + keterangan sebelum gambar dikirim.
func (a *App) confirmImage(pb *gdk.Pixbuf, path string) {
	chat := a.current
	dlg, _ := gtk.DialogNew()
	dlg.SetTransientFor(a.win)
	dlg.SetModal(true)
	dlg.SetTitle("Kirim gambar ke " + a.chatTitle.GetLabel())
	dlg.AddButton("Batal", gtk.RESPONSE_CANCEL)
	if webpEncoderAvailable() {
		st, _ := dlg.AddButton("Jadikan stiker…", gtk.RESPONSE_APPLY)
		st.SetTooltipText("Ubah gambar ini jadi stiker WhatsApp")
	}
	ok, _ := dlg.AddButton("Kirim", gtk.RESPONSE_OK)
	addClass(ok, "suggested-action")
	dlg.SetDefaultResponse(gtk.RESPONSE_OK)
	area, _ := dlg.GetContentArea()
	area.SetSpacing(10)
	area.SetMarginStart(12)
	area.SetMarginEnd(12)
	area.SetMarginTop(12)

	w, h := pb.GetWidth(), pb.GetHeight()
	const maxW, maxH = 560, 400
	pw, ph := w, h
	if pw > maxW {
		pw, ph = maxW, h*maxW/w
	}
	if ph > maxH {
		pw, ph = pw*maxH/ph, maxH
	}
	preview := pb
	if pw != w || ph != h {
		if s, err := pb.ScaleSimple(max(pw, 1), max(ph, 1), gdk.INTERP_BILINEAR); err == nil {
			preview = s
		}
	}
	img, _ := gtk.ImageNewFromPixbuf(preview)
	info, _ := gtk.LabelNew(fmt.Sprintf("%d × %d px", w, h))
	addClass(info, "when")
	cap, _ := gtk.EntryNew()
	cap.SetPlaceholderText("Tambahkan keterangan… (opsional)")
	cap.SetActivatesDefault(true)
	cap.SetProperty("show-emoji-icon", true)
	area.PackStart(img, false, false, 0)
	area.PackStart(info, false, false, 0)
	area.PackStart(cap, false, false, 0)
	dlg.ShowAll()
	cap.GrabFocus()
	resp := dlg.Run()
	caption := strings.TrimSpace(cap.GetChars(0, -1))
	dlg.Destroy()
	if resp == gtk.RESPONSE_APPLY {
		a.createStickerDialog(pb)
		return
	}
	if resp != gtk.RESPONSE_OK {
		a.input.GrabFocus()
		return
	}
	temp := false
	if path == "" {
		dir := filepath.Join(os.TempDir(), "obrol")
		os.MkdirAll(dir, 0o700)
		path = filepath.Join(dir, "tempelan-"+time.Now().Format("20060102-150405")+".jpg")
		if err := pb.SaveJPEG(path, 92); err != nil {
			path = strings.TrimSuffix(path, ".jpg") + ".png"
			if err := pb.SavePNG(path, 6); err != nil {
				a.errorDialog("Gagal menyiapkan gambar", err.Error())
				return
			}
		}
		temp = true
	}
	a.sendFileTo(chat, path, caption)
	if temp {
		os.Remove(path)
	}
	a.input.GrabFocus()
}

func (a *App) confirmFiles(files []string) {
	if len(files) == 1 && isImagePath(files[0]) {
		if pb, err := gdk.PixbufNewFromFile(files[0]); err == nil {
			a.confirmImage(pb, files[0])
			return
		}
	}
	names := make([]string, 0, len(files))
	for _, f := range files {
		names = append(names, "• "+filepath.Base(f))
	}
	d := gtk.MessageDialogNew(a.win, gtk.DIALOG_MODAL, gtk.MESSAGE_QUESTION, gtk.BUTTONS_OK_CANCEL,
		"Kirim %d file ke %s?", len(files), a.chatTitle.GetLabel())
	d.FormatSecondaryText("%s", strings.Join(names, "\n"))
	resp := d.Run()
	d.Destroy()
	if resp != gtk.RESPONSE_OK {
		return
	}
	chat := a.current
	for _, f := range files {
		a.sendFileTo(chat, f, "")
	}
}

// sendFileTo mengirim file/gambar dan menampilkannya di percakapan.
func (a *App) sendFileTo(chat, path, caption string) {
	var m *Msg
	if a.wa != nil {
		var err error
		m, err = a.wa.SendFile(chat, path, caption)
		if err != nil {
			a.errorDialog("Gagal mengirim file", err.Error())
			return
		}
	} else { // mode demo
		data, err := os.ReadFile(path)
		if err != nil {
			return
		}
		m = &Msg{Chat: chat, ID: fmt.Sprint(time.Now().UnixNano()), FromMe: true, TS: time.Now().Unix(),
			Kind: "document", Text: filepath.Base(path), Status: StSent, IsRead: true}
		if isImagePath(path) {
			m.Kind, m.Text = "image", caption
			m.Thumb, _, _ = makeThumbSize(data, 520, 82)
		}
		a.st.SaveMessage(m, false)
	}
	if chat == a.current {
		a.stickBottom = true
		a.appendMessage(m)
	}
	a.reloadChats()
}

// ------------------------------------------------------------ edit pesan

func (a *App) canEdit(m Msg) bool {
	return m.FromMe && m.Kind == "text" && (a.wa == nil || time.Since(time.Unix(m.TS, 0)) <= EditWindow)
}

func (a *App) startEdit(chat, id string) {
	m, ok := a.st.GetMessage(chat, id)
	if !ok || chat != a.current {
		return
	}
	if !a.canEdit(m) {
		a.flashStatus("Pesan hanya bisa diedit maksimal 15 menit setelah dikirim")
		return
	}
	a.editID, a.editChat = id, chat
	a.editLbl.SetMarkup("<b>Edit pesan</b>  <span alpha='70%'>" + escape(oneLine(m.Text, 80)) + "</span>")
	a.editBar.Show()
	a.setInputText(m.Text)
	a.input.GrabFocus()
}

func (a *App) cancelEdit() {
	if a.editID == "" {
		return
	}
	a.editID, a.editChat = "", ""
	a.editBar.Hide()
	a.setInputText("")
}

func (a *App) finishEdit(text string) {
	chat, id := a.editChat, a.editID
	orig, _ := a.st.GetMessage(chat, id)
	a.cancelEdit()
	if text == "" || text == orig.Text {
		return
	}
	if a.wa != nil {
		if err := a.wa.EditText(chat, id, text); err != nil {
			a.errorDialog("Tidak bisa mengedit", err.Error())
			return
		}
	} else {
		a.st.EditMessage(chat, id, text)
	}
	a.flashStatus("Pesan diedit")
	if chat == a.current {
		a.reopenKeepScroll(chat)
	}
	a.reloadChats()
}

// editLast: tombol ↑ di kolom kosong → edit pesan terakhir saya.
func (a *App) editLast() bool {
	if a.current == "" {
		return false
	}
	msgs, _ := a.st.LoadMessages(a.current, 0, 40)
	for i := len(msgs) - 1; i >= 0; i-- {
		if a.canEdit(msgs[i]) {
			a.startEdit(a.current, msgs[i].ID)
			return true
		}
	}
	return false
}

// ------------------------------------------------------------ menu pesan

// fillMsgMenu menambahkan aksi pesan ke menu (klik kanan bubble / teks).
func (a *App) fillMsgMenu(menu *gtk.Menu, chat, id string, withCopy bool) {
	m, ok := a.st.GetMessage(chat, id)
	if !ok {
		return
	}
	add := func(icon, label string, fn func()) *gtk.MenuItem {
		it := menuItem(icon, label, fn)
		menu.Append(it)
		return it
	}
	if m.Kind == "deleted" {
		add("ban", "Pesan sudah dihapus", nil)
		return
	}
	if m.Kind == "sticker" {
		a.stickerMenuItems(menu, m)
	}
	if withCopy && m.Text != "" {
		add("copy", "Salin teks", func() {
			if cb, err := gtk.ClipboardGet(gdk.SELECTION_CLIPBOARD); err == nil {
				cb.SetText(m.Text)
				a.flashStatus("Teks disalin")
			}
		})
	}
	if m.FromMe && m.Kind == "text" {
		if a.canEdit(m) {
			add("edit", "Edit pesan", func() { a.startEdit(chat, id) })
		} else {
			add("edit", "Edit (lewat batas 15 menit)", nil)
		}
	}
	if a.st.IsPinned(chat, id) {
		add("pin-off", "Lepas sematan", func() { a.pinMessage(chat, id, false, 0) })
	} else {
		pin := add("pin", "Sematkan pesan", func() {})
		sub, _ := gtk.MenuNew()
		for _, o := range []struct {
			label string
			d     time.Duration
		}{{"24 jam", 24 * time.Hour}, {"7 hari", 7 * 24 * time.Hour}, {"30 hari", 30 * 24 * time.Hour}} {
			d := o.d
			it, _ := gtk.MenuItemNewWithLabel(o.label)
			it.Connect("activate", func() { a.pinMessage(chat, id, true, d) })
			sub.Append(it)
		}
		pin.SetSubmenu(sub)
	}
	if m.FromMe && (a.wa == nil || time.Since(time.Unix(m.TS, 0)) <= RevokeWindow) {
		sep, _ := gtk.SeparatorMenuItemNew()
		menu.Append(sep)
		add("trash", "Hapus untuk semua orang…", func() { a.confirmRevoke(chat, id) })
	}
}

func (a *App) popupMsgMenu(ev *gdk.Event, chat, id string) {
	menu, _ := gtk.MenuNew()
	a.fillMsgMenu(menu, chat, id, true)
	menu.ShowAll()
	menu.PopupAtPointer(ev)
}

func (a *App) confirmRevoke(chat, id string) {
	d := gtk.MessageDialogNew(a.win, gtk.DIALOG_MODAL, gtk.MESSAGE_QUESTION, gtk.BUTTONS_YES_NO,
		"Hapus pesan ini untuk semua orang?")
	d.FormatSecondaryText("Pesan akan diganti dengan \"Pesan ini dihapus\" di semua perangkat.")
	resp := d.Run()
	d.Destroy()
	if resp != gtk.RESPONSE_YES {
		return
	}
	if a.wa != nil {
		a.wa.Revoke(chat, id)
	} else {
		a.st.RevokeMessage(chat, id)
	}
	if chat == a.current {
		a.reopenKeepScroll(chat)
	}
	a.reloadChats()
}

// ------------------------------------------------------------ sematkan pesan

func (a *App) pinMessage(chat, id string, pin bool, d time.Duration) {
	if a.wa != nil {
		if err := a.wa.PinMessage(chat, id, pin, d); err != nil {
			a.errorDialog("Gagal menyematkan", err.Error())
			return
		}
	} else if pin {
		a.st.AddPin(chat, id, time.Now().Add(d).Unix())
	} else {
		a.st.RemovePin(chat, id)
	}
	if pin {
		a.flashStatus("Pesan disematkan")
		a.pinIdx = 0
	} else {
		a.flashStatus("Sematan dilepas")
	}
	if chat == a.current {
		a.refreshPins()
	}
}

func (a *App) buildPinBar() gtk.IWidget {
	a.pinBar, _ = gtk.EventBoxNew()
	a.pinBar.SetNoShowAll(true)
	addClass(a.pinBar, "pinbar")
	h, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 8)
	ic := newIcon("pin", 18)
	addClass(ic, "accent")
	v, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	a.pinTitle, _ = gtk.LabelNew("")
	addClass(a.pinTitle, "pintitle")
	a.pinTitle.SetXAlign(0)
	a.pinLbl, _ = gtk.LabelNew("")
	a.pinLbl.SetXAlign(0)
	a.pinLbl.SetEllipsize(pango.ELLIPSIZE_END)
	v.PackStart(a.pinTitle, false, false, 0)
	v.PackStart(a.pinLbl, false, false, 0)
	un, _ := gtk.ButtonNewWithLabel("Lepas")
	un.SetRelief(gtk.RELIEF_NONE)
	un.SetTooltipText("Lepas sematan pesan ini")
	un.SetVAlign(gtk.ALIGN_CENTER)
	un.Connect("clicked", func() {
		if a.pinIdx < len(a.pins) {
			p := a.pins[a.pinIdx]
			a.pinMessage(p.Chat, p.ID, false, 0)
		}
	})
	h.PackStart(ic, false, false, 0)
	h.PackStart(v, true, true, 0)
	h.PackEnd(un, false, false, 0)
	a.pinBar.Add(h)
	h.ShowAll()
	a.pinBar.SetTooltipText("Klik untuk melompat ke pesan tersemat")
	a.pinBar.Connect("button-press-event", func(_ *gtk.EventBox, ev *gdk.Event) bool {
		if gdk.EventButtonNewFromEvent(ev).Button() != gdk.BUTTON_PRIMARY || len(a.pins) == 0 {
			return false
		}
		p := a.pins[a.pinIdx%len(a.pins)]
		a.jumpTo(p.ID)
		// klik berikutnya → pesan tersemat berikutnya (seperti WhatsApp)
		a.pinIdx = (a.pinIdx + 1) % len(a.pins)
		a.updatePinLabel()
		return true
	})
	return a.pinBar
}

func (a *App) refreshPins() {
	if a.current == "" {
		a.pinBar.Hide()
		return
	}
	a.pins = a.st.Pins(a.current)
	if len(a.pins) == 0 {
		a.pinBar.Hide()
		return
	}
	if a.pinIdx >= len(a.pins) {
		a.pinIdx = 0
	}
	a.updatePinLabel()
	a.pinBar.Show()
}

func (a *App) updatePinLabel() {
	if len(a.pins) == 0 {
		return
	}
	p := a.pins[a.pinIdx%len(a.pins)]
	title := "Pesan tersemat"
	if len(a.pins) > 1 {
		title = fmt.Sprintf("Pesan tersemat %d dari %d", a.pinIdx%len(a.pins)+1, len(a.pins))
	}
	if p.Until > 0 {
		title += " · sampai " + dayLabelShort(time.Unix(p.Until, 0))
	}
	a.pinTitle.SetText(title)
	if m, ok := a.st.GetMessage(p.Chat, p.ID); ok {
		a.pinLbl.SetText(previewText(&m))
	} else {
		a.pinLbl.SetText("(pesan tidak ada di riwayat lokal)")
	}
}

// jumpTo menggulir ke pesan tertentu (memuat riwayat lama bila perlu).
func (a *App) jumpTo(id string) {
	if row, ok := a.rowsByID[id]; ok {
		a.scrollToRow(row)
		return
	}
	m, ok := a.st.GetMessage(a.current, id)
	if !ok {
		a.flashStatus("Pesan tersemat tidak ada di riwayat lokal")
		return
	}
	n := a.st.CountSince(a.current, m.TS) + 10
	a.clearMessages()
	msgs, _ := a.st.LoadMessages(a.current, 0, n)
	a.noMore = len(msgs) < n
	a.stickBottom = false
	a.prependBatch(msgs)
	glib.TimeoutAdd(200, func() bool {
		if row, ok := a.rowsByID[id]; ok {
			a.scrollToRow(row)
		}
		return false
	})
}

func (a *App) scrollToRow(row *gtk.ListBoxRow) {
	a.stickBottom = false
	alloc := row.GetAllocation()
	adj := a.msgScroll.GetVAdjustment()
	adj.SetValue(float64(alloc.GetY()) - adj.GetPageSize()/3)
	addClass(row, "flash")
	glib.TimeoutAdd(1600, func() bool {
		if sc, err := row.GetStyleContext(); err == nil {
			sc.RemoveClass("flash")
		}
		return false
	})
}

// ------------------------------------------------------------ sematkan obrolan

func (a *App) pinChat(jid string, pin bool) {
	if pin && a.st.PinnedChatCount() >= 3 {
		a.errorDialog("Maksimal 3 obrolan disematkan", "Lepas sematan salah satu obrolan dulu (sama seperti di WhatsApp).")
		return
	}
	if a.wa != nil {
		a.wa.PinChat(jid, pin)
	} else {
		a.st.SetChatPinned(jid, pin)
	}
	if pin {
		a.flashStatus("Obrolan disematkan di atas")
	} else {
		a.flashStatus("Sematan obrolan dilepas")
	}
	a.reloadChats()
	if jid == a.current {
		c, _ := a.st.GetChat(jid)
		a.refreshMuteBtn(c)
	}
}

// ------------------------------------------------------------ util

func oneLine(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if r := []rune(s); len(r) > n {
		return string(r[:n]) + "…"
	}
	return s
}

func escape(s string) string { return html.EscapeString(s) }
