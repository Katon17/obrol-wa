package main

// searchstar.go — v1.11: pencarian pesan (global & dalam chat) + pesan
// berbintang. Semua tampilan hasil dibangun malas dan dibongkar saat
// ditinggalkan (clearList + trimMem) supaya tidak menambah RSS menetap.

import (
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
)

// ------------------------------------------------------------ halaman daftar

func (a *App) buildListPageBase() (gtk.IWidget, *gtk.ListBox, *gtk.Label) {
	v, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	hb, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 8)
	hb.SetMarginStart(8)
	hb.SetMarginEnd(8)
	hb.SetMarginTop(8)
	hb.SetMarginBottom(8)
	back, _ := gtk.ButtonNew()
	back.SetRelief(gtk.RELIEF_NONE)
	addClass(back, "iconbtn")
	back.Add(newIcon("reply", 18))
	back.SetTooltipText(T("Kembali"))
	hb.PackStart(back, false, false, 0)
	tl, _ := gtk.LabelNew("")
	tl.SetXAlign(0)
	hb.PackStart(tl, true, true, 0)
	v.PackStart(hb, false, false, 0)
	sw, _ := gtk.ScrolledWindowNew(nil, nil)
	lb, _ := gtk.ListBoxNew()
	lb.SetSelectionMode(gtk.SELECTION_NONE)
	addClass(lb, "msgbg")
	sw.Add(lb)
	v.PackStart(sw, true, true, 0)
	back.Connect("clicked", func() { a.leaveListPage(lb) })
	lb.Connect("row-activated", func(_ *gtk.ListBox, r *gtk.ListBoxRow) {
		n, err := r.GetName()
		if err == nil && n != "" {
			a.openChat(n)
		}
	})
	return v, lb, tl
}

func (a *App) leaveListPage(lb *gtk.ListBox) {
	a.clearList(lb)
	trimMem()
	if a.current != "" {
		a.openChat(a.current)
	} else {
		a.chatStack.SetVisibleChildName("empty")
	}
}

func (a *App) clearList(lb *gtk.ListBox) {
	for {
		r := lb.GetRowAtIndex(0)
		if r == nil {
			return
		}
		lb.Remove(r)
		r.Destroy()
	}
}

// compactRow: baris hasil ringkas (nama chat + waktu + cuplikan satu baris).
func (a *App) compactRow(m *Msg, showChat bool) *gtk.ListBoxRow {
	row, _ := gtk.ListBoxRowNew()
	row.SetName(m.Chat)
	vb, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 2)
	vb.SetMarginStart(12)
	vb.SetMarginEnd(12)
	vb.SetMarginTop(6)
	vb.SetMarginBottom(6)
	top, _ := gtk.LabelNew("")
	top.SetXAlign(0)
	addClass(top, "meta")
	nm := ""
	if showChat {
		if c, ok := a.st.GetChat(m.Chat); ok {
			nm = a.displayName(c) + " · "
		} else if isGroupJID(m.Chat) {
			nm = T("Grup") + " · "
		}
	}
	top.SetText(nm + time.Unix(m.TS, 0).Format("02/01 15:04"))
	txt := m.Text
	if txt == "" {
		txt = kindLabel(m.Kind)
	}
	if m.Starred {
		txt = "★ " + txt
	}
	bl, _ := gtk.LabelNew(txt)
	bl.SetXAlign(0)
	bl.SetEllipsize(pango.ELLIPSIZE_END)
	vb.PackStart(top, false, false, 0)
	vb.PackStart(bl, false, false, 0)
	row.Add(vb)
	return row
}

func (a *App) emptyRow(lb *gtk.ListBox, text string) {
	l, _ := gtk.LabelNew(text)
	addClass(l, "emptyhint")
	l.SetMarginTop(24)
	l.SetMarginBottom(24)
	r, _ := gtk.ListBoxRowNew()
	r.SetSelectable(false)
	r.Add(l)
	lb.Add(r)
}

// ------------------------------------------------------------ bintang

func (a *App) openStarred() {
	a.clearMessages() // baris chat tersembunyi ikut dilepas saat halaman ini tampil
	a.chatStack.SetVisibleChildName("starred")
	a.fillStarred()
}

func (a *App) fillStarred() {
	a.clearList(a.starList)
	msgs, _ := a.st.ListStarred(100)
	a.starTitle.SetMarkup("<b>" + html.EscapeString(T("Pesan berbintang")) +
		"</b>   <small>" + fmt.Sprint(len(msgs)) + "</small>")
	if len(msgs) == 0 {
		a.emptyRow(a.starList, T("Belum ada pesan berbintang.\nKlik kanan pesan → «Bintangi pesan»."))
	} else {
		for i := range msgs {
			a.starList.Add(a.compactRow(&msgs[i], true))
		}
	}
	a.starList.ShowAll()
}

func (a *App) toggleStar(chat, id string, star bool) {
	a.st.SetStarred(chat, id, star)
	if star {
		a.flashStatus(T("Pesan dibintangi"))
	} else {
		a.flashStatus(T("Bintang dilepas"))
	}
	if chat == a.current && !a.searchMode {
		a.reopenKeepScroll(chat)
	}
	if a.chatStack.GetVisibleChildName() == "starred" {
		a.fillStarred()
	}
}

// ------------------------------------------------------------ pencarian

func (a *App) updateSearchMsgRow() {
	if a.smsgRow == nil {
		return
	}
	q := strings.TrimSpace(a.search.GetChars(0, -1))
	if q == "" {
		a.smsgRow.SetVisible(false)
		return
	}
	a.smsgLbl.SetText(fmt.Sprintf(T("Cari pesan: %s"), q))
	a.smsgRow.SetVisible(true)
}

func (a *App) openResults(q string) {
	q = strings.TrimSpace(q)
	if q == "" {
		return
	}
	a.clearMessages()
	a.chatStack.SetVisibleChildName("results")
	a.clearList(a.resList)
	msgs, _ := a.st.SearchMessages(q, "", 100)
	a.resTitle.SetMarkup("<b>" + html.EscapeString(fmt.Sprintf(T("Hasil pencarian: %s"), q)) +
		"</b>   <small>" + fmt.Sprint(len(msgs)) + "</small>")
	if len(msgs) == 0 {
		a.emptyRow(a.resList, T("Tidak ada hasil."))
	} else {
		for i := range msgs {
			a.resList.Add(a.compactRow(&msgs[i], true))
		}
	}
	a.resList.ShowAll()
}

func (a *App) openFindBar() {
	if a.findBar == nil {
		return
	}
	if a.chatStack.GetVisibleChildName() != "chat" {
		return
	}
	a.findBar.Show()
	a.findEntry.GrabFocus()
}

func (a *App) runFind() {
	t, _ := a.findEntry.GetText()
	q := strings.TrimSpace(t)
	if q == "" || a.current == "" {
		return
	}
	msgs, _ := a.st.SearchMessages(q, a.current, 60)
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	a.clearMessages()
	a.searchMode = true
	a.noMore = true
	a.prependBatch(msgs)
	a.findLbl.SetText(fmt.Sprintf(T("%d hasil — Esc untuk kembali"), len(msgs)))
}

func (a *App) closeFind() {
	was := a.searchMode
	a.searchMode = false
	if a.findBar != nil {
		a.findBar.Hide()
	}
	if was && a.current != "" {
		a.openChat(a.current)
	}
}
