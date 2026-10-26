package main

// stickers.go — UI stiker: tampil di percakapan (termasuk animasi), panel stiker
// (Terbaru / Favorit / Buat), dan pembuat stiker dari gambar atau screenshot.

import (
	"fmt"
	"image"
	"math"
	"os"
	"strings"
	"time"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

const (
	stickerShow  = 150 // ukuran stiker di percakapan
	stickerCell  = 68  // ukuran stiker di panel
	stickerCache = 60  // jumlah stiker ter-decode yang disimpan di memori
)

type stickerAnim struct {
	frames []*gdk.Pixbuf
	delays []int
}

var (
	stickerMem     = map[string]*stickerAnim{}
	stickerMemKeys []string
	stickerPending = map[string][]func(*stickerAnim){}
)

// rgbaToPixbuf: image.RGBA (premultiplied) → GdkPixbuf (non-premultiplied).
func rgbaToPixbuf(img *image.RGBA) *gdk.Pixbuf {
	w, h := img.Rect.Dx(), img.Rect.Dy()
	pb, err := gdk.PixbufNew(gdk.COLORSPACE_RGB, true, 8, w, h)
	if err != nil {
		return nil
	}
	px := pb.GetPixels()
	rs := pb.GetRowstride()
	for y := 0; y < h; y++ {
		so := y * img.Stride
		do := y * rs
		for x := 0; x < w; x++ {
			r, g, b, a := img.Pix[so], img.Pix[so+1], img.Pix[so+2], img.Pix[so+3]
			if a != 0 && a != 255 {
				r = uint8(min(255, int(r)*255/int(a)))
				g = uint8(min(255, int(g)*255/int(a)))
				b = uint8(min(255, int(b)*255/int(a)))
			}
			px[do], px[do+1], px[do+2], px[do+3] = r, g, b, a
			so += 4
			do += 4
		}
	}
	return pb
}

// loadSticker men-decode stiker di goroutine, hasilnya di-cache. cb dipanggil di thread GTK.
func (a *App) loadSticker(sha string, size int, animate bool, cb func(*stickerAnim)) {
	key := fmt.Sprintf("%s|%d|%v", sha, size, animate)
	if s, ok := stickerMem[key]; ok {
		cb(s)
		return
	}
	if list, ok := stickerPending[key]; ok {
		stickerPending[key] = append(list, cb)
		return
	}
	stickerPending[key] = []func(*stickerAnim){cb}
	go func() {
		var frames []animFrame
		data, err := os.ReadFile(stickerPath(sha))
		if err == nil {
			maxF := 1
			if animate {
				maxF = 150
			}
			frames, err = decodeWebPFrames(data, size, maxF)
		}
		glib.IdleAdd(func() bool {
			var s *stickerAnim
			if err == nil {
				s = &stickerAnim{}
				for _, f := range frames {
					if pb := rgbaToPixbuf(f.img); pb != nil {
						s.frames = append(s.frames, pb)
						s.delays = append(s.delays, f.delay)
					}
				}
				if len(s.frames) == 0 {
					s = nil
				}
			}
			if s != nil {
				stickerMem[key] = s
				stickerMemKeys = append(stickerMemKeys, key)
				if len(stickerMemKeys) > stickerCache {
					delete(stickerMem, stickerMemKeys[0])
					stickerMemKeys = stickerMemKeys[1:]
				}
			}
			list := stickerPending[key]
			delete(stickerPending, key)
			for _, f := range list {
				f(s)
			}
			return false
		})
	}()
}

// playSticker menampilkan frame pertama dan, bila animasi, memutar frame berikutnya.
// Pemutaran berhenti saat percakapan dimuat ulang (msgGen berubah).
func (a *App) playSticker(img *gtk.Image, s *stickerAnim, gen int) {
	img.SetFromPixbuf(s.frames[0])
	if len(s.frames) < 2 {
		return
	}
	i := 0
	var tick func() bool
	tick = func() bool {
		if gen != a.msgGen {
			return false
		}
		if img.GetMapped() {
			i = (i + 1) % len(s.frames)
			img.SetFromPixbuf(s.frames[i])
			glib.TimeoutAdd(uint(s.delays[i]), tick)
		} else {
			glib.TimeoutAdd(400, tick) // jendela tersembunyi/di luar layar: hemat CPU
		}
		return false
	}
	glib.TimeoutAdd(uint(s.delays[0]), tick)
}

// stickerView: widget stiker untuk bubble pesan.
func (a *App) stickerView(m *Msg) gtk.IWidget {
	box, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	box.SetSizeRequest(stickerShow, stickerShow)
	box.SetHAlign(gtk.ALIGN_START)
	placeholder := func(tip string) {
		box.GetChildren().Foreach(func(it interface{}) {
			if w, ok := it.(*gtk.Widget); ok {
				w.Destroy()
			}
		})
		ic := newIcon("sticker", 44)
		addClass(ic, "stickerph")
		box.PackStart(ic, true, true, 0)
		box.SetTooltipText(tip)
		box.ShowAll()
	}
	ref, ok := stickerRef(m.Raw)
	if !ok {
		placeholder(T("Stiker"))
		return box
	}
	if ref.Lottie {
		if pb := pixbufFromBytes(ref.PNG); pb != nil {
			img, _ := gtk.ImageNewFromPixbuf(scalePix(pb, stickerShow))
			box.PackStart(img, true, true, 0)
			box.SetTooltipText(T("Stiker animasi Lottie (ditampilkan diam)"))
		} else {
			placeholder(T("Stiker animasi Lottie belum didukung"))
		}
		return box
	}
	gen := a.msgGen
	show := func() {
		a.loadSticker(ref.SHA, stickerShow, true, func(s *stickerAnim) {
			if gen != a.msgGen {
				return
			}
			if s == nil {
				placeholder(T("Stiker tidak bisa ditampilkan"))
				return
			}
			box.GetChildren().Foreach(func(it interface{}) {
				if w, ok := it.(*gtk.Widget); ok {
					w.Destroy()
				}
			})
			img, _ := gtk.ImageNew()
			box.PackStart(img, true, true, 0)
			box.SetTooltipText(T("Klik kanan untuk menyimpan ke favorit"))
			box.ShowAll()
			a.playSticker(img, s, gen)
		})
	}
	if stickerExists(ref.SHA) {
		a.st.AddSticker(ref.SHA, ref.Animated, m.TS)
		show()
	} else if a.wa != nil {
		placeholder(T("Mengunduh stiker…"))
		a.wa.FetchSticker(m.Raw, m.TS, func(err error) {
			glib.IdleAdd(func() bool {
				if gen != a.msgGen {
					return false
				}
				if err != nil {
					placeholder(T("Stiker tidak tersedia lagi di server"))
				} else {
					show()
				}
				return false
			})
		})
	} else {
		placeholder(T("Stiker"))
	}
	return box
}

// ------------------------------------------------------------ kirim

func (a *App) sendStickerTo(chat, sha string) {
	if chat == "" {
		return
	}
	var m *Msg
	if a.wa != nil {
		var err error
		if m, err = a.wa.SendSticker(chat, sha); err != nil {
			a.errorDialog(T("Gagal mengirim stiker"), err.Error())
			return
		}
	} else { // mode demo
		data, _ := os.ReadFile(stickerPath(sha))
		_, _, animated := webpInfo(data)
		m = &Msg{Chat: chat, ID: fmt.Sprint(time.Now().UnixNano()), FromMe: true, TS: time.Now().Unix(),
			Kind: "sticker", Raw: localStickerRaw(sha, animated), Status: StSent, IsRead: true}
		a.st.SaveMessage(m, false)
		a.st.TouchSticker(sha)
	}
	if chat == a.current {
		a.stickBottom = true
		a.appendMessage(m)
	}
	a.reloadChats()
}

// ------------------------------------------------------------ panel stiker

type stickerPanel struct {
	btn     *gtk.MenuButton
	pop     *gtk.Popover
	stack   *gtk.Stack
	recent  *gtk.FlowBox
	fav     *gtk.FlowBox
	recentE *gtk.Label
	favE    *gtk.Label
	cells   map[*gtk.FlowBox][]gtk.IWidget
}

func (a *App) buildStickerButton() gtk.IWidget {
	p := &stickerPanel{cells: map[*gtk.FlowBox][]gtk.IWidget{}}
	a.stickers = p
	p.btn, _ = gtk.MenuButtonNew()
	p.btn.SetRelief(gtk.RELIEF_NONE)
	p.btn.SetVAlign(gtk.ALIGN_END)
	p.btn.SetTooltipText(T("Stiker"))
	if old, err := p.btn.GetChild(); err == nil && old != nil {
		p.btn.Remove(old)
	}
	p.btn.Add(newIcon("sticker", 20))
	addClass(p.btn, "iconbtn")
	p.pop, _ = gtk.PopoverNew(p.btn)
	p.pop.SetPosition(gtk.POS_TOP)
	p.btn.SetPopover(p.pop)

	v, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 8)
	addClass(v, "stickerpanel")
	v.SetSizeRequest(352, 340)
	p.stack, _ = gtk.StackNew()
	p.stack.SetTransitionType(gtk.STACK_TRANSITION_TYPE_CROSSFADE)
	sw, _ := gtk.StackSwitcherNew()
	sw.SetStack(p.stack)
	sw.SetHAlign(gtk.ALIGN_CENTER)
	v.PackStart(sw, false, false, 0)
	v.PackStart(p.stack, true, true, 0)

	p.recent, p.recentE = a.stickerGrid(p, "recent", T("Terbaru"),
		T("Belum ada stiker.\nStiker yang kamu terima atau kirim\nakan muncul di sini."))
	p.fav, p.favE = a.stickerGrid(p, "fav", T("Favorit"),
		T("Belum ada favorit.\nKlik kanan sebuah stiker →\n«Tambah ke favorit»."))
	p.stack.AddTitled(a.stickerCreatePage(p), "create", T("Buat"))

	v.ShowAll()
	p.pop.Add(v)
	p.btn.Connect("toggled", func() {
		if p.btn.GetActive() {
			a.refreshStickerPanel()
		}
	})
	return p.btn
}

