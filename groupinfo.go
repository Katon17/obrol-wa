package main

// groupinfo.go — v1.12: info & manajemen grup. Halaman dibangun malas saat
// header grup diklik; daftar peserta dimuat per halaman (50) via gulir dan
// seluruhnya dibongkar saat halaman ditinggalkan (clearList + trimMem),
// sehingga grup beranggota ratusan orang tidak menambah RSS menetap.

import (
	"fmt"
	"html"
	"os"
	"strings"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
)

type giParticipant struct {
	jid   string
	name  string
	admin bool
	me    bool
}

const giPageSize = 50

func (a *App) buildGroupPage() gtk.IWidget {
	v, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)

	// v1.12.1: kepala halaman padat — kembali | foto 84px | judul+jumlah.
	hdr, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 10)
	hdr.SetMarginStart(8)
	hdr.SetMarginEnd(10)
	hdr.SetMarginTop(8)
	hdr.SetMarginBottom(6)
	back, _ := gtk.ButtonNew()
	back.SetRelief(gtk.RELIEF_NONE)
	addClass(back, "iconbtn")
	back.Add(newIcon("reply", 18))
	back.SetTooltipText(T("Kembali"))
	back.SetVAlign(gtk.ALIGN_START)
	back.Connect("clicked", func() { a.leaveGroupPage() })
	hdr.PackStart(back, false, false, 0)
	a.giAvBox, _ = gtk.EventBoxNew()
	a.giAvBox.SetSizeRequest(84, 84)
	a.giAvBox.SetVAlign(gtk.ALIGN_CENTER)
	addClass(a.giAvBox, "giphoto")
	a.giPlaceholder()
	hdr.PackStart(a.giAvBox, false, false, 0)
	tb, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 1)
	a.giTitle, _ = gtk.LabelNew("")
	a.giTitle.SetXAlign(0)
	a.giTitle.SetEllipsize(pango.ELLIPSIZE_END)
	a.giTitle.SetMaxWidthChars(34)
	a.giSub, _ = gtk.LabelNew("")
	addClass(a.giSub, "when")
	a.giSub.SetXAlign(0)
	tb.PackStart(a.giTitle, false, false, 0)
	tb.PackStart(a.giSub, false, false, 0)
	hdr.PackStart(tb, true, true, 0)
	v.PackStart(hdr, false, false, 0)

	// v1.12.1: deskripsi dikurung guliran setinggi ±4 baris — deskripsi
	// sepanjang apa pun tidak lagi mendorong daftar peserta keluar layar.
	a.giDesc, _ = gtk.LabelNew("")
	a.giDesc.SetXAlign(0)
	a.giDesc.SetLineWrap(true)
	a.giDesc.SetSelectable(true)
	a.giDesc.SetMarginStart(10)
	a.giDesc.SetMarginEnd(10)
	a.giDesc.SetMarginTop(4)
	a.giDesc.SetMarginBottom(6)
	addClass(a.giDesc, "meta")
	a.giDescSW, _ = gtk.ScrolledWindowNew(nil, nil)
	a.giDescSW.SetPolicy(gtk.POLICY_NEVER, gtk.POLICY_AUTOMATIC)
	a.giDescSW.SetSizeRequest(-1, 88)
	a.giDescSW.SetMarginStart(2)
	a.giDescSW.SetMarginEnd(2)
	a.giDescSW.Add(a.giDesc)
	v.PackStart(a.giDescSW, false, false, 0)

	// v1.12.1: aksi berlabel (bukan ikon kecil) — nyaman diklik.
	act, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 6)
	act.SetMarginStart(10)
	act.SetMarginEnd(10)
	act.SetMarginBottom(8)
	mk := func(icon, label string, fn func()) {
		b, _ := gtk.ButtonNew()
		b.SetRelief(gtk.RELIEF_NONE)
		addClass(b, "giaction")
		bb, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 6)
		bb.SetHAlign(gtk.ALIGN_CENTER)
		bb.SetMarginTop(5)
		bb.SetMarginBottom(5)
		bb.PackStart(newIcon(icon, 15), false, false, 0)
		bl, _ := gtk.LabelNew(T(label))
		bb.PackStart(bl, false, false, 0)
		b.Add(bb)
		b.SetTooltipText(T(label))
		b.Connect("clicked", func() { fn() })
		act.PackStart(b, true, true, 0)
	}
	mk("edit", "Subjek", func() { a.giSubjectDialog() })
	mk("image", "Foto", func() { a.giPhotoDialog() })
	mk("file", "Deskripsi", func() { a.giDescDialog() })
	mk("plus", "Tambah peserta", func() { a.giAddDialog() })
	v.PackStart(act, false, false, 0)

	sep, _ := gtk.SeparatorNew(gtk.ORIENTATION_HORIZONTAL)
	addClass(sep, "when")
	v.PackStart(sep, false, false, 0)

	sw, _ := gtk.ScrolledWindowNew(nil, nil)
	a.giList, _ = gtk.ListBoxNew()
	a.giList.SetSelectionMode(gtk.SELECTION_NONE)
	addClass(a.giList, "msgbg")
	sw.Add(a.giList)
	v.PackStart(sw, true, true, 0)

	gadj := sw.GetVAdjustment()
	gadj.Connect("value-changed", func() {
		if a.giOff < len(a.giAll) && gadj.GetValue()+gadj.GetPageSize() >= gadj.GetUpper()-8 {
			a.appendGiRows()
		}
	})
	return v
}

