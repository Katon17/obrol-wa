package main

// v1.9: presence (online/last seen/typing) ala WhatsApp.

import (
	"fmt"
	"strings"
	"time"

	"github.com/gotk3/gotk3/glib"
	"go.mau.fi/whatsmeow/types"
)

// OnPresence: kontak muncul/hilang (online / last seen).
func (a *App) OnPresence(jid string, available bool, lastSeen time.Time) {
	glib.IdleAdd(func() {
		a.online[jid] = available
		if !available && !lastSeen.IsZero() {
			a.lastSeen[jid] = lastSeen.Unix()
			a.st.SetLastSeen(jid, lastSeen.Unix()) // v1.14.2 (issue #9)
		}
		// v1.14.2 (issue #10): lawan offline → indikator typing-nya basi.
		if !available {
			for ch, sj := range a.typingFrom {
				if sj == jid || ch == jid {
					a.clearTyping(ch, "")
				}
			}
		}
		if a.current == jid {
			if c, ok := a.st.GetChat(jid); ok {
				a.refreshSub(c, jid)
			}
		}
	})
}

// OnTyping: seseorang sedang mengetik di chat (kontak atau grup).
func (a *App) OnTyping(chat, senderJID string, composing bool) {
	glib.IdleAdd(func() {
		if composing {
			name := ""
			if senderJID != "" && senderJID != chat {
				name = a.personName(chat, senderJID)
			}
			a.typing[chat] = name
			a.typingFrom[chat] = senderJID
			// v1.14.2 (issue #10): jaring pengaman — bila `paused` tak pernah
			// tiba (aplikasi lawan ditutup paksa), indikator lepas sendiri.
			if t := a.typingTmr[chat]; t != nil {
				t.Stop()
			}
			a.typingTmr[chat] = time.AfterFunc(30*time.Second, func() {
				glib.IdleAdd(func() { a.clearTyping(chat, "") })
			})
			if a.current == chat {
				if c, ok := a.st.GetChat(chat); ok {
					a.refreshSub(c, chat)
				}
			}
		} else {
			a.clearTyping(chat, "")
		}
	})
}

// clearTyping (v1.14.2, issue #10): buang indikator typing chat; dari sisi C
// dipanggil lewat paused, timer pengaman, presence-unavailable, dan pesan masuk.
func (a *App) clearTyping(chat, fromJID string) {
	if fromJID != "" && a.typingFrom[chat] != fromJID {
		return
	}
	delete(a.typing, chat)
	delete(a.typingFrom, chat)
	if t := a.typingTmr[chat]; t != nil {
		t.Stop()
		delete(a.typingTmr, chat)
	}
	if a.current == chat {
		if c, ok := a.st.GetChat(chat); ok {
			a.refreshSub(c, chat)
		}
	}
}

func (a *App) personName(chat, jid string) string {
	if a.wa != nil {
		if n := a.wa.DisplayName(jid, ""); n != "" {
			return n
		}
	}
	if n := a.chatName(jid); n != "" {
		return n
	}
	// anggota grup tanpa chat tersendiri: ambil dari riwayat pesan grup
	if msgs, err := a.st.LoadMessages(chat, 0, 300); err == nil {
		for _, m := range msgs {
			if m.Sender == jid && m.SenderName != "" {
				return m.SenderName
			}
		}
	}
	return formatPhone(strings.SplitN(jid, "@", 2)[0])
}

// refreshSub: isi subtitle header (typing > online > last seen > default).
func (a *App) refreshSub(c Chat, jid string) {
	if name, ok := a.typing[jid]; ok {
		if c.IsGroup && name != "" {
			a.chatSub.SetText(fmt.Sprintf(T("%s sedang mengetik…"), name))
		} else {
			a.chatSub.SetText(T("sedang mengetik…"))
		}
		return
	}
	if c.IsGroup {
		a.chatSub.SetText(T("Grup"))
		return
	}
	if a.online[jid] {
		a.chatSub.SetText(T("online"))
		return
	}
	if ls := a.lastSeen[jid]; ls > 0 {
		t := time.Unix(ls, 0)
		a.chatSub.SetText(fmt.Sprintf(T("terakhir dilihat %s, %s"), dayLabelShort(t), t.Format("15:04")))
		return
	}
	user := strings.SplitN(jid, "@", 2)[0]
	if a.wa != nil {
		user = a.wa.pnUser(jid) // v1.13.1: petakan LID→PN agar subtitle bukan digit LID
	}
	a.chatSub.SetText(formatPhone(user))
}

// pingTyping: beri tahu kontak bahwa kita sedang mengetik (throttle 2 dtk,
// otomatis "paused" setelah 4 dtk tanpa input).
func (a *App) pingTyping() {
	if a.wa == nil || a.current == "" {
		return
	}
	now := time.Now()
	if now.Sub(a.lastTypingPing) < 2*time.Second {
		return
	}
	a.lastTypingPing = now
	a.wa.sendTyping(a.current, true)
	if a.typingPause != nil {
		a.typingPause.Stop()
	}
	chat := a.current
	a.typingPause = time.AfterFunc(4*time.Second, func() {
		if a.wa != nil && a.current == chat {
			a.wa.sendTyping(chat, false)
		}
	})
}

// demoTyping: simulasi indikator mengetik untuk mode demo.
func (a *App) demoTyping() {
	grp := "120363000000000001@g.us"
	time.AfterFunc(9*time.Second, func() {
		a.OnTyping(grp, "6282222222222@s.whatsapp.net", true)
		time.AfterFunc(8*time.Second, func() {
			a.OnTyping(grp, "", false)
			a.OnPresence("6281234567890@s.whatsapp.net", true, time.Time{})
		})
	})
}

// SubscribePresence & kirim status typing kita (dipanggil dari wa.go/UI).
func (w *WA) subscribePresence(jid string) {
	j, err := types.ParseJID(jid)
	if err != nil {
		return
	}
	go w.cli.SubscribePresence(bg, j)
}

func (w *WA) sendTyping(jid string, composing bool) {
	j, err := types.ParseJID(jid)
	if err != nil {
		return
	}
	st := types.ChatPresencePaused
	if composing {
		st = types.ChatPresenceComposing
	}
	go w.cli.SendChatPresence(bg, j, st, "")
}