func (a *App) stickerGrid(p *stickerPanel, name, title, empty string) (*gtk.FlowBox, *gtk.Label) {
	ov, _ := gtk.OverlayNew()
	sw, _ := gtk.ScrolledWindowNew(nil, nil)
	sw.SetPolicy(gtk.POLICY_NEVER, gtk.POLICY_AUTOMATIC)
	fb, _ := gtk.FlowBoxNew()
	fb.SetSelectionMode(gtk.SELECTION_NONE)
	fb.SetHomogeneous(true)
	fb.SetMinChildrenPerLine(4)
	fb.SetMaxChildrenPerLine(4)
	fb.SetRowSpacing(4)
	fb.SetColumnSpacing(4)
	fb.SetVAlign(gtk.ALIGN_START)
	sw.Add(fb)
	ov.Add(sw)
	l, _ := gtk.LabelNew(empty)
	l.SetJustify(gtk.JUSTIFY_CENTER)
	addClass(l, "emptyhint")
	l.SetNoShowAll(true)
	ov.AddOverlay(l)
	p.stack.AddTitled(ov, name, T(title))
	return fb, l
}

func (a *App) refreshStickerPanel() {
	p := a.stickers
	if p == nil {
		return
	}
	fill := func(fb *gtk.FlowBox, empty *gtk.Label, list []StickerInfo) {
		for _, c := range p.cells[fb] {
			fb.Remove(c)
		}
		p.cells[fb] = nil
		empty.SetVisible(len(list) == 0)
		for _, si := range list {
			cell := a.stickerCellWidget(si)
			fb.Insert(cell, -1)
			p.cells[fb] = append(p.cells[fb], cell)
		}
		fb.ShowAll()
	}
	fill(p.recent, p.recentE, a.st.ListStickers(false, 64))
	favs := a.st.ListStickers(true, 64)
	fill(p.fav, p.favE, favs)
	if len(favs) > 0 && p.stack.GetVisibleChildName() == "" {
		p.stack.SetVisibleChildName("fav")
	}
}