// giPlaceholder: isi kotak foto sebelum foto tiba (atau bila tak ada).
func (a *App) giPlaceholder() {
	if c, err := a.giAvBox.GetChild(); err == nil && c != nil {
		a.giAvBox.Remove(c)
	}
	av, _ := gtk.LabelNew(initials(a.giName, true))
	addClass(av, "avatar")
	addClass(av, avatarClass(a.giJID))
	av.SetHAlign(gtk.ALIGN_CENTER)
	av.SetVAlign(gtk.ALIGN_CENTER)
	a.giAvBox.Add(av)
	av.Show()
}

func (a *App) leaveGroupPage() {
	a.clearList(a.giList)
	a.giPlaceholder() // lepas pixbuf foto grup
	a.giDescSW.Hide()
	a.giAll = nil
	a.giOff = 0
	trimMem()
	if a.current != "" {
		a.openChat(a.current)
	} else {
		a.chatStack.SetVisibleChildName("empty")
	}
}

func (a *App) openGroupInfo(jid string) {
	if !isGroupJID(jid) {
		return
	}
	a.giJID = jid
	a.clearMessages() // baris chat tersembunyi dilepas selama halaman info tampil
	a.chatStack.SetVisibleChildName("groupinfo")
	a.clearList(a.giList)
	a.giAll = nil
	a.giOff = 0
	a.giAdmin = false
	c, _ := a.st.GetChat(jid)
	a.giName = a.displayName(c)
	a.giTitle.SetMarkup("<b>" + html.EscapeString(a.giName) + "</b>")
	a.giSub.SetText(T("Memuat info grup…"))
	a.giDesc.SetText("")
	a.giDescSW.Hide()
	a.giPlaceholder()
	if a.wa == nil {
		a.fillDemoGroup()
		return
	}
	go func() {
		data, err := a.wa.GroupPhoto(jid)
		glib.IdleAdd(func() bool {
			if err == nil && a.giJID == jid {
				a.setGiPhoto(data)
			}
			return false
		})
	}()
	go func() {
		info, err := a.wa.GroupInfo(jid)
		glib.IdleAdd(func() bool {
			if a.giJID != jid {
				return false
			}
			if err != nil {
				a.giSub.SetText(fmt.Sprintf(T("Gagal memuat info grup: %s"), err.Error()))
				return false
			}
			me := a.wa.MyJID()
			a.giAll = nil
			a.giAdmin = false
			for _, p := range info.Participants {
				pj := p.JID.ToNonAD().String()
				gp := giParticipant{
					jid:   pj,
					name:  a.displayName(Chat{JID: pj}),
					admin: p.IsAdmin || p.IsSuperAdmin,
					me:    pj == me,
				}
				if gp.me && gp.admin {
					a.giAdmin = true
				}
				a.giAll = append(a.giAll, gp)
			}
			if info.Name != "" {
				a.giName = info.Name
			}
			a.giTitle.SetMarkup("<b>" + html.EscapeString(a.giName) + "</b>")
			a.giSub.SetText(fmt.Sprintf(T("%d peserta"), len(a.giAll)))
			if info.Topic != "" {
				a.giDesc.SetText(info.Topic)
				a.giDescSW.Show()
			} else {
				a.giDesc.SetText(T("Tidak ada deskripsi."))
				a.giDescSW.Hide()
			}
			a.appendGiRows()
			return false
		})
	}()
}

