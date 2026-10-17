package main

// wa.go — koneksi ke WhatsApp memakai whatsmeow (protokol multi-device).

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"mime"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/appstate"
	"go.mau.fi/whatsmeow/proto/waCompanionReg"
	"go.mau.fi/whatsmeow/proto/waCommon"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/proto/waWeb"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

// UIHooks dipanggil dari goroutine WhatsApp; implementasinya wajib
// meneruskan ke thread GTK (glib.IdleAdd).
type UIHooks interface {
	OnQR(code string)
	OnQRTimeout()
	OnLoggedIn()
	OnLoggedOut()
	OnStatus(text string)
	OnNewMessage(m *Msg, live bool)
	OnMessageChanged(chat string)
	OnChatsChanged()
	OnReceipt(ids []string, st int)
	OnCall(jid string, video bool, phase, reason, callID string)
}

type WA struct {
	cli       *whatsmeow.Client
	container *sqlstore.Container
	st        *Store
	ui        UIHooks
	log       waLog.Logger

	names    sync.Map // jid string -> nama tampilan
	syncing  atomic.Bool
	syncTmr  *time.Timer
	syncMu   sync.Mutex
	loggedIn atomic.Bool
}

var bg = context.Background()

func NewWA(dataDir string, st *Store, ui UIHooks, debug bool) (*WA, error) {
	lvl := "ERROR"
	if debug {
		lvl = "DEBUG"
	}
	logger := waLog.Stdout("WA", lvl, true)
	if !debug {
		logger = waLog.Noop
	}
	store.SetOSInfo("Obrol (Linux)", [3]uint32{1, 0, 0})
	store.DeviceProps.PlatformType = waCompanionReg.DeviceProps_DESKTOP.Enum()

	dsn := "file:" + filepath.Join(dataDir, "session.db") + "?_foreign_keys=on&_journal_mode=WAL&_busy_timeout=5000"
	container, err := sqlstore.New(bg, "sqlite3", dsn, logger)
	if err != nil {
		return nil, err
	}
	w := &WA{container: container, st: st, ui: ui, log: logger}
	if err := w.newClient(); err != nil {
		return nil, err
	}
	return w, nil
}

func (w *WA) newClient() error {
	dev, err := w.container.GetFirstDevice(bg)
	if err != nil {
		return err
	}
	w.cli = whatsmeow.NewClient(dev, w.log)
	w.cli.EnableAutoReconnect = true
	w.cli.AddEventHandler(w.handle)
	return nil
}

func (w *WA) IsPaired() bool { return w.cli.Store.ID != nil }

// Start: login dengan QR bila belum dipasangkan, atau langsung terhubung.
func (w *WA) Start() {
	if !w.IsPaired() {
		w.startQR()
		return
	}
	w.ui.OnStatus("Menghubungkan…")
	go func() {
		if err := w.cli.Connect(); err != nil {
			w.ui.OnStatus("Gagal terhubung: " + err.Error())
		}
	}()
}

func (w *WA) startQR() {
	go func() {
		if w.cli.IsConnected() {
			w.cli.Disconnect()
		}
		qrChan, err := w.cli.GetQRChannel(bg)
		if err != nil {
			w.ui.OnStatus("Error QR: " + err.Error())
			return
		}
		if err := w.cli.Connect(); err != nil {
			w.ui.OnStatus("Gagal terhubung: " + err.Error())
			w.ui.OnQRTimeout()
			return
		}
		for evt := range qrChan {
			switch evt.Event {
			case "code":
				w.ui.OnQR(evt.Code)
			case "success":
				// PairSuccess ditangani di handle()
			case "timeout":
				w.ui.OnQRTimeout()
			default:
				if evt.Error != nil {
					w.ui.OnStatus("Login gagal: " + evt.Error.Error())
				}
				w.ui.OnQRTimeout()
			}
		}
	}()
}

func (w *WA) RetryQR() { w.startQR() }

// PairPhone: login memakai kode 8 karakter (alternatif QR).
func (w *WA) PairPhone(phone string) (string, error) {
	phone = digitsOnly(phone)
	if strings.HasPrefix(phone, "0") {
		phone = "62" + phone[1:]
	}
	if len(phone) < 8 {
		return "", errors.New("nomor tidak valid")
	}
	return w.cli.PairPhone(bg, phone, true, whatsmeow.PairClientChrome, "Chrome (Linux)")
}

func (w *WA) Logout() {
	if w.cli.IsLoggedIn() {
		w.cli.Logout(bg)
	} else {
		w.cli.Disconnect()
		if w.cli.Store.ID != nil {
			w.cli.Store.Delete(bg)
		}
	}
	w.afterLogout()
}

func (w *WA) afterLogout() {
	w.loggedIn.Store(false)
	w.st.WipeAll()
	w.names = sync.Map{}
	w.cli.RemoveEventHandlers()
	if err := w.newClient(); err == nil {
		w.ui.OnLoggedOut()
		w.startQR()
	}
}

func (w *WA) Close() {
	if w.cli != nil {
		w.cli.Disconnect()
	}
}

// ------------------------------------------------------------------ event

func (w *WA) handle(raw any) {
	switch evt := raw.(type) {
	case *events.PairSuccess:
		w.ui.OnStatus("Berhasil dipasangkan, menyiapkan…")
	case *events.Connected:
		w.loggedIn.Store(true)
		w.ui.OnLoggedIn()
		if !w.syncing.Load() {
			w.ui.OnStatus("Terhubung")
		}
		go w.refreshGroups()
	case *events.OfflineSyncCompleted:
		if !w.syncing.Load() {
			w.ui.OnStatus("Terhubung")
		}
	case *events.Disconnected:
		w.ui.OnStatus("Terputus — menyambung ulang…")
	case *events.KeepAliveTimeout:
		w.ui.OnStatus("Koneksi lambat…")
	case *events.StreamReplaced:
		w.ui.OnStatus("Sesi dibuka di tempat lain")
	case *events.TemporaryBan:
		w.ui.OnStatus("⚠ Diblokir sementara: " + evt.String())
	case *events.ConnectFailure:
		w.ui.OnStatus(fmt.Sprintf("Gagal terhubung (%v)", evt.Reason))
	case *events.LoggedOut:
		w.ui.OnStatus("Keluar dari perangkat")
		go w.afterLogout()
	case *events.Message:
		w.onMessage(evt, true)
	case *events.HistorySync:
		go w.onHistory(evt)
	case *events.Receipt:
		w.onReceipt(evt)
	case *events.PushName:
		w.names.Delete(evt.JID.ToNonAD().String())
		w.names.Delete(evt.JIDAlt.ToNonAD().String())
	case *events.Contact:
		w.names.Delete(evt.JID.ToNonAD().String())
		w.debouncedChats()
	case *events.GroupInfo:
		if evt.Name != nil {
			w.st.SetChatName(evt.JID.String(), evt.Name.Name)
			w.ui.OnChatsChanged()
		}
	case *events.JoinedGroup:
		w.st.SetChatName(evt.JID.String(), evt.GroupInfo.Name)
		w.ui.OnChatsChanged()
	case *events.Mute:
		var until int64
		if evt.Action.GetMuted() {
			until = -1
			if ms := evt.Action.GetMuteEndTimestamp(); ms > 0 {
				until = ms / 1000
			}
		}
		w.st.SetMute(w.normJID(evt.JID).String(), until)
		w.debouncedChats()
	case *events.Pin:
		w.st.SetChatPinned(w.normJID(evt.JID).String(), evt.Action.GetPinned())
		w.debouncedChats()
	case *events.CallOffer:
		w.ui.OnCall(evt.From.String(), false, "offer", "", evt.CallID)
	case *events.CallOfferNotice:
		w.ui.OnCall(evt.From.String(), evt.Media == "video", "offer", "", evt.CallID)
	case *events.CallTerminate:
		w.ui.OnCall(evt.From.String(), false, "terminate", evt.Reason, evt.CallID)
	case *events.MarkChatAsRead:
		if evt.Action.GetRead() {
			w.st.MarkChatRead(w.normJID(evt.JID).String())
			w.ui.OnChatsChanged()
		}
	}
}