func (a *App) stickerCellWidget(si StickerInfo) gtk.IWidget {
	b, _ := gtk.ButtonNew()
	b.SetRelief(gtk.RELIEF_NONE)
	addClass(b, "stickercell")
	img, _ := gtk.ImageNew()
	img.SetSizeRequest(stickerCell, stickerCell)
	b.Add(img)
	if si.Animated {
		b.SetTooltipText(T("Stiker animasi — klik untuk mengirim"))
	} else {
		b.SetTooltipText(T("Klik untuk mengirim"))
	}
	a.loadSticker(si.SHA, stickerCell, false, func(s *stickerAnim) {
		if s != nil {
			img.SetFromPixbuf(s.frames[0])
		}
	})
	sha := si.SHA
	b.Connect("clicked", func() {
		a.stickers.pop.Popdown()
		a.sendStickerTo(a.current, sha)
	})
	b.Connect("button-press-event", func(_ *gtk.Button, ev *gdk.Event) bool {
		if gdk.EventButtonNewFromEvent(ev).Button() != gdk.BUTTON_SECONDARY {
			return false
		}
		menu, _ := gtk.MenuNew()
		if a.st.IsStickerFav(sha) {
			menu.Append(menuItem("star", T("Hapus dari favorit"), func() { a.st.SetStickerFav(sha, false); a.refreshStickerPanel() }))
		} else {
			menu.Append(menuItem("star-fill", T("Tambah ke favorit"), func() { a.st.SetStickerFav(sha, true); a.refreshStickerPanel() }))
		}
		menu.Append(menuItem("trash", T("Hapus dari daftar"), func() { a.st.HideSticker(sha); a.refreshStickerPanel() }))
		menu.ShowAll()
		menu.PopupAtPointer(ev)
		return true
	})
	return b
}