func (a *App) fillDemoGroup() {
	a.giAdmin = true
	a.giAll = []giParticipant{
		{jid: "demo-me", name: "Kamu", admin: true, me: true},
		{jid: "6282222222222@s.whatsapp.net", name: "Budi", admin: false},
		{jid: "6281111111111@s.whatsapp.net", name: "Rina", admin: false},
		{jid: "6283333333333@s.whatsapp.net", name: "Fajar", admin: false},
		{jid: "6284444444444@s.whatsapp.net", name: "Andi Kurniawan", admin: false},
	}
	a.giSub.SetText(fmt.Sprintf(T("%d peserta"), len(a.giAll)))
	a.giDesc.SetText(T("Deskripsi contoh mode demo."))
	a.giDescSW.Show()
	a.appendGiRows()
}

func (a *App) appendGiRows() {
	end := a.giOff + giPageSize
	if end > len(a.giAll) {
		end = len(a.giAll)
	}
	for _, p := range a.giAll[a.giOff:end] {
		p := p
		row, _ := gtk.ListBoxRowNew()
		row.SetName(p.jid)
		eb, _ := gtk.EventBoxNew()
		hb, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 8)
		hb.SetMarginStart(12)
		hb.SetMarginEnd(12)
		hb.SetMarginTop(6)
		hb.SetMarginBottom(6)
		av, _ := gtk.LabelNew(initials(p.name, false))
		addClass(av, "avatar")
		addClass(av, avatarClass(p.jid))
		hb.PackStart(av, false, false, 0)
		nm := html.EscapeString(p.name)
		if p.me {
			nm += " <small>(" + html.EscapeString(T("kamu")) + ")</small>"
		}
		if p.admin {
			nm += `  <small><span foreground="#e8b93c">` + html.EscapeString(T("admin")) + `</span></small>`
		}
		nl, _ := gtk.LabelNew("")
		nl.SetMarkup(nm)
		nl.SetXAlign(0)
		hb.PackStart(nl, true, true, 0)
		eb.Add(hb)
		eb.Connect("button-press-event", func(_ *gtk.EventBox, ev *gdk.Event) bool {
			if gdk.EventButtonNewFromEvent(ev).Button() != gdk.BUTTON_SECONDARY {
				return false
			}
			if p.me {
				return false
			}
			// v1.14.0: menu anggota untuk semua orang (bukan hanya admin) —
			// "Balas secara pribadi" membuka chat 1-on-1 dengan anggota ini.
			menu, _ := gtk.MenuNew()
			menu.Append(menuItem("user", T("Balas secara pribadi"), func() { a.openPrivateChat(p.jid) }))
			if a.giAdmin {
				s, _ := gtk.SeparatorMenuItemNew()
				menu.Append(s)
				menu.Append(menuItem("ban", T("Keluarkan dari grup"), func() { a.giRemove(p.jid, p.name) }))
			}
			menu.ShowAll()
			menu.PopupAtPointer(ev)
			return true
		})
		row.Add(eb)
		a.giList.Add(row)
	}
	a.giOff = end
	a.giList.ShowAll()
}

// setGiPhoto: pasang foto grup terskala 84px; pixbuf lama lepas via GC.
func (a *App) setGiPhoto(data []byte) {
	pb := pixbufFromBytes(data)
	if pb == nil {
		return
	}
	const sz = 84
	w := pb.GetWidth()
	if w > sz {
		h := pb.GetHeight() * sz / w
		if sc, err := pb.ScaleSimple(sz, h, gdk.INTERP_BILINEAR); err == nil && sc != nil {
			pb = sc
		}
	}
	if c, err := a.giAvBox.GetChild(); err == nil && c != nil {
		a.giAvBox.Remove(c)
	}
	img, _ := gtk.ImageNewFromPixbuf(pb)
	a.giAvBox.Add(img)
	img.Show()
}

