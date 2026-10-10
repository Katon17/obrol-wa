package main

// icons.go — set ikon garis Obrol (assets/icons/*.svg) yang otomatis mengikuti
// warna teks tema (terang/gelap, baris terpilih, item nonaktif), plus logo SVG.

import (
	"embed"
	"fmt"
	"strings"

	"github.com/gotk3/gotk3/cairo"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
)

//go:embed assets/icons/*.svg
var iconFS embed.FS

//go:embed assets/obrol.svg
var logoSVG []byte

//go:embed assets/obrol-unread.svg
var logoUnreadSVG []byte

var iconCache = map[string]*gdk.Pixbuf{}

// svgPixbuf merender SVG pada ukuran tertentu (lewat loader librsvg gdk-pixbuf).
func svgPixbuf(svg []byte, size int) *gdk.Pixbuf {
	s := strings.Replace(string(svg), "<svg ", fmt.Sprintf(`<svg width="%d" height="%d" `, size, size), 1)
	return pixbufFromBytes([]byte(s))
}

func iconPixbuf(name string, size int, color string) *gdk.Pixbuf {
	key := fmt.Sprintf("%s|%d|%s", name, size, color)
	if pb, ok := iconCache[key]; ok {
		return pb
	}
	data, err := iconFS.ReadFile("assets/icons/" + name + ".svg")
	if err != nil {
		return nil
	}
	pb := svgPixbuf([]byte(strings.ReplaceAll(string(data), "currentColor", color)), size)
	iconCache[key] = pb
	return pb
}

// Icon adalah widget ikon kecil yang menggambar SVG dengan warna CSS `color` widget.
type Icon struct {
	*gtk.DrawingArea
	name string
	size int
}

func newIcon(name string, size int) *Icon {
	da, _ := gtk.DrawingAreaNew()
	ic := &Icon{DrawingArea: da, name: name, size: size}
	da.SetSizeRequest(size, size)
	da.SetHAlign(gtk.ALIGN_CENTER)
	da.SetVAlign(gtk.ALIGN_CENTER)
	addClass(da, "icon")
	da.Connect("draw", func(w *gtk.DrawingArea, cr *cairo.Context) bool {
		if ic.name == "" {
			return false
		}
		sc, _ := w.GetStyleContext()
		c := sc.GetColor(sc.GetState())
		f := c.Floats()
		col := fmt.Sprintf("#%02x%02x%02x", int(f[0]*255+.5), int(f[1]*255+.5), int(f[2]*255+.5))
		sf := w.GetScaleFactor()
		if sf < 1 {
			sf = 1
		}
		pb := iconPixbuf(ic.name, ic.size*sf, col)
		if pb == nil {
			return false
		}
		W, H := w.GetAllocatedWidth(), w.GetAllocatedHeight()
		cr.Save()
		cr.Scale(1/float64(sf), 1/float64(sf))
		gtk.GdkCairoSetSourcePixBuf(cr, pb, float64((W-ic.size)*sf/2), float64((H-ic.size)*sf/2))
		cr.PaintWithAlpha(f[3])
		cr.Restore()
		return false
	})
	return ic
}

func (ic *Icon) SetName(name string) {
	if ic.name != name {
		ic.name = name
		ic.QueueDraw()
	}
}

// iconButton: tombol datar berisi ikon.
func iconButton(name, tooltip string, size int) *gtk.Button {
	b, _ := gtk.ButtonNew()
	b.Add(newIcon(name, size))
	b.SetRelief(gtk.RELIEF_NONE)
	b.SetTooltipText(tooltip)
	b.SetVAlign(gtk.ALIGN_CENTER)
	addClass(b, "iconbtn")
	return b
}

// menuItem: item menu dengan ikon di kiri. fn == nil → item nonaktif (informasi).
func menuItem(icon, label string, fn func()) *gtk.MenuItem {
	it, _ := gtk.MenuItemNew()
	box, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 10)
	if icon != "" {
		box.PackStart(newIcon(icon, 16), false, false, 0)
	} else {
		sp, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
		sp.SetSizeRequest(16, 16)
		box.PackStart(sp, false, false, 0)
	}
	l, _ := gtk.LabelNew(label)
	l.SetXAlign(0)
	box.PackStart(l, true, true, 0)
	it.Add(box)
	if fn == nil {
		it.SetSensitive(false)
	} else {
		it.Connect("activate", fn)
	}
	return it
}

// iconLabel: ikon + teks dalam satu baris (untuk label informasi).
func iconLabel(icon, text string, size int) (*gtk.Box, *gtk.Label) {
	box, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 6)
	box.PackStart(newIcon(icon, size), false, false, 0)
	l, _ := gtk.LabelNew(text)
	l.SetXAlign(0)
	box.PackStart(l, false, false, 0)
	return box, l
}
