package main

// tray.go — binding minimal GtkStatusIcon (ikon system tray XEmbed),
// didukung panel XFCE, MATE, Cinnamon, dll.

/*
#cgo pkg-config: gtk+-3.0
#cgo CFLAGS: -Wno-deprecated-declarations
#include <gtk/gtk.h>
#include <stdlib.h>
*/
import "C"

import (
	"unsafe"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
)

// trayIface: kontrak bersama wrapper XEmbed (StatusIcon) dan SNI (sni.go).
// v1.13.2: SNI dicoba lebih dulu (GNOME modern & sandbox Flatpak), jatuh
// kembali ke XEmbed bila watcher SNI tidak ada (panel tua).
type trayIface interface {
	SetFromPixbuf(pb *gdk.Pixbuf)
	SetTooltipText(t string)
	SetTitle(t string)
	SetVisible(v bool)
	IsEmbedded() bool
	Connect(signal string, f interface{})
}

func newTray(pb *gdk.Pixbuf) trayIface {
	if t := newSNI(pb); t != nil {
		return t
	}
	return NewStatusIcon(pb)
}

type StatusIcon struct {
	p   *C.GtkStatusIcon
	obj *glib.Object
}

func NewStatusIcon(pb *gdk.Pixbuf) *StatusIcon {
	p := C.gtk_status_icon_new_from_pixbuf((*C.GdkPixbuf)(unsafe.Pointer(pb.Native())))
	if p == nil {
		return nil
	}
	return &StatusIcon{p: p, obj: glib.Take(unsafe.Pointer(p))}
}

func (s *StatusIcon) SetFromPixbuf(pb *gdk.Pixbuf) {
	C.gtk_status_icon_set_from_pixbuf(s.p, (*C.GdkPixbuf)(unsafe.Pointer(pb.Native())))
}

func (s *StatusIcon) SetTooltipText(t string) {
	cs := C.CString(t)
	defer C.free(unsafe.Pointer(cs))
	C.gtk_status_icon_set_tooltip_text(s.p, (*C.gchar)(cs))
}

func (s *StatusIcon) SetTitle(t string) {
	cs := C.CString(t)
	defer C.free(unsafe.Pointer(cs))
	C.gtk_status_icon_set_title(s.p, (*C.gchar)(cs))
}

func (s *StatusIcon) SetVisible(v bool) {
	b := C.gboolean(0)
	if v {
		b = 1
	}
	C.gtk_status_icon_set_visible(s.p, b)
}

func (s *StatusIcon) IsEmbedded() bool { return C.gtk_status_icon_is_embedded(s.p) != 0 }

func (s *StatusIcon) Connect(signal string, f interface{}) { s.obj.Connect(signal, f) }
