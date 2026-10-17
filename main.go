package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

const version = "1.5.0"

//go:embed assets/obrol.png
var iconPNG []byte

//go:embed assets/obrol-unread.png
var iconUnreadPNG []byte

//go:embed assets/demo/*.webp
var demoStickersFS embed.FS

type Options struct {
	Hidden    bool
	Demo      bool
	DemoLogin bool
	Debug     bool
}

type Config struct {
	Notify      bool   `json:"notify"`
	Sound       bool   `json:"sound"`
	CloseToTray bool   `json:"close_to_tray"`
	MuteGroups  bool   `json:"mute_groups"`
	Theme       string `json:"theme"` // auto | light | dark
	Lang        string `json:"lang"`  // id | en
	path        string
}

func loadConfig(dir string) *Config {
	c := &Config{Notify: true, Sound: true, CloseToTray: true, Theme: "auto", path: filepath.Join(dir, "config.json")}
	if b, err := os.ReadFile(c.path); err == nil {
		json.Unmarshal(b, c)
	}
	return c
}

func (c *Config) Save() {
	b, _ := json.MarshalIndent(c, "", "  ")
	os.WriteFile(c.path, b, 0o600)
}

func dataDir() string {
	d := os.Getenv("XDG_DATA_HOME")
	if d == "" {
		home, _ := os.UserHomeDir()
		d = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(d, "obrol")
}

func main() {
	var o Options
	flag.BoolVar(&o.Hidden, "hidden", false, "mulai tersembunyi di system tray")
	flag.BoolVar(&o.Demo, "demo", false, "mode demo dengan data contoh (tanpa koneksi)")
	flag.BoolVar(&o.DemoLogin, "demo-login", false, "tampilkan halaman login contoh")
	flag.BoolVar(&o.Debug, "debug", false, "log whatsmeow ke terminal")
	showVer := flag.Bool("version", false, "tampilkan versi")
	exportIcon := flag.String("export-icon", "", "tulis ikon PNG bawaan ke path ini lalu keluar")
	flag.Parse()
	if *showVer {
		fmt.Println("Obrol", version, "— dibuat oleh", author)
		return
	}
	if *exportIcon != "" {
		if err := os.WriteFile(*exportIcon, iconPNG, 0o644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	dir := dataDir()
	appID := "id.obrol.Obrol"
	if o.Demo || o.DemoLogin {
		dir = filepath.Join(os.TempDir(), "obrol-demo")
		os.RemoveAll(dir)
		appID = "id.obrol.ObrolDemo"
	}
	appDir = dir
	if err := os.MkdirAll(dir, 0o700); err != nil {
		fmt.Fprintln(os.Stderr, "tidak bisa membuat folder data:", err)
		os.Exit(1)
	}
	st, err := OpenStore(filepath.Join(dir, "app.db"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "database error:", err)
		os.Exit(1)
	}
	defer st.Close()

	gapp, err := gtk.ApplicationNew(appID, glib.APPLICATION_FLAGS_NONE)
	if err != nil {
		fmt.Fprintln(os.Stderr, "GTK error:", err)
		os.Exit(1)
	}
	a := &App{gapp: gapp, st: st, cfg: loadConfig(dir), opts: o}
	a.pendingCalls = map[string]pendingCall{}
	if a.cfg.Lang == "" {
		a.cfg.Lang = "id"
	}
	lang = a.cfg.Lang
	if v := os.Getenv("OBROL_LANG"); v == "en" || v == "id" {
		lang = v
	}
	a.iconPath = filepath.Join(dir, "obrol.png")
	os.WriteFile(a.iconPath, iconPNG, 0o644)

	if o.Demo {
		seedDemo(st)
	} else if !o.DemoLogin {
		a.wa, err = NewWA(dir, st, a, o.Debug)
		if err != nil {
			fmt.Fprintln(os.Stderr, "whatsmeow error:", err)
			os.Exit(1)
		}
	}
	gapp.Connect("activate", a.Activate)
	// argumen milik kita tidak diteruskan ke GApplication
	os.Exit(gapp.Run([]string{os.Args[0]}))
}

// ------------------------------------------------------------ demo

func seedDemo(st *Store) {
	now := time.Now()
	ts := func(d time.Duration) int64 { return now.Add(-d).Unix() }
	sh := seedDemoStickers(st, now)
	writeDemoVN()
	type dm struct {
		chat, name string
		msgs       []Msg
		unread     int
	}
	grp := "120363000000000001@g.us"
	data := []dm{
		{"6281234567890@s.whatsapp.net", "Ibu", []Msg{
			{Text: "Le, sudah makan belum?", TS: ts(26 * time.Hour)},
			{Text: "Sudah bu, tadi makan soto di Simpang Lima 😄", FromMe: true, TS: ts(26*time.Hour - 5*time.Minute), Status: StRead},
			{Kind: "sticker", FromMe: true, Raw: localStickerRaw(sh["siap"], false), TS: ts(26*time.Hour - 6*time.Minute), Status: StRead},
			{Kind: "ptt", Raw: localAudioRaw(), TS: ts(25 * time.Hour)},
			{Kind: "call", Text: "missed-voice", TS: ts(25*time.Hour - 2*time.Minute)},
			{Text: "Nanti sore jadi pulang ke rumah?", TS: ts(40 * time.Minute)},
			{Text: "Jangan lupa bawa oleh-oleh lumpia ya 🥟", TS: ts(38 * time.Minute)},
		}, 2},
		{grp, "Tim Proyek Semarang", []Msg{
			{Sender: "6281111111111@s.whatsapp.net", SenderName: "Rina", Text: "Teman-teman, meeting besok jam 09.00 di kantor ya", TS: ts(3 * time.Hour)},
			{Sender: "6282222222222@s.whatsapp.net", SenderName: "Budi", Text: "Siap! Materi presentasi sudah saya upload: https://example.com/slide", TS: ts(2*time.Hour + 50*time.Minute)},
			{Sender: "6282222222222@s.whatsapp.net", SenderName: "Budi", Kind: "sticker", Raw: localStickerRaw(sh["halo"], true), TS: ts(2*time.Hour + 45*time.Minute)},
			{Kind: "sticker", FromMe: true, Raw: localStickerRaw(sh["otw"], false), TS: ts(2*time.Hour + 32*time.Minute), Status: StDelivered},
			{Text: "Oke, saya bawa laptop & proyektor 👍", FromMe: true, TS: ts(2*time.Hour + 30*time.Minute), Status: StDelivered},
			{Sender: "6281111111111@s.whatsapp.net", SenderName: "Rina", Kind: "document", Text: "Laporan_Q3_2026.pdf", TS: ts(20 * time.Minute)},
		}, 1},
		{"6289876543210@s.whatsapp.net", "Andi Kurniawan", []Msg{
			{Text: "Bro, link Obrol yang kamu bikin mana?", TS: ts(5 * time.Hour)},
			{Text: "Ini aplikasi WhatsApp native buat Linux Mint, super ringan. Cuma satu binary!", FromMe: true, TS: ts(4*time.Hour + 55*time.Minute), Status: StRead},
			{Kind: "ptt", Text: "0:14", TS: ts(4*time.Hour + 50*time.Minute)},
			{Text: "Mantap 🔥", TS: ts(4*time.Hour + 49*time.Minute)},
			{Kind: "sticker", FromMe: true, Raw: localStickerRaw(sh["mantap"], false), TS: ts(4*time.Hour + 48*time.Minute), Status: StRead},
		}, 0},
		{"120363000000000002@g.us", "Alumni SMA 2016 🎓", []Msg{
			{Sender: "6283333333333@s.whatsapp.net", SenderName: "Fajar", Text: "Selamat pagi semua 🙏", TS: ts(70 * time.Minute)},
			{Sender: "6284444444444@s.whatsapp.net", SenderName: "Sinta", Text: "Pagi! Reuni jadi bulan depan?", TS: ts(65 * time.Minute)},
			{Sender: "6283333333333@s.whatsapp.net", SenderName: "Fajar", Text: "Jadi dong, nanti aku share link form-nya", TS: ts(60 * time.Minute)},
		}, 3},
		{"6285555555555@s.whatsapp.net", "Kurir Paket", []Msg{
			{Text: "Paket sudah sampai di depan rumah ya kak", TS: ts(50 * time.Hour)},
			{Text: "Terima kasih pak!", FromMe: true, TS: ts(49 * time.Hour), Status: StRead},
		}, 0},
		{"6287777777777@s.whatsapp.net", "Dewi", []Msg{
			{Kind: "location", Text: "Lawang Sewu\nJl. Pemuda, Sekayu, Semarang\nhttps://maps.google.com/?q=-6.983900,110.410400", TS: ts(8 * 24 * time.Hour)},
			{Text: "Ketemu di sini jam 4 ya", TS: ts(8*24*time.Hour - time.Minute)},
		}, 0},
	}
	// v1.4: contoh balasan, reaksi, dan pesan terusan
	ibu := "6281234567890@s.whatsapp.net"
	budi := "6282222222222@s.whatsapp.net"
	rina := "6281111111111@s.whatsapp.net"
	data[0].msgs[2].ReplyID, data[0].msgs[2].ReplySender, data[0].msgs[2].ReplySnip = ibu+"-0", ibu, "Le, sudah makan belum?"
	data[0].msgs[4].Reactions = `{"demo-me":"❤️"}`
	data[1].msgs[4].ReplyID, data[1].msgs[4].ReplySender = grp+"-1", budi
	data[1].msgs[4].ReplySnip = "Siap! Materi presentasi sudah saya upload: https://example.com/slide"
	data[1].msgs[2].Reactions = `{"` + rina + `":"❤️","demo-me":"👍"}`
	data[1].msgs[3].Reactions = `{"` + rina + `":"👍"}`
	data[2].msgs[4].Forwarded = true

	for _, c := range data {
		if c.chat == grp {
			st.SetChatName(c.chat, c.name)
		} else {
			st.SetChatName(c.chat, c.name)
		}
		for i, m := range c.msgs {
			m.Chat = c.chat
			m.ID = fmt.Sprintf("%s-%d", c.chat, i)
			if m.Kind == "" {
				m.Kind = "text"
			}
			if m.Status == 0 {
				m.Status = StSent
			}
			m.IsRead = m.FromMe || i < len(c.msgs)-c.unread
			if !m.FromMe && !isGroupJID(c.chat) {
				m.Sender = c.chat
			}
			st.SaveMessage(&m, !m.IsRead)
		}
	}
	st.SetMute("120363000000000002@g.us", -1)
	st.AddPin(grp, grp+"-0", now.Add(7*24*time.Hour).Unix())
	st.SetChatPinned("6281234567890@s.whatsapp.net", true)
}

// seedDemoStickers mengisi pustaka stiker demo dari aset tertanam.
func seedDemoStickers(st *Store, now time.Time) map[string]string {
	list := []struct {
		file    string
		anim    bool
		fav     bool
		usedAgo time.Duration
	}{
		{"siap.webp", false, true, 26 * time.Hour},
		{"halo.webp", true, true, 2 * time.Hour},
		{"mantap.webp", false, true, 4 * time.Hour},
		{"otw.webp", false, false, 2 * time.Hour},
		{"makasih.webp", false, false, 30 * time.Hour},
		{"wkwk.webp", false, false, 50 * time.Hour},
	}
	out := map[string]string{}
	for _, it := range list {
		data, err := demoStickersFS.ReadFile("assets/demo/" + it.file)
		if err != nil {
			continue
		}
		sha, err := saveStickerFile(data)
		if err != nil {
			continue
		}
		st.AddSticker(sha, it.anim, now.Add(-it.usedAgo-24*time.Hour).Unix())
		if it.fav {
			st.SetStickerFav(sha, true)
		}
		st.db.Exec(`UPDATE stickers SET used_at=? WHERE sha=?`, now.Add(-it.usedAgo).Unix(), sha)
		out[strings.TrimSuffix(it.file, ".webp")] = sha
	}
	return out
}