func (w *WA) refreshGroups() {
	groups, err := w.cli.GetJoinedGroups(bg)
	if err != nil {
		return
	}
	for _, g := range groups {
		w.st.SetChatName(g.JID.String(), g.Name)
	}
	w.ui.OnChatsChanged()
}

func skipChat(j types.JID) bool {
	return j.Server == types.BroadcastServer || j.Server == types.NewsletterServer || j.IsEmpty()
}

// normJID: ubah JID LID (@lid) menjadi nomor telepon bila pemetaannya diketahui,
// supaya satu orang tidak muncul sebagai dua chat.
func (w *WA) normJID(j types.JID) types.JID {
	j = j.ToNonAD()
	if j.Server == types.HiddenUserServer {
		if pn, err := w.cli.Store.LIDs.GetPNForLID(bg, j); err == nil && !pn.IsEmpty() {
			return pn.ToNonAD()
		}
	}
	return j
}

func (w *WA) chatOf(src types.MessageSource) types.JID {
	c := src.Chat.ToNonAD()
	if c.Server == types.HiddenUserServer {
		if !src.IsFromMe && src.SenderAlt.Server == types.DefaultUserServer {
			return src.SenderAlt.ToNonAD()
		}
		if src.IsFromMe && src.RecipientAlt.Server == types.DefaultUserServer {
			return src.RecipientAlt.ToNonAD()
		}
	}
	return w.normJID(c)
}

func (w *WA) onMessage(evt *events.Message, live bool) {
	if skipChat(evt.Info.Chat) {
		return
	}
	chat := w.chatOf(evt.Info.MessageSource)
	chatS := chat.String()

	if pm := evt.Message.GetProtocolMessage(); pm != nil {
		key := pm.GetKey()
		switch pm.GetType() {
		case waE2E.ProtocolMessage_REVOKE:
			c := w.st.ChatOfMessage(key.GetID())
			if c == "" {
				c = chatS
			}
			if w.st.RevokeMessage(c, key.GetID()) {
				w.ui.OnMessageChanged(c)
			}
		case waE2E.ProtocolMessage_MESSAGE_EDIT:
			_, txt, _ := extract(pm.GetEditedMessage())
			c := w.st.ChatOfMessage(key.GetID())
			if c != "" && txt != "" && w.st.EditMessage(c, key.GetID(), txt) {
				w.ui.OnMessageChanged(c)
			}
		}
		return
	}
	if rm := evt.Message.GetReactionMessage(); rm != nil {
		w.applyReaction(chatS, evt, rm)
		return
	}
	if pin := evt.Message.GetPinInChatMessage(); pin != nil {
		if c := w.applyPin(chatS, evt); c != "" {
			w.ui.OnMessageChanged(c)
		}
		return
	}
	m := w.toMsg(evt, chat)
	if m == nil {
		return
	}
	m.Mention = !m.FromMe && w.mentionsMe(evt.Message)
	if live {
		m.IsRead = m.FromMe
		if w.st.SaveMessage(m, !m.FromMe) {
			if m.Kind == "sticker" {
				w.FetchSticker(m.Raw, m.TS, nil) // unduh di belakang layar → masuk panel "Terbaru"
			}
			w.ui.OnNewMessage(m, true)
		}
	}
}

// applyReaction memperbarui peta reaksi pesan tujuan lalu menyegarkan tampilan.
func (w *WA) applyReaction(chatS string, evt *events.Message, rm *waE2E.ReactionMessage) {
	key := rm.GetKey()
	c := w.st.ChatOfMessage(key.GetID())
	if c == "" {
		c = chatS
	}
	if _, ok := w.st.GetMessage(c, key.GetID()); !ok {
		return
	}
	sender := evt.Info.Sender.ToNonAD().String()
	if evt.Info.IsFromMe {
		sender = w.myUser()
	}
	ApplyReactionLocal(w.st, sender, c, key.GetID(), rm.GetText())
	w.ui.OnMessageChanged(c)
}

// contextOf mengambil ContextInfo dari jenis pesan apa pun (untuk balas/teruskan).
func contextOf(m *waE2E.Message) *waE2E.ContextInfo {
	switch {
	case m.GetExtendedTextMessage() != nil:
		return m.GetExtendedTextMessage().GetContextInfo()
	case m.GetImageMessage() != nil:
		return m.GetImageMessage().GetContextInfo()
	case m.GetVideoMessage() != nil:
		return m.GetVideoMessage().GetContextInfo()
	case m.GetAudioMessage() != nil:
		return m.GetAudioMessage().GetContextInfo()
	case m.GetDocumentMessage() != nil:
		return m.GetDocumentMessage().GetContextInfo()
	case m.GetStickerMessage() != nil:
		return m.GetStickerMessage().GetContextInfo()
	case m.GetContactMessage() != nil:
		return m.GetContactMessage().GetContextInfo()
	case m.GetLocationMessage() != nil:
		return m.GetLocationMessage().GetContextInfo()
	case m.GetLiveLocationMessage() != nil:
		return m.GetLiveLocationMessage().GetContextInfo()
	case m.GetGroupInviteMessage() != nil:
		return m.GetGroupInviteMessage().GetContextInfo()
	}
	return nil
}

