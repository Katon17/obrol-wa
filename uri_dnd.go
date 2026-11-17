package main

// v1.9: helper konversi URI drag & drop menjadi path lokal.

import (
	"net/url"
	"strings"

	"github.com/gotk3/gotk3/gtk"
)

// selectionURIs mengambil daftar URI dari data drop TANPA memakai
// SelectionData.GetURIs(): fungsi gotk3 tersebut cacat (nextgcharptr maju
// 1 byte, bukan 8) dan menyebabkan SIGSEGV saat drag & drop (regresi upstream,
// masih ada sampai gotk3 master). Ambil teks mentahnya lalu urai sendiri.
func selectionURIs(sel *gtk.SelectionData) []string {
	// Data mentah (text/uri-list) dibaca berbasis panjang — GetText() bisa
	// mengembalikan NULL untuk tipe uri-list, dan GetURIs() cacat (lihat atas).
	txt := string(sel.GetData())
	if txt == "" {
		txt = sel.GetText()
	}
	if txt == "" {
		return nil
	}
	var out []string
	for _, line := range strings.Split(txt, "\n") {
		line = strings.TrimSpace(strings.TrimRight(line, "\r"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue // baris kosong / komentar sesuai spesifikasi URI list
		}
		out = append(out, line)
	}
	return out
}

func uriToPath(u string) string {
	if !strings.HasPrefix(u, "file://") {
		return ""
	}
	pu, err := url.Parse(u)
	if err != nil {
		return ""
	}
	return pu.Path
}
