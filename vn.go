package main

// v1.5: pesan suara (rekam/kirim/putar) serta notifikasi panggilan masuk.

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"google.golang.org/protobuf/proto"

	waE2E "go.mau.fi/whatsmeow/proto/waE2E"
)

type pendingCall struct {
	jid   string
	video bool
	ts    int64
}

var (
	recCmd  *exec.Cmd
	recPath string
)

func removeClass(w interface {
	GetStyleContext() (*gtk.StyleContext, error)
}, classes ...string) {
	sc, err := w.GetStyleContext()
	if err != nil {
		return
	}
	for _, c := range classes {
		sc.RemoveClass(c)
	}
}

func haveCmd(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// audioSeconds: perkiraan durasi (detik); WAV PCM dihitung tepat, lainnya lewat ukuran.
func audioSeconds(data []byte) uint32 {
	if len(data) > 44 && string(data[0:4]) == "RIFF" {
		byteRate := 96000.0
		if len(data) > 32 {
			if br := binary.LittleEndian.Uint32(data[28:32]); br > 0 {
				byteRate = float64(br)
			}
		}
		return uint32(float64(len(data)-44) / byteRate)
	}
	return uint32(len(data)/4000) + 1
}

func localAudioRaw() []byte {
	raw, _ := proto.Marshal(&waE2E.Message{AudioMessage: &waE2E.AudioMessage{
		Mimetype: proto.String("audio/ogg"), PTT: proto.Bool(true), Seconds: proto.Uint32(2),
	}})
	return raw
}

func demoVNPath() string { return "/tmp/obrol-demo-vn.wav" }

// writeDemoVN membuat WAV sinus 2 detik untuk pratinjau pemutar di mode demo.
func writeDemoVN() {
	p := demoVNPath()
	if fi, err := os.Stat(p); err == nil && fi.Size() > 1000 {
		return
	}
	const sr = 8000
	var body bytes.Buffer
	for i := 0; i < sr*2; i++ {
		f := 440.0
		if i >= sr {
			f = 587.3
		}
		binary.Write(&body, binary.LittleEndian, int16(math.Sin(2*math.Pi*f*float64(i)/sr)*9000))
	}
	var h bytes.Buffer
	h.WriteString("RIFF")
	binary.Write(&h, binary.LittleEndian, uint32(36+body.Len()))
	h.WriteString("WAVEfmt ")
	binary.Write(&h, binary.LittleEndian, uint32(16))
	binary.Write(&h, binary.LittleEndian, uint16(1))
	binary.Write(&h, binary.LittleEndian, uint16(1))
	binary.Write(&h, binary.LittleEndian, uint32(sr))
	binary.Write(&h, binary.LittleEndian, uint32(sr*2))
	binary.Write(&h, binary.LittleEndian, uint16(2))
	binary.Write(&h, binary.LittleEndian, uint16(16))
	h.WriteString("data")
	binary.Write(&h, binary.LittleEndian, uint32(body.Len()))
	os.WriteFile(p, append(h.Bytes(), body.Bytes()...), 0o644)
}

// ---------------------------------------------------------------- rekam

func (a *App) toggleRec() {
	if recCmd != nil {
		a.stopRec(true)
		return
	}
	a.startRec()
}

func (a *App) startRec() {
	if a.current == "" {
		return
	}
	base := fmt.Sprintf(os.TempDir()+string(os.PathSeparator)+"obrol-vn-%d", time.Now().UnixNano())
	var cmd *exec.Cmd
	switch {
	case haveCmd("ffmpeg"):
		recPath = base + ".ogg"
		cmd = exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-f", "pulse", "-i", "default",
			"-ac", "1", "-ar", "48000", "-c:a", "libopus", "-b:a", "32k", recPath)
	case haveCmd("parecord"):
		recPath = base + ".ogg"
		cmd = exec.Command("parecord", "--channels=1", "--rate=48000", "--file-format=ogg", recPath)
	case haveCmd("arecord"):
		recPath = base + ".wav"
		cmd = exec.Command("arecord", "-q", "-f", "S16_LE", "-c", "1", "-r", "48000", recPath)
	default:
		a.flashStatus("⚠ " + T("Perekam suara tidak ditemukan (pasang ffmpeg/pulseaudio-utils)"))
		return
	}
	if err := cmd.Start(); err != nil {
		a.flashStatus("⚠ " + err.Error())
		return
	}
	recCmd = cmd
	if a.micBtn != nil {
		addClass(a.micBtn, "rec")
	}
	a.flashStatus(T("Merekam… klik lagi untuk mengirim"))
}