func (w *WA) toMsg(evt *events.Message, chat types.JID) *Msg {
	kind, text, thumb := extract(evt.Message)
	if kind == "" {
		return nil
	}
	m := &Msg{
		Chat:   chat.String(),
		ID:     evt.Info.ID,
		Sender: evt.Info.Sender.ToNonAD().String(),
		FromMe: evt.Info.IsFromMe,
		TS:     evt.Info.Timestamp.Unix(),
		Kind:   kind,
		Text:   text,
		Thumb:  thumb,
		Status: StSent,
	}
	if !m.FromMe {
		m.SenderName = evt.Info.PushName
		if m.SenderName == "" || isGroupJID(m.Chat) {
			if n := w.contactName(evt.Info.Sender); n != "" {
				m.SenderName = n
			}
		}
	}
	if isMediaKind(kind) {
		m.Raw, _ = proto.Marshal(evt.Message)
	}
	if ci := contextOf(evt.Message); ci != nil {
		if sid := ci.GetStanzaID(); sid != "" {
			m.ReplyID = sid
			m.ReplySender = ci.GetParticipant()
			if m.ReplySender == "" {
				m.ReplySender = chat.String()
			}
			qk, qt, _ := extract(ci.GetQuotedMessage())
			if qt != "" {
				m.ReplySnip = qt
			} else if qk != "" && qk != "text" {
				m.ReplySnip = "kind:" + qk
			}
		}
		m.Forwarded = ci.GetIsForwarded()
	}
	return m
}

func isMediaKind(k string) bool {
	switch k {
	case "image", "video", "gif", "audio", "ptt", "document", "sticker":
		return true
	}
	return false
}

// extract mengambil jenis, teks, dan thumbnail dari sebuah pesan.
func extract(m *waE2E.Message) (kind, text string, thumb []byte) {
	if m == nil {
		return
	}
	switch {
	case m.Conversation != nil:
		return "text", m.GetConversation(), nil
	case m.ExtendedTextMessage != nil:
		return "text", m.GetExtendedTextMessage().GetText(), nil
	case m.ImageMessage != nil:
		x := m.GetImageMessage()
		return "image", x.GetCaption(), x.GetJPEGThumbnail()
	case m.VideoMessage != nil:
		x := m.GetVideoMessage()
		k := "video"
		if x.GetGifPlayback() {
			k = "gif"
		}
		return k, x.GetCaption(), x.GetJPEGThumbnail()
	case m.AudioMessage != nil:
		x := m.GetAudioMessage()
		d := fmt.Sprintf("%d:%02d", x.GetSeconds()/60, x.GetSeconds()%60)
		if x.GetPTT() {
			return "ptt", d, nil
		}
		return "audio", d, nil
	case m.DocumentMessage != nil:
		x := m.GetDocumentMessage()
		t := x.GetFileName()
		if t == "" {
			t = x.GetTitle()
		}
		if c := x.GetCaption(); c != "" && c != t {
			t += "\n" + c
		}
		return "document", t, x.GetJPEGThumbnail()
	case m.StickerMessage != nil:
		return "sticker", "", nil
	case m.LocationMessage != nil:
		x := m.GetLocationMessage()
		t := strings.TrimSpace(x.GetName() + "\n" + x.GetAddress())
		t = strings.TrimSpace(t + fmt.Sprintf("\nhttps://maps.google.com/?q=%f,%f", x.GetDegreesLatitude(), x.GetDegreesLongitude()))
		return "location", t, x.GetJPEGThumbnail()
	case m.LiveLocationMessage != nil:
		x := m.GetLiveLocationMessage()
		return "location", fmt.Sprintf("Lokasi terkini\nhttps://maps.google.com/?q=%f,%f", x.GetDegreesLatitude(), x.GetDegreesLongitude()), x.GetJPEGThumbnail()
	case m.ContactMessage != nil:
		return "contact", m.GetContactMessage().GetDisplayName(), nil
	case m.ContactsArrayMessage != nil:
		return "contact", fmt.Sprintf("%d kontak", len(m.GetContactsArrayMessage().GetContacts())), nil
	case m.PollCreationMessage != nil || m.PollCreationMessageV2 != nil || m.PollCreationMessageV3 != nil:
		p := m.GetPollCreationMessage()
		if p == nil {
			p = m.GetPollCreationMessageV2()
		}
		if p == nil {
			p = m.GetPollCreationMessageV3()
		}
		var sb strings.Builder
		sb.WriteString(p.GetName())
		for _, o := range p.GetOptions() {
			sb.WriteString("\n○ " + o.GetOptionName())
		}
		return "poll", sb.String(), nil
	case m.TemplateMessage != nil:
		if t := m.GetTemplateMessage().GetHydratedTemplate().GetHydratedContentText(); t != "" {
			return "text", t, nil
		}
	case m.ButtonsMessage != nil:
		return "text", m.GetButtonsMessage().GetContentText(), nil
	case m.ListMessage != nil:
		return "text", m.GetListMessage().GetTitle() + "\n" + m.GetListMessage().GetDescription(), nil
	case m.InteractiveMessage != nil:
		if t := m.GetInteractiveMessage().GetBody().GetText(); t != "" {
			return "text", t, nil
		}
	}
	return "", "", nil
}

// ------------------------------------------------------------- riwayat

func (w *WA) onHistory(evt *events.HistorySync) {
	w.setSyncing()
	var batch []*Msg
	for _, conv := range evt.Data.GetConversations() {
		orig, err := types.ParseJID(conv.GetID())
		if err != nil || skipChat(orig) {
			continue
		}
		chat := orig.ToNonAD()
		if chat.Server == types.HiddenUserServer {
			if pn, err := types.ParseJID(conv.GetPnJID()); err == nil && pn.Server == types.DefaultUserServer {
				chat = pn.ToNonAD()
			} else {
				chat = w.normJID(chat)
			}
		}
		if chat.Server == types.GroupServer && conv.GetName() != "" {
			w.st.SetChatName(chat.String(), conv.GetName())
		}
		for _, hm := range conv.GetMessages() {
			web := hm.GetMessage()
			if web == nil {
				continue
			}
			if web.GetMessageStubType() == waWeb.WebMessageInfo_REVOKE {
				k := web.GetKey()
				batch = append(batch, &Msg{Chat: chat.String(), ID: k.GetID(), FromMe: k.GetFromMe(),
					TS: int64(web.GetMessageTimestamp()), Kind: "deleted", Status: StRead, IsRead: true})
				continue
			}
			pe, err := w.cli.ParseWebMessage(orig, web)
			if err != nil || pe.Message == nil || pe.Message.ProtocolMessage != nil {
				continue
			}
			if pe.Message.PinInChatMessage != nil {
				defer w.applyPin(chat.String(), pe) // setelah pesan batch tersimpan
				continue
			}
			m := w.toMsg(pe, chat)
			if m == nil {
				continue
			}
			m.IsRead = true
			switch web.GetStatus() {
			case waWeb.WebMessageInfo_READ, waWeb.WebMessageInfo_PLAYED:
				m.Status = StRead
			case waWeb.WebMessageInfo_DELIVERY_ACK:
				m.Status = StDelivered
			case waWeb.WebMessageInfo_ERROR:
				m.Status = StFailed
			default:
				m.Status = StSent
			}
			batch = append(batch, m)
		}
		if n := conv.GetUnreadCount(); n > 0 {
			defer w.st.SetUnread(chat.String(), int(n))
		}
	}
	if len(batch) > 0 {
		w.st.SaveBatch(batch)
	}
	w.ui.OnChatsChanged()
}