func (a *App) stickerCreatePage(p *stickerPanel) gtk.IWidget {
	v, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 10)
	v.SetVAlign(gtk.ALIGN_CENTER)
	v.SetMarginStart(16)
	v.SetMarginEnd(16)
	ic := newIcon("sticker", 56)
	addClass(ic, "accent")
	v.PackStart(ic, false, false, 0)
	t, _ := gtk.LabelNew("")
	t.SetMarkup("<b>" + T("Buat stiker sendiri") + "</b>\n<small>" +
		T("Ubah foto atau screenshot jadi stiker WhatsApp.\nBisa dipotong persegi, sudut membulat, dan diberi\ngaris tepi putih ala stiker.") +
		"</small>")
	t.SetJustify(gtk.JUSTIFY_CENTER)
	v.PackStart(t, false, false, 4)
	mk := func(icon, label string, fn func()) *gtk.Button {
		b, _ := gtk.ButtonNew()
		box, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 8)
		box.SetHAlign(gtk.ALIGN_CENTER)
		box.PackStart(newIcon(icon, 18), false, false, 0)
		l, _ := gtk.LabelNew(label)
		box.PackStart(l, false, false, 0)
		b.Add(box)
		b.Connect("clicked", fn)
		return b
	}
	file := mk("image", T("Pilih gambar…"), func() {
		p.pop.Popdown()
		a.stickerFromFile()
	})
	paste := mk("paste", T("Tempel dari clipboard"), func() {
		p.pop.Popdown()
		a.stickerFromClipboard()
	})
	addClass(file, "suggested-action")
	v.PackStart(file, false, false, 0)
	v.PackStart(paste, false, false, 0)
	if !webpEncoderAvailable() {
		file.SetSensitive(false)
		paste.SetSensitive(false)
		w, _ := gtk.LabelNew(T("libwebp tidak ditemukan.\nPasang dulu: sudo apt install libwebp7"))
		addClass(w, "warn")
		w.SetJustify(gtk.JUSTIFY_CENTER)
		v.PackStart(w, false, false, 0)
	}
	return v
}

func (a *App) stickerFromFile() {
	fc, _ := gtk.FileChooserNativeDialogNew(T("Pilih gambar untuk stiker"), a.win, gtk.FILE_CHOOSER_ACTION_OPEN, T("Pilih"), T("Batal"))
	flt, _ := gtk.FileFilterNew()
	flt.SetName(T("Gambar"))
	flt.AddPixbufFormats()
	fc.AddFilter(flt)
	if gtk.ResponseType(fc.Run()) != gtk.RESPONSE_ACCEPT {
		fc.Destroy()
		return
	}
	path := fc.GetFilename()
	fc.Destroy()
	pb, err := gdk.PixbufNewFromFile(path)
	if err != nil {
		a.errorDialog(T("Gambar tidak bisa dibuka"), err.Error())
		return
	}
	a.createStickerDialog(pb)
}

