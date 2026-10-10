package main

// stickers_core.go — penyimpanan stiker lokal, unduh stiker masuk, kirim stiker.
// File stiker disimpan di ~/.local/share/obrol/stickers/<sha256>.webp.

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

const stickerSize = 512

// appDir diisi main() dengan folder data aktif (normal atau demo).
var appDir string

func stickerDir() string { return filepath.Join(appDir, "stickers") }

func stickerPath(sha string) string { return filepath.Join(stickerDir(), sha+".webp") }

func stickerExists(sha string) bool {
	st, err := os.Stat(stickerPath(sha))
	return err == nil && st.Size() > 0
}

// saveStickerFile menyimpan data WebP; mengembalikan sha256-nya (hex).
func saveStickerFile(data []byte) (string, error) {
	sum := sha256.Sum256(data)
	sha := hex.EncodeToString(sum[:])
	if stickerExists(sha) {
		return sha, nil
	}
	os.MkdirAll(stickerDir(), 0o755)
	tmp := stickerPath(sha) + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return "", err
	}
	return sha, os.Rename(tmp, stickerPath(sha))
}

// StickerRef: info stiker yang diambil dari raw proto sebuah pesan.
type StickerRef struct {
	SHA      string
	Animated bool
	Lottie   bool
	PNG      []byte // thumbnail PNG (dipakai untuk stiker Lottie)
	msg      *waE2E.StickerMessage
}

func stickerRef(raw []byte) (StickerRef, bool) {
	if len(raw) == 0 {
		return StickerRef{}, false
	}
	var m waE2E.Message
	if proto.Unmarshal(raw, &m) != nil || m.StickerMessage == nil {
		return StickerRef{}, false
	}
	s := m.StickerMessage
	if len(s.GetFileSHA256()) == 0 {
		return StickerRef{}, false
	}
	return StickerRef{
		SHA: hex.EncodeToString(s.GetFileSHA256()), Animated: s.GetIsAnimated(), Lottie: s.GetIsLottie(),
		PNG: s.GetPngThumbnail(), msg: s,
	}, true
}

// ------------------------------------------------------------ store

type StickerInfo struct {
	SHA      string
	Animated bool
	Fav      bool
}

func (s *Store) migrateStickers() {
	s.db.Exec(`CREATE TABLE IF NOT EXISTS stickers (
  sha TEXT PRIMARY KEY,
  animated INTEGER NOT NULL DEFAULT 0,
  fav INTEGER NOT NULL DEFAULT 0,
  used_at INTEGER NOT NULL DEFAULT 0,
  added_at INTEGER NOT NULL DEFAULT 0,
  hidden INTEGER NOT NULL DEFAULT 0)`)
}

// AddSticker mencatat stiker (diterima/dibuat). Tidak memunculkan lagi stiker yang disembunyikan.
func (s *Store) AddSticker(sha string, animated bool, ts int64) {
	s.db.Exec(`INSERT INTO stickers(sha, animated, added_at) VALUES(?,?,?)
ON CONFLICT(sha) DO UPDATE SET added_at = max(added_at, excluded.added_at)`, sha, b2i(animated), ts)
}

func (s *Store) TouchSticker(sha string) {
	s.db.Exec(`UPDATE stickers SET used_at=?, hidden=0 WHERE sha=?`, time.Now().Unix(), sha)
}

func (s *Store) SetStickerFav(sha string, fav bool) {
	s.db.Exec(`INSERT INTO stickers(sha, added_at) VALUES(?,?) ON CONFLICT(sha) DO NOTHING`, sha, time.Now().Unix())
	s.db.Exec(`UPDATE stickers SET fav=?, hidden=0 WHERE sha=?`, b2i(fav), sha)
}

func (s *Store) HideSticker(sha string) {
	s.db.Exec(`UPDATE stickers SET hidden=1, fav=0 WHERE sha=?`, sha)
}

func (s *Store) IsStickerFav(sha string) bool {
	var f int
	s.db.QueryRow(`SELECT fav FROM stickers WHERE sha=?`, sha).Scan(&f)
	return f == 1
}

// ListStickers: favorit saja (fav=true) atau semua yang terbaru dipakai/diterima.
func (s *Store) ListStickers(fav bool, limit int) []StickerInfo {
	q := `SELECT sha, animated, fav FROM stickers WHERE hidden=0 ORDER BY max(used_at, added_at) DESC LIMIT ?`
	if fav {
		q = `SELECT sha, animated, fav FROM stickers WHERE hidden=0 AND fav=1 ORDER BY max(used_at, added_at) DESC LIMIT ?`
	}
	rows, err := s.db.Query(q, limit)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []StickerInfo
	for rows.Next() {
		var si StickerInfo
		var an, fv int
		if rows.Scan(&si.SHA, &an, &fv) == nil && stickerExists(si.SHA) {
			si.Animated, si.Fav = an == 1, fv == 1
			out = append(out, si)
		}
	}
	return out
}