func (w *WA) setSyncing() {
	w.syncMu.Lock()
	defer w.syncMu.Unlock()
	if !w.syncing.Load() {
		w.syncing.Store(true)
		w.ui.OnStatus("Menyinkronkan riwayat chat…")
	}
	if w.syncTmr != nil {
		w.syncTmr.Stop()
	}
	w.syncTmr = time.AfterFunc(6*time.Second, func() {
		w.syncing.Store(false)
		if w.cli.IsConnected() {
			w.ui.OnStatus("Terhubung")
		}
		w.ui.OnChatsChanged()
	})
}

var chatsTmr *time.Timer
var chatsMu sync.Mutex

func (w *WA) debouncedChats() {
	chatsMu.Lock()
	defer chatsMu.Unlock()
	if chatsTmr != nil {
		chatsTmr.Stop()
	}
	chatsTmr = time.AfterFunc(800*time.Millisecond, func() {
		w.names = sync.Map{}
		w.ui.OnChatsChanged()
	})
}

func (w *WA) onReceipt(evt *events.Receipt) {
	switch evt.Type {
	case types.ReceiptTypeReadSelf:
		// dibaca dari perangkat lain (misal ponsel)
		chat := w.chatOf(evt.MessageSource).String()
		w.st.MarkChatRead(chat)
		w.ui.OnChatsChanged()
	case types.ReceiptTypeRead, types.ReceiptTypePlayed:
		w.st.UpdateStatus(evt.MessageIDs, StRead)
		w.ui.OnReceipt(evt.MessageIDs, StRead)
	case types.ReceiptTypeDelivered:
		w.st.UpdateStatus(evt.MessageIDs, StDelivered)
		w.ui.OnReceipt(evt.MessageIDs, StDelivered)
	}
}

// ------------------------------------------------------------- nama

func (w *WA) contactName(j types.JID) string {
	j = j.ToNonAD()
	ci, err := w.cli.Store.Contacts.GetContact(bg, j)
	if err == nil && ci.Found {
		for _, n := range []string{ci.FullName, ci.FirstName, ci.BusinessName, ci.PushName} {
			if n != "" {
				return n
			}
		}
	}
	if j.Server == types.HiddenUserServer {
		if pn, err := w.cli.Store.LIDs.GetPNForLID(bg, j); err == nil && !pn.IsEmpty() {
			return w.contactName(pn)
		}
		if err == nil && ci.RedactedPhone != "" {
			return ci.RedactedPhone
		}
	}
	return ""
}

// DisplayName: nama untuk daftar chat / header.
func (w *WA) DisplayName(jid string, stored string) string {
	if isGroupJID(jid) {
		if stored != "" {
			return stored
		}
		return "Grup"
	}
	if v, ok := w.names.Load(jid); ok {
		return v.(string)
	}
	j, err := types.ParseJID(jid)
	name := ""
	if err == nil {
		name = w.contactName(j)
		if name == "" {
			if j.Server == types.DefaultUserServer {
				name = "+" + j.User
			} else {
				name = j.User
			}
		}
	}
	if name == "" {
		name = jid
	}
	if j.User == w.myUser() {
		name += " (Anda)"
	}
	w.names.Store(jid, name)
	return name
}

func (w *WA) myUser() string {
	if w.cli.Store.ID == nil {
		return ""
	}
	return w.cli.Store.ID.User
}

func (w *WA) MyName() string {
	n := w.cli.Store.PushName
	if w.cli.Store.ID != nil {
		if n != "" {
			return n + " (+" + w.cli.Store.ID.User + ")"
		}
		return "+" + w.cli.Store.ID.User
	}
	return n
}

// mentionsMe: true bila pesan @menyebut saya atau membalas pesan saya.
func (w *WA) mentionsMe(m *waE2E.Message) bool {
	var ci *waE2E.ContextInfo
	switch {
	case m.ExtendedTextMessage != nil:
		ci = m.GetExtendedTextMessage().GetContextInfo()
	case m.ImageMessage != nil:
		ci = m.GetImageMessage().GetContextInfo()
	case m.VideoMessage != nil:
		ci = m.GetVideoMessage().GetContextInfo()
	case m.AudioMessage != nil:
		ci = m.GetAudioMessage().GetContextInfo()
	case m.DocumentMessage != nil:
		ci = m.GetDocumentMessage().GetContextInfo()
	case m.StickerMessage != nil:
		ci = m.GetStickerMessage().GetContextInfo()
	}
	if ci == nil {
		return false
	}
	me := map[string]bool{}
	if id := w.cli.Store.ID; id != nil {
		me[id.User] = true
	}
	if lid := w.cli.Store.LID; !lid.IsEmpty() {
		me[lid.User] = true
	}
	if len(me) == 0 {
		return false
	}
	for _, j := range ci.GetMentionedJID() {
		if me[strings.SplitN(j, "@", 2)[0]] {
			return true
		}
	}
	if p := ci.GetParticipant(); p != "" && me[strings.SplitN(strings.SplitN(p, "@", 2)[0], ":", 2)[0]] {
		return true
	}
	return false
}

// SetMute membisukan chat (dur 0 = selamanya) atau membunyikan kembali (mute=false).
// Status disinkronkan ke WhatsApp sehingga ikut berlaku di ponsel.
func (w *WA) SetMute(chat string, mute bool, dur time.Duration) {
	var until int64
	if mute {
		until = -1
		if dur > 0 {
			until = time.Now().Add(dur).Unix()
		}
	}
	w.st.SetMute(chat, until)
	j, err := types.ParseJID(chat)
	if err != nil || !w.cli.IsLoggedIn() {
		return
	}
	go func() {
		if err := w.cli.SendAppState(bg, appstate.BuildMute(j, mute, dur)); err != nil {
			w.ui.OnStatus("Mute hanya lokal (gagal sinkron ke ponsel)")
		}
	}()
}