func (a *App) stopRec(send bool) {
	cmd, path := recCmd, recPath
	recCmd, recPath = nil, ""
	if a.micBtn != nil {
		removeClass(a.micBtn, "rec")
	}
	if cmd == nil {
		return
	}
	cmd.Process.Signal(os.Interrupt)
	done := make(chan struct{})
	go func() { cmd.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		cmd.Process.Kill()
		<-done
	}
	if !send {
		os.Remove(path)
		return
	}
	fi, err := os.Stat(path)
	if err != nil || fi.Size() < 600 {
		os.Remove(path)
		a.flashStatus("⚠ " + T("Rekaman kosong atau gagal"))
		return
	}
	ptt := strings.HasSuffix(path, ".ogg")
	chat, reply := a.current, a.replyTo
	a.cancelReply()
	var m *Msg
	if a.wa != nil {
		m, err = a.wa.SendAudioEx(chat, path, ptt, reply)
	} else {
		kind := "audio"
		if ptt {
			kind = "ptt"
		}
		m = &Msg{Chat: chat, ID: fmt.Sprint(time.Now().UnixNano()), FromMe: true, TS: time.Now().Unix(),
			Kind: kind, Status: StRead, IsRead: true, Raw: localAudioRaw()}
		fillReply(m, reply)
		a.st.SaveMessage(m, false)
	}
	os.Remove(path)
	if m != nil {
		a.stickBottom = true
		a.appendMessage(m)
		a.reloadChats()
		a.flashStatus(T("Pesan suara terkirim"))
	} else if err != nil {
		a.flashStatus("⚠ " + err.Error())
	}
}

// ---------------------------------------------------------------- putar

func (a *App) playMsg(chat, id string) {
	m, ok := a.st.GetMessage(chat, id)
	if !ok || (m.Kind != "audio" && m.Kind != "ptt") {
		a.openMedia(chat, id)
		return
	}
	if a.wa == nil {
		a.flashStatus(T("Memutar pesan suara…"))
		a.playAudio(demoVNPath())
		return
	}
	a.flashStatus(T("Memutar pesan suara…"))
	go func() {
		path, err := a.wa.OpenMediaPath(chat, id)
		glib.IdleAdd(func() {
			if err != nil {
				a.flashStatus("⚠ " + err.Error())
				return
			}
			a.playAudio(path)
		})
	}()
}

func (a *App) playAudio(path string) {
	players := [][]string{
		{"mpv", "--no-video", "--really-quiet"},
		{"ffplay", "-nodisp", "-autoexit", "-loglevel", "quiet"},
		{"paplay"},
		{"vlc", "--intf", "dummy", "--play-and-exit"},
	}
	for _, c := range players {
		if haveCmd(c[0]) {
			args := append(append([]string{}, c[1:]...), path)
			if err := exec.Command(c[0], args...).Start(); err == nil {
				return
			}
		}
	}
	a.flashStatus("⚠ " + T("Pemutar audio tidak ditemukan"))
}

// ---------------------------------------------------------------- panggilan

func callLabel(code string) string {
	switch code {
	case "missed-voice":
		return T("Panggilan suara tak terjawab")
	case "missed-video":
		return T("Panggilan video tak terjawab")
	case "done-voice":
		return T("Panggilan suara · diangkat di ponsel")
	case "done-video":
		return T("Panggilan video · diangkat di ponsel")
	}
	return T("Panggilan")
}

// OnCall dipanggil whatsmeow saat ada peristiwa panggilan (offer/terminate).
func (a *App) OnCall(jid string, video bool, phase, reason, callID string) {
	glib.IdleAdd(func() {
		now := time.Now().Unix()
		txt := fmt.Sprintf(T("Panggilan %s masuk dari %s — angkat di ponsel"),
			map[bool]string{true: T("video"), false: T("suara")}[video], a.chatName(jid))
		if phase == "offer" {
			a.pendingCalls[callID] = pendingCall{jid: jid, video: video, ts: now}
			a.flashStatus("📞 " + txt)
			a.callNotify(txt, jid)
			return
		}
		pc, ok := a.pendingCalls[callID]
		delete(a.pendingCalls, callID)
		missed := !ok || reason == "timeout" || reason == "reject" || reason == "busy" || reason == "lost"
		if !ok {
			pc = pendingCall{jid: jid, video: video, ts: now}
		}
		code := "done-voice"
		if missed {
			code = "missed-voice"
		}
		if pc.video {
			code = strings.Replace(code, "voice", "video", 1)
		}
		m := &Msg{Chat: pc.jid, ID: "call-" + callID, Sender: pc.jid, TS: now, Kind: "call", Text: code, IsRead: true, Status: StRead}
		a.st.SaveMessage(m, false)
		if a.current == pc.jid {
			a.appendMessage(m)
		}
		a.reloadChats()
		if missed {
			a.callNotify(callLabel(code), pc.jid)
		}
	})
}

func (a *App) chatName(jid string) string {
	c, _ := a.st.GetChat(jid)
	return a.displayName(c)
}

func (a *App) callNotify(body, chat string) {
	if !a.cfg.Notify {
		return
	}
	c, _ := a.st.GetChat(chat)
	n := glib.NotificationNew("📞 " + a.displayName(c))
	n.SetBody(body)
	if a.iconPath != "" {
		n.SetIcon(a.iconPath)
	}
	n.SetDefaultAction("app.open-chat::" + chat)
	a.gapp.SendNotification(chat, n)
	if a.cfg.Sound {
		go exec.Command("canberra-gtk-play", "-i", "phone-incoming-call", "-d", "Obrol").Run()
	}
}
