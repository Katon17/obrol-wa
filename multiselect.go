package main

// multiselect.go — v1.13: mode pilih beberapa pesan (ceklist) + aksi massal.

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
)

// selW: widget per pesan untuk mode seleksi (v1.13).
type selW struct {
	id  string
	box *gtk.Box // wadah bulatan ceklist (tersembunyi di luar mode)
	off *Icon
	on  *Icon
	lbl *gtk.Label // label teks pesan (selectable dimatikan selama mode)
}

// buildSelBar: baris aksi mode seleksi di atas daftar pesan (v1.13).
func (a *App) buildSelBar() *gtk.Box {
	bar, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 6)
	addClass(bar, "selbar")
	bar.SetNoShowAll(true) // kebal ShowAll() jendela; tampil hanya via enterSelect (v1.13)
	bar.SetMarginStart(8)
	bar.SetMarginEnd(8)
	bar.SetMarginTop(4)
	bar.SetMarginBottom(4)
	a.selLbl, _ = gtk.LabelNew("")
	addClass(a.selLbl, "selcount")
	a.selLbl.SetXAlign(0)
	a.selLbl.Show()
	bar.PackStart(a.selLbl, true, true, 0)
	mk := func(icon, label string, fn func()) {
		bt, _ := gtk.ButtonNew()
		bt.SetRelief(gtk.RELIEF_NONE)
		addClass(bt, "iconbtn")
		hb, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 4)
		hb.PackStart(newIcon(icon, 15), false, false, 0)
		lb, _ := gtk.LabelNew(T(label))
		hb.PackStart(lb, false, false, 0)
		bt.Add(hb)
		hb.ShowAll()
		bt.Show()
		bt.SetTooltipText(T(label))
		bt.Connect("clicked", func() { fn() })
		bar.PackStart(bt, false, false, 0)
	}
	mk("forward", T("Teruskan…"), a.selForward)
	mk("star", T("Bintang"), a.selStar)
	mk("copy", T("Salin teks"), a.selCopy)
	mk("trash", T("Hapus…"), a.selDeleteDialog)
	x, _ := gtk.ButtonNew()
	x.SetRelief(gtk.RELIEF_NONE)
	addClass(x, "iconbtn")
	x.Add(newIcon("close", 14))
	x.SetTooltipText(T("Batal"))
	x.Connect("clicked", func() { a.exitSelect() })
	bar.PackStart(x, false, false, 0)
	x.Show()
	bar.Hide() // menunggu mode seleksi (v1.13)
	return bar
}

// enterSelect: masuk mode seleksi dengan pesan id sudah tercentang (v1.13).
func (a *App) enterSelect(id string) {
	if a.selMode {
		a.toggleSel(id)
		return
	}
	a.selMode = true
	a.sel = map[string]bool{id: true}
	for mid, w := range a.selW {
		w.box.Show()
		a.setSelCheck(w, a.sel[mid])
		if w.lbl != nil {
			w.lbl.SetSelectable(false)
		}
	}
	a.selBar.Show()
	a.updateSelBar()
}

// exitSelect: keluar mode seleksi tanpa menyentuh data (v1.13).
func (a *App) exitSelect() {
	if !a.selMode {
		return
	}
	a.selMode = false
	a.sel = nil
	for _, w := range a.selW {
		a.setSelCheck(w, false)
		w.box.Hide()
		if w.lbl != nil {
			w.lbl.SetSelectable(true)
		}
	}
	a.selBar.Hide()
}

func (a *App) setSelCheck(w *selW, on bool) {
	if on {
		w.off.Hide()
		w.on.Show()
	} else {
		w.on.Hide()
		w.off.Show()
	}
}

// toggleSel: centang / lepas satu pesan (v1.13).
func (a *App) toggleSel(id string) {
	if !a.selMode {
		return
	}
	if a.sel[id] {
		delete(a.sel, id)
	} else {
		a.sel[id] = true
	}
	if w, ok := a.selW[id]; ok {
		a.setSelCheck(w, a.sel[id])
	}
	a.updateSelBar()
}

func (a *App) updateSelBar() {
	if a.selLbl != nil {
		a.selLbl.SetText(fmt.Sprintf(T("%d dipilih"), len(a.sel)))
	}
}

// selMsgs: pesan terpilih urut waktu (v1.13).
func (a *App) selMsgs() []Msg {
	var out []Msg
	for id := range a.sel {
		if m, ok := a.st.GetMessage(a.current, id); ok {
			out = append(out, m)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TS < out[j].TS })
	return out
}

func (a *App) selStar() {
	ms := a.selMsgs()
	if len(ms) == 0 {
		return
	}
	all := true
	for _, m := range ms {
		if !m.Starred {
			all = false
		}
	}
	for _, m := range ms {
		a.st.SetStarred(a.current, m.ID, !all)
	}
	chat := a.current
	a.exitSelect()
	a.OnMessageChanged(chat)
	if all {
		a.flashStatus(T("Bintang dilepas"))
	} else {
		a.flashStatus(T("Pesan dibintangi"))
	}
}

func (a *App) selCopy() {
	ms := a.selMsgs()
	var parts []string
	for _, m := range ms {
		if m.Text != "" {
			parts = append(parts, m.Text)
		}
	}
	a.exitSelect()
	if len(parts) == 0 {
		return
	}
	if cb, err := gtk.ClipboardGet(gdk.SELECTION_CLIPBOARD); err == nil {
		cb.SetText(strings.Join(parts, "\n\n"))
		a.flashStatus(T("Teks disalin"))
	}
}

func (a *App) selForward() {
	ms := a.selMsgs()
	if len(ms) == 0 {
		return
	}
	a.exitSelect()
	a.forwardDialog(ms)
}

// selDeleteDialog: hapus massal untuk saya / untuk semua orang (v1.13).
func (a *App) selDeleteDialog() {
	ms := a.selMsgs()
	if len(ms) == 0 {
		return
	}
	allMine := true
	for _, m := range ms {
		if !m.FromMe || time.Since(time.Unix(m.TS, 0)) > RevokeWindow {
			allMine = false
		}
	}
	dlg := gtk.MessageDialogNew(a.win, gtk.DIALOG_MODAL, gtk.MESSAGE_QUESTION, gtk.BUTTONS_NONE,
		"%s", fmt.Sprintf(T("Hapus %d pesan?"), len(ms)))
	dlg.AddButton(T("Batal"), gtk.RESPONSE_CANCEL)
	dlg.AddButton(T("Hapus untuk saya"), gtk.RESPONSE_YES)
	if allMine && a.wa != nil {
		dlg.AddButton(T("Hapus untuk semua orang"), gtk.RESPONSE_ACCEPT)
	}
	resp := dlg.Run()
	dlg.Destroy()
	ids := make([]string, 0, len(ms))
	for _, m := range ms {
		ids = append(ids, m.ID)
	}
	chat := a.current
	switch resp {
	case gtk.RESPONSE_YES:
		a.st.DeleteMessages(chat, ids)
		a.exitSelect()
		a.OnMessageChanged(chat)
	case gtk.RESPONSE_ACCEPT:
		a.exitSelect()
		go func() {
			for _, id := range ids {
				_ = a.wa.Revoke(chat, id)
			}
		}()
		a.OnMessageChanged(chat)
	}
}
