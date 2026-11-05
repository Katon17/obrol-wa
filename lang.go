package main

// lang.go — penentuan bahasa antarmuka: pilihan eksplisit atau ikuti sistem.

import (
	"os"
	"strings"
)

// normalizeLang mengubah nama locale menjadi kode bahasa kanonis yang
// didukung ("auto", "id", "en"). Format seperti "id_ID.UTF-8" atau "en-US"
// diterima; locale yang tidak didukung mengembalikan ok=false.
func normalizeLang(s string) (string, bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return "", false
	}
	// Buang encoding ("id_ID.UTF-8") dan modifier ("en@modifier").
	if i := strings.IndexAny(s, ".@"); i >= 0 {
		s = s[:i]
	}
	// Ambil bagian bahasa saja, baik pemisahnya "-" maupun "_".
	if i := strings.IndexAny(s, "_-"); i >= 0 {
		s = s[:i]
	}
	switch s {
	case "auto":
		return "auto", true
	case "id":
		return "id", true
	case "en":
		return "en", true
	}
	return "", false
}

// systemLang membaca locale dari environment dan mengembalikan bahasa yang
// didukung. LANGUAGE (daftar dipisah ":") diprioritaskan, lalu LC_ALL,
// LC_MESSAGES, dan LANG. Bila tidak ada yang cocok, dipakai "en".
func systemLang() string {
	if v := os.Getenv("LANGUAGE"); v != "" {
		for _, part := range strings.Split(v, ":") {
			if code, ok := normalizeLang(part); ok && code != "auto" {
				return code
			}
		}
	}
	for _, key := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if code, ok := normalizeLang(os.Getenv(key)); ok && code != "auto" {
			return code
		}
	}
	return "en"
}

// resolveLang menentukan bahasa aktif dari setelan konfigurasi. Nilai
// eksplisit "id"/"en" menang; "auto", kosong, atau nilai tak dikenal
// mengikuti locale sistem.
func resolveLang(setting string) string {
	if code, ok := normalizeLang(setting); ok && code != "auto" {
		return code
	}
	return systemLang()
}