// applyPin memproses pesan "sematkan/lepas sematan" dari orang lain / ponsel.
func (w *WA) applyPin(chat string, evt *events.Message) string {
	pin := evt.Message.GetPinInChatMessage()
	id := pin.GetKey().GetID()
	if c := w.st.ChatOfMessage(id); c != "" {
		chat = c
	}
	switch pin.GetType() {
	case waE2E.PinInChatMessage_PIN_FOR_ALL:
		dur := int64(evt.Message.GetMessageContextInfo().GetMessageAddOnDurationInSecs())
		if dur <= 0 {
			dur = 7 * 86400
		}
		w.st.AddPin(chat, id, evt.Info.Timestamp.Unix()+dur)
	case waE2E.PinInChatMessage_UNPIN_FOR_ALL:
		w.st.RemovePin(chat, id)
	}
	return chat
}

const EditWindow = 15 * time.Minute // batas edit WhatsApp
const RevokeWindow = 60 * time.Hour // batas "hapus untuk semua orang"

// EditText mengedit pesan teks yang sudah terkirim.
func (w *WA) EditText(chat, id, text string) error {
	m, ok := w.st.GetMessage(chat, id)
	if !ok || !m.FromMe {
		return errors.New("pesan tidak ditemukan")
	}
	if time.Since(time.Unix(m.TS, 0)) > EditWindow {
		return errors.New("pesan hanya bisa diedit maksimal 15 menit setelah dikirim")
	}
	cj, err := types.ParseJID(chat)
	if err != nil {
		return err
	}
	w.st.EditMessage(chat, id, text)
	go func() {
		_, err := w.cli.SendMessage(bg, cj, w.cli.BuildEdit(cj, id, &waE2E.Message{Conversation: proto.String(text)}))
		if err != nil {
			w.ui.OnStatus("Gagal mengirim editan: " + err.Error())
		}
	}()
	return nil
}

// Revoke: hapus pesan saya untuk semua orang.
func (w *WA) Revoke(chat, id string) error {
	cj, err := types.ParseJID(chat)
	if err != nil {
		return err
	}
	w.st.RevokeMessage(chat, id)
	go func() {
		if _, err := w.cli.SendMessage(bg, cj, w.cli.BuildRevoke(cj, types.EmptyJID, id)); err != nil {
			w.ui.OnStatus("Gagal menghapus: " + err.Error())
		}
	}()
	return nil
}

// PinMessage menyematkan (dur>0) atau melepas sematan pesan untuk semua anggota chat.
func (w *WA) PinMessage(chat, id string, pin bool, dur time.Duration) error {
	m, ok := w.st.GetMessage(chat, id)
	if !ok {
		return errors.New("pesan tidak ditemukan")
	}
	cj, err := types.ParseJID(chat)
	if err != nil {
		return err
	}
	sender := types.EmptyJID
	if !m.FromMe {
		sender, _ = types.ParseJID(m.Sender)
		if sender.IsEmpty() {
			sender = cj
		}
	}
	typ := waE2E.PinInChatMessage_UNPIN_FOR_ALL
	if pin {
		typ = waE2E.PinInChatMessage_PIN_FOR_ALL
		w.st.AddPin(chat, id, time.Now().Add(dur).Unix())
	} else {
		w.st.RemovePin(chat, id)
	}
	msg := &waE2E.Message{PinInChatMessage: &waE2E.PinInChatMessage{
		Key:               w.cli.BuildMessageKey(cj, sender, id),
		Type:              typ.Enum(),
		SenderTimestampMS: proto.Int64(time.Now().UnixMilli()),
	}}
	if pin {
		msg.MessageContextInfo = &waE2E.MessageContextInfo{MessageAddOnDurationInSecs: proto.Uint32(uint32(dur.Seconds()))}
	}
	go func() {
		if _, err := w.cli.SendMessage(bg, cj, msg); err != nil {
			w.ui.OnStatus("Sematan hanya lokal: " + err.Error())
		}
	}()
	return nil
}

// PinChat menyematkan obrolan di atas daftar (sinkron ke ponsel, maks. 3).
func (w *WA) PinChat(chat string, pin bool) {
	w.st.SetChatPinned(chat, pin)
	j, err := types.ParseJID(chat)
	if err != nil || !w.cli.IsLoggedIn() {
		return
	}
	go func() {
		if err := w.cli.SendAppState(bg, appstate.BuildPin(j, pin)); err != nil {
			w.ui.OnStatus("Sematan obrolan hanya lokal (gagal sinkron)")
		}
	}()
}

// ------------------------------------------------------------- aksi

func (w *WA) MarkRead(chat string) {
	unread := w.st.UnreadIncoming(chat)
	w.st.MarkChatRead(chat)
	if len(unread) == 0 || !w.cli.IsLoggedIn() {
		return
	}
	cj, err := types.ParseJID(chat)
	if err != nil {
		return
	}
	go func() {
		for sender, ids := range unread {
			sj, _ := types.ParseJID(sender)
			if sj.IsEmpty() {
				sj = cj
			}
			for i := 0; i < len(ids); i += 50 {
				end := min(i+50, len(ids))
				w.cli.MarkRead(bg, ids[i:end], time.Now(), cj, sj)
			}
		}
	}()
}

func (w *WA) newOutgoing(chat, kind, text string) *Msg {
	return &Msg{
		Chat: chat, ID: w.cli.GenerateMessageID(), Sender: w.myUser(),
		FromMe: true, TS: time.Now().Unix(), Kind: kind, Text: text,
		Status: StPending, IsRead: true,
	}
}

func (w *WA) SendText(chat, text string) *Msg { return w.SendTextEx(chat, text, nil, false) }

// SendTextEx mengirim teks dengan opsi balasan (reply) dan penanda terusan.
func (w *WA) SendTextEx(chat, text string, reply *Msg, forwarded bool) *Msg {
	m := w.newOutgoing(chat, "text", text)
	fillReply(m, reply)
	m.Forwarded = forwarded
	w.st.SaveMessage(m, false)
	go func() {
		to, _ := types.ParseJID(chat)
		var msg *waE2E.Message
		if ci := quoteContext(reply, forwarded); ci != nil {
			msg = &waE2E.Message{ExtendedTextMessage: &waE2E.ExtendedTextMessage{Text: proto.String(text), ContextInfo: ci}}
		} else {
			msg = &waE2E.Message{Conversation: proto.String(text)}
		}
		_, err := w.cli.SendMessage(bg, to, msg, whatsmeow.SendRequestExtra{ID: m.ID})
		w.finishSend(m, err)
	}()
	return m
}

// fillReply menyalin info kutipan ke pesan baru yang akan dikirim.
func fillReply(m, reply *Msg) {
	if reply == nil {
		return
	}
	m.ReplyID, m.ReplySender = reply.ID, reply.Sender
	if t := oneLine(reply.Text, 90); t != "" {
		m.ReplySnip = t
	} else if reply.Kind != "text" {
		m.ReplySnip = "kind:" + reply.Kind
	}
}