func (a *App) stickerFromClipboard() {
	cb, err := gtk.ClipboardGet(gdk.SELECTION_CLIPBOARD)
	if err == nil && cb.WaitIsImageAvailable() {
		if pb, err := cb.WaitForImage(); err == nil && pb != nil {
			a.createStickerDialog(pb)
			return
		}
	}
	for _, t := range []string{"x-special/gnome-copied-files", "text/uri-list"} {
		if files := parseFileList(string(clipboardTarget(t))); len(files) > 0 && isImagePath(files[0]) {
			if pb, err := gdk.PixbufNewFromFile(files[0]); err == nil {
				a.createStickerDialog(pb)
				return
			}
		}
	}
	a.errorDialog(T("Clipboard tidak berisi gambar"), T("Salin gambar atau ambil screenshot (PrtSc) dulu, lalu coba lagi."))
}

// ------------------------------------------------------------ pembuat stiker

type stickerOpts struct{ crop, round, outline bool }

// makeStickerRGBA menghasilkan kanvas 512×512 RGBA (non-premultiplied).
func makeStickerRGBA(src *gdk.Pixbuf, o stickerOpts) ([]byte, error) {
	if !src.GetHasAlpha() {
		if src = src.AddAlpha(false, 0, 0, 0); src == nil {
			return nil, fmt.Errorf(T("gambar tidak valid"))
		}
	}
	const S = stickerSize
	margin := 0
	if o.outline {
		margin = 20
	}
	box := S - 2*margin
	w, h := src.GetWidth(), src.GetHeight()
	dst, err := gdk.PixbufNew(gdk.COLORSPACE_RGB, true, 8, S, S)
	if err != nil {
		return nil, err
	}
	dst.Fill(0)
	var cx, cy, cw, ch int // area konten di kanvas
	if o.crop {
		s := float64(box) / float64(min(w, h))
		offX := float64(margin) - (float64(w)*s-float64(box))/2
		offY := float64(margin) - (float64(h)*s-float64(box))/2
		src.Scale(dst, margin, margin, box, box, offX, offY, s, s, gdk.INTERP_HYPER)
		cx, cy, cw, ch = margin, margin, box, box
	} else {
		s := float64(box) / float64(max(w, h))
		sw, sh := max(1, int(math.Round(float64(w)*s))), max(1, int(math.Round(float64(h)*s)))
		cx, cy, cw, ch = (S-sw)/2, (S-sh)/2, sw, sh
		src.Scale(dst, cx, cy, sw, sh, float64(cx), float64(cy), s, s, gdk.INTERP_HYPER)
	}
	px := dst.GetPixels()
	rs := dst.GetRowstride()
	out := make([]byte, S*S*4)
	for y := 0; y < S; y++ {
		copy(out[y*S*4:(y+1)*S*4], px[y*rs:y*rs+S*4])
	}
	if o.round {
		roundCorners(out, S, cx, cy, cw, ch, 0.16*float64(min(cw, ch)))
	}
	if o.outline {
		addOutline(out, S, 12)
	}
	return out, nil
}

// roundCorners mengalikan alpha dengan bentuk persegi bersudut bulat (anti-alias).
func roundCorners(p []byte, S, x0, y0, w, h int, r float64) {
	for y := y0; y < y0+h; y++ {
		for x := x0; x < x0+w; x++ {
			fx, fy := float64(x-x0)+0.5, float64(y-y0)+0.5
			dx := math.Max(0, math.Max(r-fx, fx-(float64(w)-r)))
			dy := math.Max(0, math.Max(r-fy, fy-(float64(h)-r)))
			if dx == 0 || dy == 0 {
				continue
			}
			cov := math.Min(1, math.Max(0, r-math.Hypot(dx, dy)+0.5))
			o := (y*S + x) * 4
			p[o+3] = uint8(float64(p[o+3]) * cov)
		}
	}
}