// ------------------------------------------------------------ WA

var (
	stickerMu       sync.Mutex
	stickerInflight = map[string][]func(error){}
)

// FetchSticker mengunduh file stiker dari pesan (sekali saja walau diminta berkali-kali).
// cb dipanggil dari goroutine lain.
func (w *WA) FetchSticker(raw []byte, ts int64, cb func(error)) {
	ref, ok := stickerRef(raw)
	if !ok || ref.Lottie {
		if cb != nil {
			cb(errors.New("stiker tidak didukung"))
		}
		return
	}
	if stickerExists(ref.SHA) {
		w.st.AddSticker(ref.SHA, ref.Animated, ts)
		if cb != nil {
			cb(nil)
		}
		return
	}
	stickerMu.Lock()
	cbs, running := stickerInflight[ref.SHA]
	stickerInflight[ref.SHA] = append(cbs, cb)
	stickerMu.Unlock()
	if running {
		return
	}
	go func() {
		data, err := w.cli.Download(bg, ref.msg)
		if err == nil {
			if _, err2 := webpChunks(data); err2 != nil {
				err = err2
			} else {
				os.MkdirAll(stickerDir(), 0o755)
				tmp := stickerPath(ref.SHA) + ".tmp"
				if err = os.WriteFile(tmp, data, 0o644); err == nil {
					err = os.Rename(tmp, stickerPath(ref.SHA))
				}
			}
		}
		if err == nil {
			w.st.AddSticker(ref.SHA, ref.Animated, ts)
		}
		stickerMu.Lock()
		list := stickerInflight[ref.SHA]
		delete(stickerInflight, ref.SHA)
		stickerMu.Unlock()
		for _, f := range list {
			if f != nil {
				f(err)
			}
		}
	}()
}

// localStickerRaw: proto minimal agar stiker kiriman sendiri bisa ditampilkan sebelum terunggah.
func localStickerRaw(sha string, animated bool) []byte {
	sum, _ := hex.DecodeString(sha)
	raw, _ := proto.Marshal(&waE2E.Message{StickerMessage: &waE2E.StickerMessage{
		FileSHA256: sum, Mimetype: proto.String("image/webp"), IsAnimated: proto.Bool(animated),
		Width: proto.Uint32(stickerSize), Height: proto.Uint32(stickerSize),
	}})
	return raw
}

// SendSticker mengunggah ulang file stiker lokal lalu mengirimnya.
func (w *WA) SendSticker(chat, sha string) (*Msg, error) {
	data, err := os.ReadFile(stickerPath(sha))
	if err != nil {
		return nil, errors.New("file stiker tidak ditemukan")
	}
	wd, ht, animated := webpInfo(data)
	if wd == 0 {
		wd, ht = stickerSize, stickerSize
	}
	m := w.newOutgoing(chat, "sticker", "")
	m.Raw = localStickerRaw(sha, animated)
	w.st.SaveMessage(m, false)
	w.st.TouchSticker(sha)

	go func() {
		to, _ := types.ParseJID(chat)
		up, err := w.cli.Upload(bg, data, whatsmeow.MediaImage)
		if err != nil {
			w.finishSend(m, err)
			return
		}
		msg := &waE2E.Message{StickerMessage: &waE2E.StickerMessage{
			URL: proto.String(up.URL), DirectPath: proto.String(up.DirectPath), MediaKey: up.MediaKey,
			Mimetype: proto.String("image/webp"), FileEncSHA256: up.FileEncSHA256, FileSHA256: up.FileSHA256,
			FileLength: proto.Uint64(uint64(len(data))), Width: proto.Uint32(uint32(wd)), Height: proto.Uint32(uint32(ht)),
			IsAnimated: proto.Bool(animated), MediaKeyTimestamp: proto.Int64(time.Now().Unix()),
			StickerSentTS: proto.Int64(time.Now().UnixMilli()),
		}}
		raw, _ := proto.Marshal(msg)
		w.st.db.Exec(`UPDATE messages SET raw=? WHERE chat=? AND id=?`, raw, m.Chat, m.ID)
		_, err = w.cli.SendMessage(bg, to, msg, whatsmeow.SendRequestExtra{ID: m.ID})
		w.finishSend(m, err)
	}()
	return m, nil
}