// quoteContext menyusun ContextInfo untuk pesan keluar (balasan / terusan).
func quoteContext(reply *Msg, forwarded bool) *waE2E.ContextInfo {
	if reply == nil && !forwarded {
		return nil
	}
	ci := &waE2E.ContextInfo{}
	if forwarded {
		ci.IsForwarded = proto.Bool(true)
	}
	if reply != nil {
		ci.StanzaID = proto.String(reply.ID)
		ci.Participant = proto.String(reply.Sender)
		ci.QuotedMessage = quotedProto(reply)
	}
	return ci
}

// quotedProto menyusun proto minimal pesan yang dikutip (untuk tampilan di ponsel).
func quotedProto(m *Msg) *waE2E.Message {
	switch m.Kind {
	case "sticker":
		return &waE2E.Message{StickerMessage: &waE2E.StickerMessage{Mimetype: proto.String("image/webp")}}
	case "image":
		return &waE2E.Message{ImageMessage: &waE2E.ImageMessage{Caption: proto.String(m.Text), JPEGThumbnail: m.Thumb}}
	case "video", "gif":
		return &waE2E.Message{VideoMessage: &waE2E.VideoMessage{Caption: proto.String(m.Text)}}
	case "document":
		return &waE2E.Message{DocumentMessage: &waE2E.DocumentMessage{FileName: proto.String(oneLine(m.Text, 60))}}
	case "audio", "ptt":
		return &waE2E.Message{AudioMessage: &waE2E.AudioMessage{}}
	default:
		return &waE2E.Message{Conversation: proto.String(m.Text)}
	}
}

// SendReaction mengirim (atau menghapus, bila emoji kosong) reaksi ke sebuah pesan.
func (w *WA) SendReaction(chat, id, emoji string) error {
	m, ok := w.st.GetMessage(chat, id)
	if !ok {
		return errors.New("pesan tidak ditemukan")
	}
	to, _ := types.ParseJID(chat)
	key := &waCommon.MessageKey{RemoteJID: proto.String(chat), FromMe: proto.Bool(m.FromMe), ID: proto.String(id)}
	if isGroupJID(chat) && !m.FromMe {
		key.Participant = proto.String(m.Sender)
	}
	_, err := w.cli.SendMessage(bg, to, &waE2E.Message{ReactionMessage: &waE2E.ReactionMessage{
		Key: key, Text: proto.String(emoji), SenderTimestampMS: proto.Int64(time.Now().UnixMilli()),
	}}, whatsmeow.SendRequestExtra{ID: w.cli.GenerateMessageID()})
	return err
}

func (w *WA) finishSend(m *Msg, err error) {
	if err != nil {
		w.st.SetStatus(m.Chat, m.ID, StFailed)
		w.ui.OnStatus("Gagal mengirim: " + err.Error())
		w.ui.OnReceipt([]string{m.ID}, StFailed)
		return
	}
	w.st.UpdateStatus([]string{m.ID}, StSent)
	w.ui.OnReceipt([]string{m.ID}, StSent)
}

const maxUpload = 100 << 20

// SendFile mengirim foto (jpg/png) sebagai gambar, file lain sebagai dokumen.
func (w *WA) SendFile(chat, path, caption string) (*Msg, error) {
	return w.SendFileEx(chat, path, caption, nil, false)
}

// SendFileEx = SendFile dengan opsi balasan & penanda terusan.
func (w *WA) SendFileEx(chat, path, caption string, reply *Msg, forwarded bool) (*Msg, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(data) > maxUpload {
		return nil, errors.New("ukuran file maksimal 100 MB")
	}
	mt := mime.TypeByExtension(strings.ToLower(filepath.Ext(path)))
	if mt == "" {
		mt = http.DetectContentType(data)
	}
	isImg := mt == "image/jpeg" || mt == "image/png"
	var thumb []byte
	var wd, ht uint32
	if isImg {
		thumb, wd, ht = makeThumb(data)
	}
	kind := "document"
	if isImg {
		kind = "image"
	}
	text := caption
	if !isImg {
		text = filepath.Base(path)
		if caption != "" {
			text += "\n" + caption
		}
	}
	m := w.newOutgoing(chat, kind, text)
	m.Thumb = thumb
	if isImg { // pratinjau lokal yang lebih tajam (thumbnail protokol tetap kecil)
		if p, _, _ := makeThumbSize(data, 520, 82); p != nil {
			m.Thumb = p
		}
	}
	fillReply(m, reply)
	m.Forwarded = forwarded
	w.st.SaveMessage(m, false)

	go func() {
		to, _ := types.ParseJID(chat)
		var msg *waE2E.Message
		if isImg {
			up, err := w.cli.Upload(bg, data, whatsmeow.MediaImage)
			if err != nil {
				w.finishSend(m, err)
				return
			}
			msg = &waE2E.Message{ImageMessage: &waE2E.ImageMessage{
				URL: proto.String(up.URL), DirectPath: proto.String(up.DirectPath), MediaKey: up.MediaKey,
				Mimetype: proto.String(mt), FileEncSHA256: up.FileEncSHA256, FileSHA256: up.FileSHA256,
				FileLength: proto.Uint64(uint64(len(data))), JPEGThumbnail: thumb,
				Width: proto.Uint32(wd), Height: proto.Uint32(ht),
			}}
			if caption != "" {
				msg.ImageMessage.Caption = proto.String(caption)
			}
		} else {
			up, err := w.cli.Upload(bg, data, whatsmeow.MediaDocument)
			if err != nil {
				w.finishSend(m, err)
				return
			}
			if mt == "" {
				mt = "application/octet-stream"
			}
			msg = &waE2E.Message{DocumentMessage: &waE2E.DocumentMessage{
				URL: proto.String(up.URL), DirectPath: proto.String(up.DirectPath), MediaKey: up.MediaKey,
				Mimetype: proto.String(mt), FileEncSHA256: up.FileEncSHA256, FileSHA256: up.FileSHA256,
				FileLength: proto.Uint64(uint64(len(data))), FileName: proto.String(filepath.Base(path)),
				Title: proto.String(filepath.Base(path)),
			}}
			if caption != "" {
				msg.DocumentMessage.Caption = proto.String(caption)
			}
		}
		if ci := quoteContext(reply, forwarded); ci != nil {
			if msg.ImageMessage != nil {
				msg.ImageMessage.ContextInfo = ci
			} else if msg.DocumentMessage != nil {
				msg.DocumentMessage.ContextInfo = ci
			}
		}
		raw, _ := proto.Marshal(msg)
		w.st.db.Exec(`UPDATE messages SET raw=? WHERE chat=? AND id=?`, raw, m.Chat, m.ID)
		_, err := w.cli.SendMessage(bg, to, msg, whatsmeow.SendRequestExtra{ID: m.ID})
		w.finishSend(m, err)
	}()
	return m, nil
}

