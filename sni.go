package main

// sni.go — StatusNotifierItem (tray DBus modern) untuk GNOME/KDE/Flatpak,
// diimplementasikan langsung di atas GDBus via cgo (sni.c/sni.h) TANPA
// dependensi Go baru: percobaan dengan godbus/dbus/v5 menambah ±13 MB ke
// biner karena interaksi reflect/generics dengan gotk3.
// v1.13.2: GtkStatusIcon (XEmbed) mati di GNOME modern dan tidak tersedia di
// sandbox Flatpak; SNI adalah penggantinya. Bila watcher SNI tidak ada (panel
// tua berbasis XEmbed), setupTray jatuh kembali ke GtkStatusIcon.
//
// Catatan cgo: file ini memakai //export, jadi preamble hanya boleh berisi
// deklarasi — semua definisi C hidup di sni.c.

/*
#cgo pkg-config: gtk+-3.0 gio-2.0
#cgo CFLAGS: -Wno-deprecated-declarations
#include <stdlib.h>
#include "sni.h"
*/
import "C"

import (
	"sync"
	"unsafe"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
)

const sniBusName = "io.github.Katon17.Obrol.SNI"

type sniTray struct {
	mu       sync.Mutex
	title    string
	tooltip  string
	status   string
	handlers map[string]func()
	menuShow func()
	menuQuit func()
	lastX    int
	lastY    int
}

func newSNI(pb *gdk.Pixbuf) *sniTray {
	t := &sniTray{
		title:    "Obrol",
		status:   "Active",
		handlers: map[string]func(){},
	}
	t.pushPixbuf(pb)
	t.mu.Lock()
	t.pushStrings()
	t.mu.Unlock()
	cname := C.CString(sniBusName)
	defer C.free(unsafe.Pointer(cname))
	if C.obrol_sni_start(cname) == 0 {
		return nil
	}
	sniMu.Lock()
	sniLive = t
	sniMu.Unlock()
	return t
}

// pushPixbuf: RGB/RGBA → ARGB32 big-endian (format IconPixmap SNI) ke sisi C.
func (t *sniTray) pushPixbuf(pb *gdk.Pixbuf) {
	if pb == nil {
		C.obrol_sni_set_pixels(nil, 0, 0, 0)
		return
	}
	w, h := pb.GetWidth(), pb.GetHeight()
	rowstride, ch := pb.GetRowstride(), pb.GetNChannels()
	raw := pb.GetPixels()
	out := make([]byte, 0, w*h*4)
	for y := 0; y < h; y++ {
		row := y * rowstride
		for x := 0; x < w; x++ {
			i := row + x*ch
			if i+2 >= len(raw) {
				out = append(out, 0, 0, 0, 0)
				continue
			}
			var a byte = 255
			if ch == 4 {
				a = raw[i+3]
			}
			out = append(out, a, raw[i], raw[i+1], raw[i+2])
		}
	}
	C.obrol_sni_set_pixels(unsafe.Pointer(&out[0]), C.int(len(out)), C.int(w), C.int(h))
}

func (t *sniTray) pushStrings() {
	ct := C.CString(t.title)
	ctt := C.CString(t.tooltip)
	cs := C.CString(t.status)
	C.obrol_sni_set_strings(ct, ctt, cs)
	C.free(unsafe.Pointer(ct))
	C.free(unsafe.Pointer(ctt))
	C.free(unsafe.Pointer(cs))
}

//export goSniMethod
func goSniMethod(name *C.char, x, y C.gint) {
	signal := "activate"
	if C.GoString(name) == "ContextMenu" {
		signal = "popup-menu"
	}
	if signal == "popup-menu" {
		sniMu.Lock()
		t := sniLive
		sniMu.Unlock()
		if t != nil {
			t.mu.Lock()
			t.lastX, t.lastY = int(x), int(y)
			t.mu.Unlock()
		}
	}
	sniDispatch(signal)
}

var (
	sniMu   sync.Mutex
	sniLive *sniTray
)

func sniDispatch(signal string) {
	sniMu.Lock()
	t := sniLive
	sniMu.Unlock()
	if t == nil {
		return
	}
	t.mu.Lock()
	h := t.handlers[signal]
	t.mu.Unlock()
	if h == nil {
		return
	}
	glib.IdleAdd(func() { h() })
}

//export goSniMenuEvent
func goSniMenuEvent(id C.int) {
	sniMu.Lock()
	t := sniLive
	sniMu.Unlock()
	if t == nil {
		return
	}
	t.mu.Lock()
	var h func()
	switch int(id) {
	case 1:
		h = t.menuShow
	case 3:
		h = t.menuQuit
	}
	t.mu.Unlock()
	if h != nil {
		glib.IdleAdd(func() { h() })
	}
}

// ---------- antarmuka trayIface (sama dengan wrapper GtkStatusIcon) ----------

func (t *sniTray) SetFromPixbuf(pb *gdk.Pixbuf) {
	t.pushPixbuf(pb)
	c := C.CString("NewIcon")
	defer C.free(unsafe.Pointer(c))
	C.obrol_sni_emit(c)
}

func (t *sniTray) SetTooltipText(s string) {
	t.mu.Lock()
	t.tooltip = s
	t.pushStrings()
	t.mu.Unlock()
	c := C.CString("NewToolTip")
	defer C.free(unsafe.Pointer(c))
	C.obrol_sni_emit(c)
}

func (t *sniTray) SetTitle(s string) {
	t.mu.Lock()
	t.title = s
	t.pushStrings()
	t.mu.Unlock()
	c := C.CString("NewTitle")
	defer C.free(unsafe.Pointer(c))
	C.obrol_sni_emit(c)
}

func (t *sniTray) SetVisible(v bool) {
	t.mu.Lock()
	if v {
		t.status = "Active"
	} else {
		t.status = "Passive"
	}
	t.pushStrings()
	t.mu.Unlock()
	c := C.CString("NewStatus")
	defer C.free(unsafe.Pointer(c))
	C.obrol_sni_emit(c)
}

func (t *sniTray) IsEmbedded() bool { return true }

func (t *sniTray) Connect(signal string, f interface{}) {
	if h, ok := f.(func()); ok {
		t.mu.Lock()
		t.handlers[signal] = h
		t.mu.Unlock()
	}
}

// SetMenuItems mendaftarkan aksi menu klik-kanan yang dirender native oleh
// host SNI lewat com.canonical.dbusmenu (id 1 = tampilkan, 3 = keluar);
// label ikut T() supaya sesuai bahasa antarmuka.
func (t *sniTray) SetMenuItems(show, quit func()) {
	t.mu.Lock()
	t.menuShow, t.menuQuit = show, quit
	t.mu.Unlock()
	cs := C.CString(T("Tampilkan Obrol"))
	cq := C.CString(T("Keluar"))
	C.obrol_sni_set_menu_labels(cs, cq)
	C.free(unsafe.Pointer(cs))
	C.free(unsafe.Pointer(cq))
}

// MenuCalls = jumlah panggilan com.canonical.dbusmenu dari host; 0 berkelanjutan
// berarti host tidak mampu merender menu kita (mis. panel tanpa libdbusmenu).
func (t *sniTray) MenuCalls() int { return int(C.obrol_sni_menu_calls()) }

// MenuPoint mengembalikan koordinat layar terakhir dari ContextMenu() host,
// supaya GtkMenu fallback bisa muncul tepat di kursor meski tanpa event GDK.
func (t *sniTray) MenuPoint() (int, int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.lastX, t.lastY
}
