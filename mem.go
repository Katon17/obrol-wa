package main

// v1.9.5: pengembalian memori C ke sistem operasi.
//
// Pixbuf (thumbnail, frame stiker) dialokasikan di memori C. Setelah dibebaskan
// pun, glibc biasanya tidak mengembalikannya ke OS (menahan di arena malloc),
// sehingga RSS terlihat menetap tinggi. malloc_trim(0) memaksa arena utama
// mengembalikan bagian bebas di puncak heap ke sistem.

/*
#include <malloc.h>
*/
import "C"

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/gotk3/gotk3/glib"
)

// v1.9.7: strategi arena malloc disetel SEKALI di awal proses, sebelum
// thread C lain lahir. Masalah lama: memori C yang sudah dilepas (pixbuf,
// cache pango) tertahan di arena sebagai "high-water mark" — RSS menempel
// di puncak walau logisnya sudah bebas. Dengan mmap-threshold tetap 128 KB,
// alokasi besar dipetakan lewat mmap dan langsung dikembalikan ke OS saat
// free(); arena dibatasi 2 supaya trim thread utama mencakup hampir semua.
func init() {
	C.mallopt(C.M_ARENA_MAX, 2)
	C.mallopt(C.M_MMAP_THRESHOLD, 128*1024)
	C.mallopt(C.M_TRIM_THRESHOLD, 128*1024)
}

// trimMem: satu siklus GC (menjalankan finalizer pixbuf) lalu trim arena malloc
// thread pemanggil — mayoritas pixbuf dibuat di thread GTK utama.
func trimMem() {
	runtime.GC()
	C.malloc_trim(0)
}

// startMemTicker: v1.9.6 — trim berkala tiap 60 dtk di thread GTK utama supaya
// sesi panjang tetap ramping walau pengguna jarang ganti chat.
func startMemTicker() {
	glib.TimeoutAdd(60000, func() bool {
		trimMem()
		logMem()
		return true
	})
}

// logMem: satu baris diagnostik per menit ke mem.log di folder data —
// bukti lapangan pemakaian RAM (RSS vs heap Go vs cache stiker).
func logMem() {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	line := fmt.Sprintf("%s rss=%dKB heap=%dKB sys=%dKB gc=%d stickers=%d\n",
		time.Now().Format("15:04:05"), vmRSSKB(), ms.HeapAlloc/1024, ms.Sys/1024, ms.NumGC, stickerCount())
	dir := appDir
	if dir == "" {
		dir = dataDir()
	}
	_ = os.MkdirAll(dir, 0o755)
	f, err := os.OpenFile(filepath.Join(dir, "mem.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	_, _ = f.WriteString(line)
	_ = f.Close()
}

func vmRSSKB() int64 {
	b, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return 0
	}
	for _, ln := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(ln, "VmRSS:") {
			var v int64
			_, _ = fmt.Sscanf(ln, "VmRSS: %d kB", &v)
			return v
		}
	}
	return 0
}