// SendAudio mengirim berkas audio sebagai pesan suara (ptt) atau audio biasa.
func (w *WA) SendAudio(chat, path string, ptt bool) (*Msg, error) { return w.SendAudioEx(chat, path, ptt, nil) }

// SendAudioEx = SendAudio dengan opsi balasan.
func (w *WA) SendAudioEx(chat, path string, ptt bool, reply *Msg) (*Msg, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(data) > maxUpload {
		return nil, errors.New("ukuran file maksimal 100 MB")
	}
	mt := mime.TypeByExtension(strings.ToLower(filepath.Ext(path)))
	if mt == "" {
		mt = http.DetectContentType(data)
	}
	if mt == "" || !strings.HasPrefix(mt, "audio/") {
		mt = "audio/ogg"
	}
	kind := "audio"
	if ptt {
		kind = "ptt"
	}
	m := w.newOutgoing(chat, kind, "")
	fillReply(m, reply)
	w.st.SaveMessage(m, false)
	go func() {
		to, _ := types.ParseJID(chat)
		up, err := w.cli.Upload(bg, data, whatsmeow.MediaAudio)
		if err != nil {
			w.finishSend(m, err)
			return
		}
		msg := &waE2E.Message{AudioMessage: &waE2E.AudioMessage{
			URL: proto.String(up.URL), DirectPath: proto.String(up.DirectPath), MediaKey: up.MediaKey,
			Mimetype: proto.String(mt), FileEncSHA256: up.FileEncSHA256, FileSHA256: up.FileSHA256,
			FileLength: proto.Uint64(uint64(len(data))), PTT: proto.Bool(ptt),
			Seconds: proto.Uint32(audioSeconds(data)),
		}}
		if ci := quoteContext(reply, false); ci != nil {
			msg.AudioMessage.ContextInfo = ci
		}
		raw, _ := proto.Marshal(msg)
		w.st.db.Exec(`UPDATE messages SET raw=? WHERE chat=? AND id=?`, raw, m.Chat, m.ID)
		_, err = w.cli.SendMessage(bg, to, msg, whatsmeow.SendRequestExtra{ID: m.ID})
		w.finishSend(m, err)
	}()
	return m, nil
}

// ForwardMessage mengirim ulang isi pesan src ke chat dst (media diunduh lalu diunggah ulang).
func (w *WA) ForwardMessage(dst string, src Msg) (*Msg, error) {
	switch src.Kind {
	case "text":
		return w.SendTextEx(dst, src.Text, nil, true), nil
	case "sticker":
		sha := ""
		if ref, ok := stickerRef(src.Raw); ok && stickerExists(ref.SHA) {
			sha = ref.SHA
		} else {
			data, err := w.downloadMsg(src)
			if err != nil {
				return nil, err
			}
			if sha, err = saveStickerFile(data); err != nil {
				return nil, err
			}
			w.st.AddSticker(sha, false, time.Now().Unix())
		}
		return w.SendStickerEx(dst, sha, true)
	case "image", "video", "gif", "audio", "ptt", "document":
		data, err := w.downloadMsg(src)
		if err != nil {
			return nil, err
		}
		return w.sendMediaData(dst, src, data)
	}
	return nil, errors.New("jenis pesan ini tidak bisa diteruskan")
}

// downloadMsg mengunduh media pesan dari server WhatsApp berdasarkan raw proto-nya.
func (w *WA) downloadMsg(src Msg) ([]byte, error) {
	var msg waE2E.Message
	if proto.Unmarshal(src.Raw, &msg) != nil {
		return nil, errors.New("media tidak tersedia")
	}
	var dl whatsmeow.DownloadableMessage
	switch src.Kind {
	case "image":
		dl = msg.GetImageMessage()
	case "gif", "video":
		dl = msg.GetVideoMessage()
	case "audio", "ptt":
		dl = msg.GetAudioMessage()
	case "document":
		dl = msg.GetDocumentMessage()
	case "sticker":
		dl = msg.GetStickerMessage()
	}
	if dl == nil {
		return nil, errors.New("media tidak tersedia")
	}
	return w.cli.Download(bg, dl)
}

// sendMediaData mengunggah ulang data media sebagai pesan terusan dengan jenis yang sama.
func (w *WA) sendMediaData(dst string, src Msg, data []byte) (*Msg, error) {
	var orig waE2E.Message
	proto.Unmarshal(src.Raw, &orig)
	m := w.newOutgoing(dst, src.Kind, src.Text)
	m.Thumb = src.Thumb
	m.Forwarded = true
	w.st.SaveMessage(m, false)
	go func() {
		to, _ := types.ParseJID(dst)
		media := whatsmeow.MediaDocument
		switch src.Kind {
		case "image":
			media = whatsmeow.MediaImage
		case "gif", "video":
			media = whatsmeow.MediaVideo
		case "audio", "ptt":
			media = whatsmeow.MediaAudio
		}
		up, err := w.cli.Upload(bg, data, media)
		if err != nil {
			w.finishSend(m, err)
			return
		}
		ci := &waE2E.ContextInfo{IsForwarded: proto.Bool(true)}
		fl := proto.Uint64(uint64(len(data)))
		enc := func() (string, []byte, []byte) { return up.URL, up.FileEncSHA256, up.FileSHA256 }
		url, encSHA, sha := enc()
		var out *waE2E.Message
		switch src.Kind {
		case "image":
			mt := orig.GetImageMessage().GetMimetype()
			if mt == "" {
				mt = "image/jpeg"
			}
			out = &waE2E.Message{ImageMessage: &waE2E.ImageMessage{
				URL: proto.String(url), DirectPath: proto.String(up.DirectPath), MediaKey: up.MediaKey,
				Mimetype: proto.String(mt), FileEncSHA256: encSHA, FileSHA256: sha, FileLength: fl,
				JPEGThumbnail: src.Thumb, Caption: proto.String(src.Text), ContextInfo: ci}}
		case "gif", "video":
			mt := orig.GetVideoMessage().GetMimetype()
			if mt == "" {
				mt = "video/mp4"
			}
			out = &waE2E.Message{VideoMessage: &waE2E.VideoMessage{
				URL: proto.String(url), DirectPath: proto.String(up.DirectPath), MediaKey: up.MediaKey,
				Mimetype: proto.String(mt), FileEncSHA256: encSHA, FileSHA256: sha, FileLength: fl,
				JPEGThumbnail: src.Thumb, Caption: proto.String(src.Text),
				GifPlayback: proto.Bool(src.Kind == "gif"), ContextInfo: ci}}
		case "audio", "ptt":
			mt := orig.GetAudioMessage().GetMimetype()
			if mt == "" {
				mt = "audio/ogg; codecs=opus"
			}
			out = &waE2E.Message{AudioMessage: &waE2E.AudioMessage{
				URL: proto.String(url), DirectPath: proto.String(up.DirectPath), MediaKey: up.MediaKey,
				Mimetype: proto.String(mt), FileEncSHA256: encSHA, FileSHA256: sha, FileLength: fl,
				PTT: proto.Bool(src.Kind == "ptt"), ContextInfo: ci}}
		default:
			mt := orig.GetDocumentMessage().GetMimetype()
			if mt == "" {
				mt = "application/octet-stream"
			}
			fn := orig.GetDocumentMessage().GetFileName()
			if fn == "" {
				fn = oneLine(src.Text, 60)
			}
			out = &waE2E.Message{DocumentMessage: &waE2E.DocumentMessage{
				URL: proto.String(url), DirectPath: proto.String(up.DirectPath), MediaKey: up.MediaKey,
				Mimetype: proto.String(mt), FileEncSHA256: encSHA, FileSHA256: sha, FileLength: fl,
				FileName: proto.String(fn), Title: proto.String(fn), ContextInfo: ci}}
		}
		raw, _ := proto.Marshal(out)
		w.st.db.Exec(`UPDATE messages SET raw=? WHERE chat=? AND id=?`, raw, m.Chat, m.ID)
		_, err = w.cli.SendMessage(bg, to, out, whatsmeow.SendRequestExtra{ID: m.ID})
		w.finishSend(m, err)
	}()
	return m, nil
}

