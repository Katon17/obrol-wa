package main

// store.go — database lokal aplikasi (daftar chat + pesan).
// Terpisah dari database sesi whatsmeow (session.db).

import (
	"database/sql"
	"encoding/json"
	"strings"
	"sort"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Chat struct {
	JID      string
	Name     string
	LastTS   int64
	LastText string
	Unread   int
	IsGroup  bool
	// MutedUntil: 0 = tidak dibisukan, -1 = selamanya, >0 = unix detik
	MutedUntil int64
	// Pinned: 0 = tidak disematkan, >0 = waktu disematkan (unix detik)
	Pinned int64
}

func (c Chat) Muted() bool {
	return c.MutedUntil == -1 || c.MutedUntil > time.Now().Unix()
}

type Msg struct {
	Chat       string
	ID         string
	Sender     string
	SenderName string
	FromMe     bool
	Mention    bool // pesan menyebut/membalas saya (tidak disimpan)
	TS         int64
	Kind       string // text, image, video, audio, ptt, document, sticker, location, contact, poll, deleted
	Text       string
	Thumb      []byte
	Raw        []byte // proto waE2E.Message (untuk unduh media)
	Status     int    // 0=pending 1=terkirim 2=diterima 3=dibaca -1=gagal
	IsRead     bool
	Edited     bool

	// v1.4: balas, teruskan, reaksi
	ReplyID     string // id pesan yang dibalas
	ReplySender string // JID pengirim pesan yang dibalas
	ReplySnip   string // cuplikan teks pesan yang dibalas
	Forwarded   bool
	Reactions   string // JSON: map JID -> emoji
}

// ReactionAgg adalah satu emoji reaksi berikut jumlahnya dan apakah saya ikut.
type ReactionAgg struct {
	Emoji string
	Count int
	Mine  bool
}

// ParseReactions mengurai kolom reactions menjadi daftar teragregasi (urut kemunculan).
func (m *Msg) ParseReactions(myJID string) []ReactionAgg {
	if m.Reactions == "" {
		return nil
	}
	var per map[string]string
	if json.Unmarshal([]byte(m.Reactions), &per) != nil {
		return nil
	}
	order := []string{}
	counts := map[string]int{}
	// urutkan map agar stabil: kumpulkan dulu kunci emoji unik sesuai urutan map (acak) → sortir per emoji
	for _, e := range per {
		if e == "" {
			continue
		}
		if _, ok := counts[e]; !ok {
			order = append(order, e)
			counts[e] = 0
		}
		counts[e]++
	}
	sort.Strings(order)
	var out []ReactionAgg
	for _, e := range order {
		out = append(out, ReactionAgg{Emoji: e, Count: counts[e]})
	}
	for jid, e := range per {
		if jid == myJID && e != "" {
			for i := range out {
				if out[i].Emoji == e {
					out[i].Mine = true
				}
			}
		}
	}
	return out
}

const (
	StPending   = 0
	StSent      = 1
	StDelivered = 2
	StRead      = 3
	StFailed    = -1
)

type Store struct {
	db *sql.DB
	mu sync.Mutex
}

func OpenStore(path string) (*Store, error) {
	db, err := sql.Open("sqlite3", "file:"+path+"?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	_, err = db.Exec(`
CREATE TABLE IF NOT EXISTS chats (
  jid TEXT PRIMARY KEY,
  name TEXT NOT NULL DEFAULT '',
  last_ts INTEGER NOT NULL DEFAULT 0,
  last_text TEXT NOT NULL DEFAULT '',
  unread INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS messages (
  chat TEXT NOT NULL,
  id TEXT NOT NULL,
  sender TEXT NOT NULL DEFAULT '',
  sender_name TEXT NOT NULL DEFAULT '',
  from_me INTEGER NOT NULL DEFAULT 0,
  ts INTEGER NOT NULL DEFAULT 0,
  kind TEXT NOT NULL DEFAULT 'text',
  text TEXT NOT NULL DEFAULT '',
  thumb BLOB,
  raw BLOB,
  status INTEGER NOT NULL DEFAULT 1,
  is_read INTEGER NOT NULL DEFAULT 1,
  PRIMARY KEY (chat, id)
);
CREATE INDEX IF NOT EXISTS idx_msg_chat_ts ON messages(chat, ts);
CREATE INDEX IF NOT EXISTS idx_msg_id ON messages(id);
`)
	if err != nil {
		return nil, err
	}
	// migrasi v1.1: kolom mute (abaikan error bila sudah ada)
	db.Exec(`ALTER TABLE chats ADD COLUMN muted_until INTEGER NOT NULL DEFAULT 0`)
	// migrasi v1.4: balas, teruskan, reaksi
	for _, q := range []string{
		`ALTER TABLE messages ADD COLUMN reply_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE messages ADD COLUMN reply_sender TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE messages ADD COLUMN reply_snip TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE messages ADD COLUMN forwarded INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE messages ADD COLUMN reactions TEXT NOT NULL DEFAULT ''`,
	} {
		db.Exec(q)
	}
	// migrasi v1.2: edit, sematkan obrolan, sematkan pesan
	db.Exec(`ALTER TABLE messages ADD COLUMN edited INTEGER NOT NULL DEFAULT 0`)
	db.Exec(`ALTER TABLE chats ADD COLUMN pinned INTEGER NOT NULL DEFAULT 0`)
	db.Exec(`CREATE TABLE IF NOT EXISTS pins (chat TEXT NOT NULL, id TEXT NOT NULL, until INTEGER NOT NULL DEFAULT 0, pinned_at INTEGER NOT NULL DEFAULT 0, PRIMARY KEY(chat, id))`)
	// bersihkan penanda lama " ✎" dari v1.0/1.1
	db.Exec(`UPDATE messages SET text = substr(text, 1, length(text)-2), edited = 1 WHERE text LIKE '% ✎'`)
	st := &Store{db: db}
	st.migrateStickers() // v1.3
	return st, nil
}

func (s *Store) Close() { s.db.Close() }

// ApplyReactionLocal memperbarui peta reaksi satu pesan (untuk optimisme UI & mode demo).
func ApplyReactionLocal(st *Store, myJID, chat, id, emoji string) {
	m, ok := st.GetMessage(chat, id)
	if !ok {
		return
	}
	per := map[string]string{}
	if m.Reactions != "" {
		json.Unmarshal([]byte(m.Reactions), &per)
	}
	if emoji == "" {
		delete(per, myJID)
	} else {
		per[myJID] = emoji
	}
	b, _ := json.Marshal(per)
	st.SetReactions(chat, id, string(b))
}

// SetReactions menyimpan peta reaksi (JID -> emoji) sebuah pesan.
func (s *Store) SetReactions(chat, id, jsonStr string) {
	s.db.Exec(`UPDATE messages SET reactions=? WHERE chat=? AND id=?`, jsonStr, chat, id)
}

func isGroupJID(j string) bool { return strings.HasSuffix(j, "@g.us") }

func (s *Store) ListChats() ([]Chat, error) {
	rows, err := s.db.Query(`SELECT jid, name, last_ts, last_text, unread, muted_until, pinned FROM chats WHERE last_ts > 0 ORDER BY pinned DESC, last_ts DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Chat
	for rows.Next() {
		var c Chat
		if err := rows.Scan(&c.JID, &c.Name, &c.LastTS, &c.LastText, &c.Unread, &c.MutedUntil, &c.Pinned); err != nil {
			return nil, err
		}
		c.IsGroup = isGroupJID(c.JID)
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) GetChat(jid string) (Chat, bool) {
	var c Chat
	err := s.db.QueryRow(`SELECT jid, name, last_ts, last_text, unread, muted_until, pinned FROM chats WHERE jid=?`, jid).
		Scan(&c.JID, &c.Name, &c.LastTS, &c.LastText, &c.Unread, &c.MutedUntil, &c.Pinned)
	c.IsGroup = isGroupJID(jid)
	return c, err == nil
}

// EnsureChat membuat baris chat bila belum ada (tanpa mengubah data lama).
func (s *Store) EnsureChat(jid, name string) {
	s.db.Exec(`INSERT INTO chats(jid, name) VALUES(?, ?) ON CONFLICT(jid) DO NOTHING`, jid, name)
}

func (s *Store) SetChatName(jid, name string) {
	if name == "" {
		return
	}
	s.db.Exec(`INSERT INTO chats(jid, name) VALUES(?, ?) ON CONFLICT(jid) DO UPDATE SET name=excluded.name`, jid, name)
}

func (s *Store) SetUnread(jid string, n int) {
	s.db.Exec(`UPDATE chats SET unread=? WHERE jid=?`, n, jid)
}

// TotalUnread menghitung pesan belum dibaca, TIDAK termasuk chat yang dibisukan
// (dan grup bila "bisukan semua grup" aktif).
func (s *Store) TotalUnread(muteGroups bool) int {
	var n int
	q := `SELECT COALESCE(SUM(unread),0) FROM chats WHERE NOT (muted_until = -1 OR muted_until > ?)`
	if muteGroups {
		q += ` AND jid NOT LIKE '%@g.us'`
	}
	s.db.QueryRow(q, time.Now().Unix()).Scan(&n)
	return n
}

func (s *Store) SetMute(jid string, until int64) {
	s.db.Exec(`INSERT INTO chats(jid, muted_until) VALUES(?, ?) ON CONFLICT(jid) DO UPDATE SET muted_until=excluded.muted_until`, jid, until)
}

// touchChat memperbarui pratinjau pesan terakhir bila pesan ini lebih baru.
func touchChat(ex execer, m *Msg, preview string, incUnread bool) {
	inc := 0
	if incUnread {
		inc = 1
	}
	ex.Exec(`INSERT INTO chats(jid, last_ts, last_text, unread) VALUES(?, ?, ?, ?)
ON CONFLICT(jid) DO UPDATE SET
  last_text = CASE WHEN excluded.last_ts >= chats.last_ts THEN excluded.last_text ELSE chats.last_text END,
  last_ts   = MAX(chats.last_ts, excluded.last_ts),
  unread    = chats.unread + ?`, m.Chat, m.TS, preview, inc, inc)
}

type execer interface {
	Exec(query string, args ...any) (sql.Result, error)
}

// SaveMessage menyimpan pesan; return true bila pesan baru (belum pernah ada).
func (s *Store) SaveMessage(m *Msg, incUnread bool) bool {
	return s.saveMsg(s.db, m, incUnread)
}

func (s *Store) saveMsg(ex execer, m *Msg, incUnread bool) bool {
	res, err := ex.Exec(`INSERT INTO messages(chat,id,sender,sender_name,from_me,ts,kind,text,thumb,raw,status,is_read,reply_id,reply_sender,reply_snip,forwarded,reactions)
VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(chat,id) DO NOTHING`,
		m.Chat, m.ID, m.Sender, m.SenderName, b2i(m.FromMe), m.TS, m.Kind, m.Text, m.Thumb, m.Raw, m.Status, b2i(m.IsRead),
		m.ReplyID, m.ReplySender, m.ReplySnip, b2i(m.Forwarded), m.Reactions)
	if err != nil {
		return false
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return false
	}
	touchChat(ex, m, previewText(m), incUnread)
	return true
}

// SaveBatch dipakai untuk sinkronisasi riwayat (banyak pesan sekaligus).
func (s *Store) SaveBatch(msgs []*Msg) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	for _, m := range msgs {
		s.saveMsg(tx, m, false)
	}
	return tx.Commit()
}

func (s *Store) LoadMessages(chat string, beforeTS int64, limit int) ([]Msg, error) {
	if beforeTS <= 0 {
		beforeTS = 1 << 62
	}
	rows, err := s.db.Query(`SELECT `+msgCols+`
FROM messages WHERE chat=? AND ts < ? ORDER BY ts DESC, rowid DESC LIMIT ?`, chat, beforeTS, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Msg
	for rows.Next() {
		m, err := scanMsg(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	// balik urutan → lama ke baru
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, rows.Err()
}

const msgCols = `chat,id,sender,sender_name,from_me,ts,kind,text,thumb,raw,status,is_read,edited,reply_id,reply_sender,reply_snip,forwarded,reactions`

type scanner interface{ Scan(dest ...any) error }

func scanMsg(r scanner) (Msg, error) {
	var m Msg
	var fm, rd, ed, fw int
	err := r.Scan(&m.Chat, &m.ID, &m.Sender, &m.SenderName, &fm, &m.TS, &m.Kind, &m.Text, &m.Thumb, &m.Raw, &m.Status, &rd, &ed,
		&m.ReplyID, &m.ReplySender, &m.ReplySnip, &fw, &m.Reactions)
	m.FromMe, m.IsRead, m.Edited, m.Forwarded = fm == 1, rd == 1, ed == 1, fw == 1
	return m, err
}

func (s *Store) GetMessage(chat, id string) (Msg, bool) {
	m, err := scanMsg(s.db.QueryRow(`SELECT `+msgCols+` FROM messages WHERE chat=? AND id=?`, chat, id))
	return m, err == nil
}

// CountSince: jumlah pesan dengan ts >= ts (untuk melompat ke pesan lama).
func (s *Store) CountSince(chat string, ts int64) int {
	var n int
	s.db.QueryRow(`SELECT COUNT(*) FROM messages WHERE chat=? AND ts >= ?`, chat, ts).Scan(&n)
	return n
}

// refreshPreview memperbarui pratinjau di daftar chat bila pesan ini yang terakhir.
func (s *Store) refreshPreview(chat, id string) {
	m, ok := s.GetMessage(chat, id)
	if !ok {
		return
	}
	s.db.Exec(`UPDATE chats SET last_text=? WHERE jid=? AND last_ts=?`, previewText(&m), chat, m.TS)
}

// ---- sematkan obrolan
func (s *Store) SetChatPinned(jid string, pinned bool) {
	var v int64
	if pinned {
		v = time.Now().Unix()
	}
	s.db.Exec(`INSERT INTO chats(jid, pinned) VALUES(?, ?) ON CONFLICT(jid) DO UPDATE SET pinned=excluded.pinned`, jid, v)
}

func (s *Store) PinnedChatCount() int {
	var n int
	s.db.QueryRow(`SELECT COUNT(*) FROM chats WHERE pinned > 0`).Scan(&n)
	return n
}

// ---- sematkan pesan
type Pin struct {
	Chat, ID string
	Until    int64
}

func (s *Store) AddPin(chat, id string, until int64) {
	s.db.Exec(`INSERT INTO pins(chat,id,until,pinned_at) VALUES(?,?,?,?) ON CONFLICT(chat,id) DO UPDATE SET until=excluded.until, pinned_at=excluded.pinned_at`,
		chat, id, until, time.Now().Unix())
}

func (s *Store) RemovePin(chat, id string) bool {
	res, err := s.db.Exec(`DELETE FROM pins WHERE chat=? AND id=?`, chat, id)
	if err != nil {
		return false
	}
	n, _ := res.RowsAffected()
	return n > 0
}

// Pins: pesan tersemat yang masih berlaku, terbaru dulu.
func (s *Store) Pins(chat string) []Pin {
	now := time.Now().Unix()
	s.db.Exec(`DELETE FROM pins WHERE until > 0 AND until < ?`, now)
	rows, err := s.db.Query(`SELECT chat,id,until FROM pins WHERE chat=? ORDER BY pinned_at DESC`, chat)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []Pin
	for rows.Next() {
		var p Pin
		rows.Scan(&p.Chat, &p.ID, &p.Until)
		out = append(out, p)
	}
	return out
}

func (s *Store) IsPinned(chat, id string) bool {
	var n int
	s.db.QueryRow(`SELECT COUNT(*) FROM pins WHERE chat=? AND id=? AND (until=0 OR until > ?)`, chat, id, time.Now().Unix()).Scan(&n)
	return n > 0
}

func (s *Store) GetRaw(chat, id string) []byte {
	var raw []byte
	s.db.QueryRow(`SELECT raw FROM messages WHERE chat=? AND id=?`, chat, id).Scan(&raw)
	return raw
}

// UnreadIncoming mengembalikan pesan masuk yang belum ditandai dibaca (id → sender).
func (s *Store) UnreadIncoming(chat string) map[string][]string {
	rows, err := s.db.Query(`SELECT id, sender FROM messages WHERE chat=? AND from_me=0 AND is_read=0`, chat)
	out := map[string][]string{}
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var id, sender string
		rows.Scan(&id, &sender)
		out[sender] = append(out[sender], id)
	}
	return out
}

func (s *Store) MarkChatRead(chat string) {
	s.db.Exec(`UPDATE messages SET is_read=1 WHERE chat=? AND is_read=0`, chat)
	s.db.Exec(`UPDATE chats SET unread=0 WHERE jid=?`, chat)
}

// UpdateStatus menaikkan status centang (tidak pernah turun).
func (s *Store) UpdateStatus(ids []string, st int) {
	for _, id := range ids {
		s.db.Exec(`UPDATE messages SET status=? WHERE id=? AND from_me=1 AND status < ? AND status >= 0`, st, id, st)
	}
}

func (s *Store) SetStatus(chat, id string, st int) {
	s.db.Exec(`UPDATE messages SET status=? WHERE chat=? AND id=?`, st, chat, id)
}

func (s *Store) EditMessage(chat, id, text string) bool {
	res, err := s.db.Exec(`UPDATE messages SET text=?, edited=1 WHERE chat=? AND id=?`, text, chat, id)
	if err != nil {
		return false
	}
	n, _ := res.RowsAffected()
	if n > 0 {
		s.refreshPreview(chat, id)
	}
	return n > 0
}

func (s *Store) RevokeMessage(chat, id string) bool {
	res, err := s.db.Exec(`UPDATE messages SET kind='deleted', text='', thumb=NULL, raw=NULL WHERE chat=? AND id=?`, chat, id)
	if err != nil {
		return false
	}
	n, _ := res.RowsAffected()
	if n > 0 {
		s.RemovePin(chat, id)
		s.refreshPreview(chat, id)
	}
	return n > 0
}

func (s *Store) ChatOfMessage(id string) string {
	var c string
	s.db.QueryRow(`SELECT chat FROM messages WHERE id=? LIMIT 1`, id).Scan(&c)
	return c
}

func (s *Store) WipeAll() {
	s.db.Exec(`DELETE FROM messages; DELETE FROM chats; DELETE FROM pins;`)
	s.db.Exec(`VACUUM`)
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

func previewText(m *Msg) string {
	t := strings.ReplaceAll(m.Text, "\n", " ")
	lbl := kindLabel(m.Kind)
	switch {
	case m.Kind == "deleted":
		t = "🚫 Pesan ini dihapus"
	case lbl != "" && t != "":
		t = lbl + " · " + t
	case lbl != "":
		t = lbl
	}
	if m.FromMe {
		t = "Anda: " + t
	} else if isGroupJID(m.Chat) && m.SenderName != "" {
		t = m.SenderName + ": " + t
	}
	if len([]rune(t)) > 120 {
		t = string([]rune(t)[:120]) + "…"
	}
	return t
}

func kindLabel(kind string) string {
	switch kind {
	case "image":
		return "📷 " + T("Foto")
	case "video":
		return "🎥 " + T("Video")
	case "audio":
		return "🎵 " + T("Audio")
	case "ptt":
		return "🎤 " + T("Pesan suara")
	case "document":
		return "📄 " + T("Dokumen")
	case "sticker":
		return "🏷️ " + T("Stiker")
	case "location":
		return "📍 " + T("Lokasi")
	case "contact":
		return "👤 " + T("Kontak")
	case "poll":
		return "📊 " + T("Polling")
	case "gif":
		return "🎞️ GIF"
	}
	return ""
}