// addOutline menambahkan garis tepi putih selebar rad px di sekeliling bagian yang tidak transparan.
func addOutline(p []byte, S int, rad float64) {
	const inf = 1e9
	d := make([]float64, S*S)
	for i := range d {
		if p[i*4+3] > 110 {
			d[i] = 0
		} else {
			d[i] = inf
		}
	}
	// transformasi jarak chamfer 2-lintasan (cukup akurat untuk garis tepi)
	const a, b = 1.0, math.Sqrt2
	for y := 0; y < S; y++ {
		for x := 0; x < S; x++ {
			i := y*S + x
			v := d[i]
			if x > 0 {
				v = math.Min(v, d[i-1]+a)
			}
			if y > 0 {
				v = math.Min(v, d[i-S]+a)
				if x > 0 {
					v = math.Min(v, d[i-S-1]+b)
				}
				if x < S-1 {
					v = math.Min(v, d[i-S+1]+b)
				}
			}
			d[i] = v
		}
	}
	for y := S - 1; y >= 0; y-- {
		for x := S - 1; x >= 0; x-- {
			i := y*S + x
			v := d[i]
			if x < S-1 {
				v = math.Min(v, d[i+1]+a)
			}
			if y < S-1 {
				v = math.Min(v, d[i+S]+a)
				if x < S-1 {
					v = math.Min(v, d[i+S+1]+b)
				}
				if x > 0 {
					v = math.Min(v, d[i+S-1]+b)
				}
			}
			d[i] = v
		}
	}
	for i := 0; i < S*S; i++ {
		oa := math.Min(1, math.Max(0, rad+0.5-d[i])) // alpha garis tepi
		if oa == 0 {
			continue
		}
		o := i * 4
		sa := float64(p[o+3]) / 255
		ra := sa + oa*(1-sa)
		for c := 0; c < 3; c++ {
			sc := float64(p[o+c]) / 255
			p[o+c] = uint8(math.Round((sc*sa + 1*oa*(1-sa)) / ra * 255))
		}
		p[o+3] = uint8(math.Round(ra * 255))
	}
}

// previewPixbuf: kanvas stiker di atas pola kotak-kotak (agar transparansi terlihat).
func previewPixbuf(rgba []byte, S, size int, dark bool) *gdk.Pixbuf {
	pb, err := gdk.PixbufNew(gdk.COLORSPACE_RGB, true, 8, S, S)
	if err != nil {
		return nil
	}
	px := pb.GetPixels()
	rs := pb.GetRowstride()
	c1, c2 := 255.0, 230.0
	if dark {
		c1, c2 = 52, 40
	}
	for y := 0; y < S; y++ {
		for x := 0; x < S; x++ {
			o := (y*S + x) * 4
			bg := c1
			if (x/32+y/32)%2 == 1 {
				bg = c2
			}
			al := float64(rgba[o+3]) / 255
			d := y*rs + x*4
			for c := 0; c < 3; c++ {
				px[d+c] = uint8(float64(rgba[o+c])*al + bg*(1-al))
			}
			px[d+3] = 255
		}
	}
	if s, err := pb.ScaleSimple(size, size, gdk.INTERP_BILINEAR); err == nil {
		return s
	}
	return pb
}