// makeThumb: thumbnail JPEG kecil (maks 96px) untuk dikirim di pesan.
func makeThumb(data []byte) ([]byte, uint32, uint32) { return makeThumbSize(data, 96, 70) }

// makeThumbSize: perkecil gambar (rata-rata area, tanpa dependensi tambahan) jadi JPEG.
func makeThumbSize(data []byte, maxS, quality int) ([]byte, uint32, uint32) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, 0, 0
	}
	b := img.Bounds()
	W, H := b.Dx(), b.Dy()
	if W < 1 || H < 1 {
		return nil, 0, 0
	}
	tw, th := W, H
	if W >= H && W > maxS {
		tw, th = maxS, H*maxS/W
	} else if H > W && H > maxS {
		tw, th = W*maxS/H, maxS
	}
	tw, th = max(tw, 1), max(th, 1)
	dst := image.NewRGBA(image.Rect(0, 0, tw, th))
	for y := 0; y < th; y++ {
		y0, y1 := y*H/th, max((y+1)*H/th, y*H/th+1)
		for x := 0; x < tw; x++ {
			x0, x1 := x*W/tw, max((x+1)*W/tw, x*W/tw+1)
			// sampel maks 4x4 per piksel tujuan supaya tetap cepat
			sx, sy := max((x1-x0)/4, 1), max((y1-y0)/4, 1)
			var r, g, bl, n uint32
			for yy := y0; yy < y1; yy += sy {
				for xx := x0; xx < x1; xx += sx {
					cr, cg, cb, _ := img.At(b.Min.X+xx, b.Min.Y+yy).RGBA()
					r, g, bl, n = r+cr, g+cg, bl+cb, n+1
				}
			}
			o := dst.PixOffset(x, y)
			dst.Pix[o], dst.Pix[o+1], dst.Pix[o+2], dst.Pix[o+3] = uint8(r/n>>8), uint8(g/n>>8), uint8(bl/n>>8), 255
		}
	}
	var buf bytes.Buffer
	jpeg.Encode(&buf, dst, &jpeg.Options{Quality: quality})
	return buf.Bytes(), uint32(W), uint32(H)
}

// OpenMedia mengunduh media (bila belum) lalu membukanya dengan aplikasi default.
func (w *WA) OpenMediaPath(chat, id string) (string, error) {
	raw := w.st.GetRaw(chat, id)
	if raw == nil {
		return "", errors.New("media tidak tersedia")
	}
	var m waE2E.Message
	if err := proto.Unmarshal(raw, &m); err != nil {
		return "", err
	}
	var dl whatsmeow.DownloadableMessage
	var mt, name string
	switch {
	case m.ImageMessage != nil:
		dl, mt = m.ImageMessage, m.ImageMessage.GetMimetype()
	case m.VideoMessage != nil:
		dl, mt = m.VideoMessage, m.VideoMessage.GetMimetype()
	case m.AudioMessage != nil:
		dl, mt = m.AudioMessage, m.AudioMessage.GetMimetype()
	case m.DocumentMessage != nil:
		dl, mt, name = m.DocumentMessage, m.DocumentMessage.GetMimetype(), m.DocumentMessage.GetFileName()
	case m.StickerMessage != nil:
		dl, mt = m.StickerMessage, m.StickerMessage.GetMimetype()
	default:
		return "", errors.New("jenis media tidak didukung")
	}
	dir := downloadDir()
	os.MkdirAll(dir, 0o755)
	if name == "" {
		ext := ".bin"
		base := strings.SplitN(mt, ";", 2)[0]
		if exts, _ := mime.ExtensionsByType(base); len(exts) > 0 {
			ext = exts[len(exts)-1]
		}
		switch base {
		case "image/jpeg":
			ext = ".jpg"
		case "audio/ogg":
			ext = ".ogg"
		case "video/mp4":
			ext = ".mp4"
		}
		name = id + ext
	}
	name = strings.ReplaceAll(filepath.Base(name), "/", "_")
	path := filepath.Join(dir, name)
	if _, err := os.Stat(path); err != nil {
		data, err := w.cli.Download(bg, dl)
		if err != nil {
			return "", fmt.Errorf("unduh gagal: %w", err)
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return "", err
		}
	}
	return path, nil
}

func (w *WA) OpenMedia(chat, id string) error {
	p, err := w.OpenMediaPath(chat, id)
	if err != nil {
		return err
	}
	return exec.Command("xdg-open", p).Start()
}

func downloadDir() string {
	out, err := exec.Command("xdg-user-dir", "DOWNLOAD").Output()
	d := strings.TrimSpace(string(out))
	if err != nil || d == "" {
		home, _ := os.UserHomeDir()
		d = filepath.Join(home, "Downloads")
	}
	return filepath.Join(d, "Obrol")
}

// CheckNumber memastikan nomor terdaftar di WhatsApp, lalu kembalikan JID-nya.
func (w *WA) CheckNumber(phone string) (string, error) {
	phone = digitsOnly(phone)
	if strings.HasPrefix(phone, "0") {
		phone = "62" + phone[1:]
	}
	if len(phone) < 8 {
		return "", errors.New("nomor tidak valid")
	}
	res, err := w.cli.IsOnWhatsApp(bg, []string{"+" + phone})
	if err != nil {
		return "", err
	}
	if len(res) == 0 || !res[0].IsIn {
		return "", errors.New("nomor ini tidak terdaftar di WhatsApp")
	}
	return res[0].JID.ToNonAD().String(), nil
}

func digitsOnly(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
