package main

// v1.7: dialog profil kontak/grup (foto, nomor, about, info grup).

import (
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	whatsmeow "go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

func profileDir() string {
	d := filepath.Join(dataDir(), "profiles")
	os.MkdirAll(d, 0o755)
	return d
}

func profilePicPath(jid string) string {
	safe := strings.NewReplacer("/", "_", ":", "_", "@", "_").Replace(jid)
	return filepath.Join(profileDir(), safe+".jpg")
}

// fetchProfilePic mengunduh foto profil (preview) lalu menyimpannya ke cache.
func (w *WA) fetchProfilePic(jid string) ([]byte, error) {
	j, err := types.ParseJID(jid)
	if err != nil {
		return nil, err
	}
	pi, err := w.cli.GetProfilePictureInfo(bg, j, &whatsmeow.GetProfilePictureParams{Preview: true})
	if err != nil || pi == nil {
		return nil, fmt.Errorf("foto tidak tersedia")
	}
	resp, err := http.Get(pi.URL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil || len(data) == 0 {
		return nil, fmt.Errorf("foto gagal diunduh")
	}
	os.WriteFile(profilePicPath(jid), data, 0o644)
	return data, nil
}

// fetchAbout = teks "about/status" milik kontak (bukan grup).
func (w *WA) fetchAbout(jid string) string {
	if isGroupJID(jid) {
		return ""
	}
	j, err := types.ParseJID(jid)
	if err != nil {
		return ""
	}
	info, err := w.cli.GetUserInfo(bg, []types.JID{j})
	if err != nil {
		return ""
	}
	return info[j].Status
}

// fetchGroup = deskripsi (topik) grup beserta jumlah pesertanya.
func (w *WA) fetchGroup(jid string) (topic string, members int) {
	j, err := types.ParseJID(jid)
	if err != nil {
		return "", 0
	}
	gi, err := w.cli.GetGroupInfo(bg, j)
	if err != nil || gi == nil {
		return "", 0
	}
	n := gi.ParticipantCount
	if n == 0 {
		n = len(gi.Participants)
	}
	return gi.Topic, n
}

func prettyPhone(jid string) string {
	u := strings.SplitN(jid, "@", 2)[0]
	if !strings.HasPrefix(u, "+") {
		u = "+" + u
	}
	return u
}

// demoProfile menyediakan isi profil tiruan untuk mode demo.
func demoProfile(jid string) (about, topic string, members int) {
	switch jid {
	case "6281234567890@s.whatsapp.net":
		return "Suka masak & lumpia Semarang 🥟", "", 0
	case "6282222222222@s.whatsapp.net":
		return "Trainer kantor & fotografer lepas 📷", "", 0
	case "6281111111111@s.whatsapp.net":
		return "Admin proyek · jangan lupa backup!", "", 0
	case "6283333333333@s.whatsapp.net":
		return "Panitia reuni 2026 🎓", "", 0
	case "6284444444444@s.whatsapp.net":
		return "Hey there! I am using WhatsApp.", "", 0
	case "6285555555555@s.whatsapp.net":
		return "Kurir wilayah Semarang barat 📦", "", 0
	case "6287777777777@s.whatsapp.net":
		return "Coffee hunter ☕", "", 0
	case "120363000000000001@g.us":
		return "", "Koordinasi proyek kantor Semarang", 4
	case "120363000000000002@g.us":
		return "", "Grup reuni angkatan 2016", 6
	}
	return "Hey there! I am using WhatsApp.", "", 0
}

// showProfile membuka dialog profil untuk satu chat (kontak atau grup).
func (a *App) showProfile(jid string) {
	if jid == "" {
		return
	}
	c, _ := a.st.GetChat(jid)
	name := a.displayName(c)
	isGrp := isGroupJID(jid)

	dlg, _ := gtk.DialogNew()
	dlg.SetTransientFor(a.win)
	dlg.SetModal(true)
	dlg.SetTitle(T("Profil"))
	dlg.SetDefaultSize(380, 460)
	dlg.SetResizable(false)
	area, _ := dlg.GetContentArea()
	area.SetSpacing(8)
	area.SetMarginTop(14)
	area.SetMarginBottom(10)
	area.SetMarginStart(16)
	area.SetMarginEnd(16)

	// avatar / foto
	holder, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	holder.SetHAlign(gtk.ALIGN_CENTER)
	initLbl, _ := gtk.LabelNew(initials(name, isGrp))
	addClass(initLbl, "avatar", "big", avatarClass(jid))
	holder.PackStart(initLbl, false, false, 0)
	area.PackStart(holder, false, false, 0)

	nameLbl, _ := gtk.LabelNew("")
	nameLbl.SetMarkup("<span size='x-large' weight='bold'>" + html.EscapeString(name) + "</span>")
	nameLbl.SetXAlign(0.5)
	nameLbl.SetLineWrap(true)
	area.PackStart(nameLbl, false, false, 4)

	subLbl, _ := gtk.LabelNew("")
	addClass(subLbl, "meta")
	subLbl.SetXAlign(0.5)
	if isGrp {
		subLbl.SetText(T("Grup"))
	} else {
		subLbl.SetText(prettyPhone(jid))
	}
	area.PackStart(subLbl, false, false, 0)

	aboutLbl, _ := gtk.LabelNew("")
	aboutLbl.SetXAlign(0.5)
	aboutLbl.SetLineWrap(true)
	aboutLbl.SetJustify(gtk.JUSTIFY_CENTER)
	aboutLbl.SetMarginTop(8)
	area.PackStart(aboutLbl, false, false, 0)

	topicBox, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 4)
	topicBox.SetMarginTop(6)
	topicHead, _ := gtk.LabelNew(T("Deskripsi grup"))
	addClass(topicHead, "meta")
	topicHead.SetXAlign(0)
	topicLbl, _ := gtk.LabelNew("")
	topicLbl.SetLineWrap(true)
	topicLbl.SetXAlign(0)
	topicBox.PackStart(topicHead, false, false, 0)
	topicBox.PackStart(topicLbl, false, false, 0)
	topicBox.SetNoShowAll(true)
	area.PackStart(topicBox, false, false, 0)

	btns, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 8)
	btns.SetHAlign(gtk.ALIGN_CENTER)
	btns.SetMarginTop(10)
	if !isGrp {
		cp, _ := gtk.ButtonNewWithLabel(T("Salin nomor"))
		cp.Connect("clicked", func() {
			if cb, err := gtk.ClipboardGet(gdk.SELECTION_CLIPBOARD); err == nil {
				cb.SetText(prettyPhone(jid))
				a.flashStatus(T("Nomor disalin"))
			}
		})
		btns.PackStart(cp, false, false, 0)
	}
	closeBtn, _ := gtk.ButtonNewWithLabel(T("Tutup"))
	closeBtn.Connect("clicked", func() { dlg.Response(gtk.RESPONSE_CLOSE) })
	btns.PackStart(closeBtn, false, false, 0)
	area.PackStart(btns, false, false, 0)
	area.ShowAll()

	var kill func() = func() { initLbl.Destroy() }
	setPic := func(data []byte) {
		pb := pixbufFromBytes(data)
		if pb == nil {
			return
		}
		pb, _ = pb.ScaleSimple(96, 96, gdk.INTERP_BILINEAR)
		img, _ := gtk.ImageNewFromPixbuf(pb)
		if kill != nil {
			kill()
			kill = nil
		}
		holder.PackStart(img, false, false, 0)
		img.Show()
	}
	if data, err := os.ReadFile(profilePicPath(jid)); err == nil && len(data) > 0 {
		setPic(data)
	}

	go func() {
		var about, topic string
		var members int
		var pic []byte
		if a.wa == nil {
			about, topic, members = demoProfile(jid)
		} else {
			about = a.wa.fetchAbout(jid)
			if isGrp {
				topic, members = a.wa.fetchGroup(jid)
			}
			pic, _ = a.wa.fetchProfilePic(jid)
		}
		glib.IdleAdd(func() {
			if isGrp {
				if members > 0 {
					subLbl.SetText(fmt.Sprintf(T("%d peserta"), members))
				}
				if topic != "" {
					topicLbl.SetText(topic)
					topicBox.SetNoShowAll(false)
					topicBox.ShowAll()
				}
			} else if about != "" {
				aboutLbl.SetText("“" + about + "”")
			}
			if len(pic) > 0 {
				setPic(pic)
			}
		})
	}()

	dlg.Run()
	dlg.Destroy()
}