func (a *App) createStickerDialog(src *gdk.Pixbuf) {
	chat := a.current
	dlg, _ := gtk.DialogNew()
	dlg.SetTransientFor(a.win)
	dlg.SetModal(true)
	dlg.SetTitle(T("Buat stiker"))
	dlg.SetResizable(false)
	dlg.AddButton(T("Batal"), gtk.RESPONSE_CANCEL)
	saveBtn, _ := dlg.AddButton(T("Simpan saja"), gtk.RESPONSE_APPLY)
	saveBtn.SetTooltipText(T("Simpan ke favorit tanpa mengirim"))
	okLabel := T("Kirim stiker")
	if chat == "" {
		okLabel = T("Simpan")
	}
	ok, _ := dlg.AddButton(okLabel, gtk.RESPONSE_OK)
	addClass(ok, "suggested-action")
	dlg.SetDefaultResponse(gtk.RESPONSE_OK)

	area, _ := dlg.GetContentArea()
	h, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 18)
	h.SetMarginStart(16)
	h.SetMarginEnd(16)
	h.SetMarginTop(16)
	h.SetMarginBottom(8)
	area.PackStart(h, true, true, 0)
	img, _ := gtk.ImageNew()
	img.SetSizeRequest(256, 256)
	addClass(img, "thumb")
	h.PackStart(img, false, false, 0)

	opts, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 6)
	opts.SetVAlign(gtk.ALIGN_CENTER)
	title, _ := gtk.LabelNew("")
	title.SetMarkup("<b>" + T("Bentuk") + "</b>")
	title.SetXAlign(0)
	fit, _ := gtk.RadioButtonNewWithLabel(nil, T("Gambar utuh"))
	crop, _ := gtk.RadioButtonNewWithLabelFromWidget(fit, T("Potong persegi (tengah)"))
	if w, hh := src.GetWidth(), src.GetHeight(); float64(max(w, hh))/float64(max(1, min(w, hh))) < 1.35 {
		crop.SetActive(true)
	}
	t2, _ := gtk.LabelNew("")
	t2.SetMarkup("<b>" + T("Gaya") + "</b>")
	t2.SetXAlign(0)
	round, _ := gtk.CheckButtonNewWithLabel(T("Sudut membulat"))
	round.SetActive(true)
	outline, _ := gtk.CheckButtonNewWithLabel(T("Garis tepi putih"))
	outline.SetActive(src.GetHasAlpha())
	outline.SetTooltipText(T("Paling cocok untuk gambar berlatar transparan (PNG)"))
	fav, _ := gtk.CheckButtonNewWithLabel(T("Simpan ke favorit"))
	fav.SetActive(true)
	info, _ := gtk.LabelNew("")
	addClass(info, "when")
	info.SetXAlign(0)
	for _, w := range []gtk.IWidget{title, fit, crop, t2, round, outline} {
		opts.PackStart(w, false, false, 0)
	}
	sep, _ := gtk.SeparatorNew(gtk.ORIENTATION_HORIZONTAL)
	opts.PackStart(sep, false, false, 6)
	opts.PackStart(fav, false, false, 0)
	opts.PackStart(info, false, false, 4)
	h.PackStart(opts, false, false, 0)

	var cur []byte
	update := func() {
		o := stickerOpts{crop: crop.GetActive(), round: round.GetActive(), outline: outline.GetActive()}
		rgba, err := makeStickerRGBA(src, o)
		if err != nil {
			info.SetText("⚠ " + err.Error())
			return
		}
		cur = rgba
		img.SetFromPixbuf(previewPixbuf(rgba, stickerSize, 256, a.dark))
		info.SetText(fmt.Sprintf("512 × 512 px · WebP\n"+T("sumber")+" %d × %d px", src.GetWidth(), src.GetHeight()))
	}
	fit.Connect("toggled", update)
	round.Connect("toggled", update)
	outline.Connect("toggled", update)
	update()
	dlg.ShowAll()
	ok.GrabFocus()
	resp := dlg.Run()
	keepFav := fav.GetActive()
	dlg.Destroy()
	if resp != gtk.RESPONSE_OK && resp != gtk.RESPONSE_APPLY || cur == nil {
		return
	}
	data, err := encodeSticker(cur, stickerSize, stickerSize)
	if err != nil {
		a.errorDialog(T("Gagal membuat stiker"), err.Error())
		return
	}
	sha, err := saveStickerFile(data)
	if err != nil {
		a.errorDialog(T("Gagal menyimpan stiker"), err.Error())
		return
	}
	a.st.AddSticker(sha, false, time.Now().Unix())
	if keepFav || resp == gtk.RESPONSE_APPLY {
		a.st.SetStickerFav(sha, true)
	}
	if resp == gtk.RESPONSE_OK && chat != "" {
		a.sendStickerTo(chat, sha)
	} else {
		a.flashStatus(T("Stiker disimpan ke favorit"))
	}
}

// stickerMenuItems: aksi stiker untuk menu klik kanan pesan.
func (a *App) stickerMenuItems(menu *gtk.Menu, m Msg) {
	ref, ok := stickerRef(m.Raw)
	if !ok || !stickerExists(ref.SHA) {
		return
	}
	sha := ref.SHA
	if a.st.IsStickerFav(sha) {
		menu.Append(menuItem("star", T("Hapus dari favorit stiker"), func() { a.st.SetStickerFav(sha, false) }))
	} else {
		menu.Append(menuItem("star-fill", T("Tambah ke favorit stiker"), func() {
			a.st.SetStickerFav(sha, true)
			a.flashStatus(T("Stiker ditambahkan ke favorit"))
		}))
	}
	menu.Append(menuItem("send", T("Kirim stiker ini lagi"), func() { a.sendStickerTo(a.current, sha) }))
}

var _ = strings.TrimSpace