// ------------------------------------------------------------ aksi manajemen

func (a *App) giGuard() bool {
	if a.wa == nil {
		a.flashStatus(T("Tidak tersedia dalam mode demo."))
		return false
	}
	return true
}

func (a *App) giSubjectDialog() {
	if !a.giGuard() {
		return
	}
	dlg, _ := gtk.DialogNew()
	dlg.SetTransientFor(a.win)
	dlg.SetModal(true)
	dlg.SetTitle(T("Ganti subjek grup"))
	dlg.AddButton(T("Batal"), gtk.RESPONSE_CANCEL)
	ok, _ := dlg.AddButton(T("Simpan"), gtk.RESPONSE_OK)
	addClass(ok, "suggested-action")
	dlg.SetDefaultResponse(gtk.RESPONSE_OK)
	area, _ := dlg.GetContentArea()
	area.SetSpacing(8)
	area.SetMarginStart(14)
	area.SetMarginEnd(14)
	area.SetMarginTop(10)
	ent, _ := gtk.EntryNew()
	ent.SetText(a.giName)
	ent.SetActivatesDefault(true)
	area.PackStart(ent, false, false, 0)
	dlg.ShowAll()
	dlg.Connect("response", func(_ *gtk.Dialog, resp gtk.ResponseType) {
		defer dlg.Destroy()
		if resp != gtk.RESPONSE_OK {
			return
		}
		name := strings.TrimSpace(ent.GetChars(0, -1))
		if name == "" {
			return
		}
		jid := a.giJID
		go func() {
			err := a.wa.GroupSetName(jid, name)
			glib.IdleAdd(func() bool {
				if err != nil {
					a.flashStatus("⚠ " + err.Error())
				} else {
					a.st.SetChatName(jid, name)
					a.flashStatus(T("Subjek grup diperbarui"))
					a.openGroupInfo(jid)
				}
				return false
			})
		}()
	})
}

func (a *App) giDescDialog() {
	if !a.giGuard() {
		return
	}
	dlg, _ := gtk.DialogNew()
	dlg.SetTransientFor(a.win)
	dlg.SetModal(true)
	dlg.SetTitle(T("Ganti deskripsi grup"))
	dlg.AddButton(T("Batal"), gtk.RESPONSE_CANCEL)
	ok, _ := dlg.AddButton(T("Simpan"), gtk.RESPONSE_OK)
	addClass(ok, "suggested-action")
	dlg.SetDefaultResponse(gtk.RESPONSE_OK)
	area, _ := dlg.GetContentArea()
	area.SetSpacing(8)
	area.SetMarginStart(14)
	area.SetMarginEnd(14)
	area.SetMarginTop(10)
	tv, _ := gtk.TextViewNew()
	buf, _ := tv.GetBuffer()
	if t, _ := a.giDesc.GetText(); t != "" && t != T("Tidak ada deskripsi.") {
		buf.SetText(t)
	}
	sw, _ := gtk.ScrolledWindowNew(nil, nil)
	sw.SetSizeRequest(360, 140)
	sw.Add(tv)
	area.PackStart(sw, false, false, 0)
	dlg.ShowAll()
	dlg.Connect("response", func(_ *gtk.Dialog, resp gtk.ResponseType) {
		defer dlg.Destroy()
		if resp != gtk.RESPONSE_OK {
			return
		}
		s, e := buf.GetBounds()
		topic, _ := buf.GetText(s, e, true)
		jid := a.giJID
		go func() {
			err := a.wa.GroupSetDesc(jid, topic)
			glib.IdleAdd(func() bool {
				if err != nil {
					a.flashStatus("⚠ " + err.Error())
				} else {
					a.flashStatus(T("Deskripsi grup diperbarui"))
					a.openGroupInfo(jid)
				}
				return false
			})
		}()
	})
}

