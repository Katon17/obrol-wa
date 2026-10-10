package main

// Pembacaan clipboard berdasarkan target (mis. file yang di-copy dari Thunar).
// gotk3 SelectionData.native() salah meng-cast pointer, jadi pakai cgo langsung.

/*
#cgo pkg-config: gtk+-3.0
#include <gtk/gtk.h>
#include <stdlib.h>
#include <string.h>

static char *obrol_clip_target(const char *target, int *len) {
	GtkClipboard *cb = gtk_clipboard_get(GDK_SELECTION_CLIPBOARD);
	GdkAtom atom = gdk_atom_intern(target, FALSE);
	*len = 0;
	if (!gtk_clipboard_wait_is_target_available(cb, atom)) return NULL;
	GtkSelectionData *sd = gtk_clipboard_wait_for_contents(cb, atom);
	if (!sd) return NULL;
	gint n = 0;
	const guchar *d = gtk_selection_data_get_data_with_length(sd, &n);
	char *out = NULL;
	if (d && n > 0) {
		out = malloc(n);
		memcpy(out, d, n);
		*len = n;
	}
	gtk_selection_data_free(sd);
	return out;
}
*/
import "C"
import "unsafe"

// clipboardTarget mengembalikan isi clipboard untuk target MIME tertentu (nil bila tidak ada).
func clipboardTarget(target string) []byte {
	ct := C.CString(target)
	defer C.free(unsafe.Pointer(ct))
	var n C.int
	p := C.obrol_clip_target(ct, &n)
	if p == nil {
		return nil
	}
	defer C.free(unsafe.Pointer(p))
	return C.GoBytes(unsafe.Pointer(p), n)
}