func (a *App) giPhotoDialog() {
	if !a.giGuard() {
		return
	}
	dlg, _ := gtk.FileChooserDialogNewWith2Buttons(T("Ganti foto grup"), a.win, gtk.FILE_CHOOSER_ACTION_OPEN,
		T("Batal"), gtk.RESPONSE_CANCEL, T("Pakai"), gtk.RESPONSE_ACCEPT)
	defer dlg.Destroy()
	dlg.SetModal(true)
	ff, _ := gtk.FileFilterNew()
	ff.SetName("PNG/JPEG")
	ff.AddPattern("*.png")
	ff.AddPattern("*.jpg")
	ff.AddPattern("*.jpeg")
	dlg.AddFilter(ff)
	if dlg.Run() != gtk.RESPONSE_ACCEPT {
		return
	}
	path := dlg.GetFilename()
	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 {
		a.flashStatus("⚠ " + err.Error())
		return
	}
	jid := a.giJID
	go func() {
		err := a.wa.GroupSetPhoto(jid, data)
		glib.IdleAdd(func() bool {
			if err != nil {
				a.flashStatus("⚠ " + err.Error())
			} else {
				a.flashStatus(T("Foto grup diperbarui"))
			}
			return false
		})
	}()
}

func (a *App) giAddDialog() {
	if !a.giGuard() {
		return
	}
	dlg, _ := gtk.DialogNew()
	dlg.SetTransientFor(a.win)
	dlg.SetModal(true)
	dlg.SetTitle(T("Tambah peserta"))
	dlg.AddButton(T("Batal"), gtk.RESPONSE_CANCEL)
	ok, _ := dlg.AddButton(T("Tambah"), gtk.RESPONSE_OK)
	addClass(ok, "suggested-action")
	dlg.SetDefaultResponse(gtk.RESPONSE_OK)
	area, _ := dlg.GetContentArea()
	area.SetSpacing(8)
	area.SetMarginStart(14)
	area.SetMarginEnd(14)
	area.SetMarginTop(10)
	lbl, _ := gtk.LabelNew(T("Nomor WhatsApp, pisahkan dengan koma atau spasi:"))
	lbl.SetXAlign(0)
	ent, _ := gtk.EntryNew()
	ent.SetPlaceholderText(T("0812 3456 7890, 62811 1111 1111"))
	ent.SetActivatesDefault(true)
	info, _ := gtk.LabelNew("")
	info.SetXAlign(0)
	addClass(info, "meta")
	area.PackStart(lbl, false, false, 0)
	area.PackStart(ent, false, false, 0)
	area.PackStart(info, false, false, 0)
	dlg.ShowAll()
	dlg.Connect("response", func(_ *gtk.Dialog, resp gtk.ResponseType) {
		if resp != gtk.RESPONSE_OK {
			dlg.Destroy()
			return
		}
		raw := ent.GetChars(0, -1)
		toks := strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == ';' || r == ' ' || r == '\n' })
		if len(toks) == 0 {
			dlg.Destroy()
			return
		}
		info.SetText(T("Memeriksa nomor…"))
		go func() {
			var jids []string
			for _, t := range toks {
				j, err := a.wa.CheckNumber(t)
				if err == nil {
					jids = append(jids, j)
				}
			}
			jid := a.giJID
			var err error
			if len(jids) > 0 {
				err = a.wa.GroupUpdateParticipants(jid, jids, true)
			}
			glib.IdleAdd(func() bool {
				dlg.Destroy()
				if len(jids) == 0 {
					a.flashStatus(T("Tidak ada nomor yang valid"))
				} else if err != nil {
					a.flashStatus("⚠ " + err.Error())
				} else {
					a.flashStatus(fmt.Sprintf(T("%d peserta ditambahkan"), len(jids)))
					a.openGroupInfo(jid)
				}
				return false
			})
		}()
	})
}

func (a *App) giRemove(pj, name string) {
	d := gtk.MessageDialogNew(a.win, gtk.DIALOG_MODAL, gtk.MESSAGE_QUESTION, gtk.BUTTONS_YES_NO,
		"%s", fmt.Sprintf(T("Keluarkan %s dari grup?"), name))
	resp := d.Run()
	d.Destroy()
	if resp != gtk.RESPONSE_YES {
		return
	}
	jid := a.giJID
	go func() {
		err := a.wa.GroupUpdateParticipants(jid, []string{pj}, false)
		glib.IdleAdd(func() bool {
			if err != nil {
				a.flashStatus("⚠ " + err.Error())
			} else {
				a.flashStatus(T("Peserta dikeluarkan"))
				a.openGroupInfo(jid)
			}
			return false
		})
	}()
}
