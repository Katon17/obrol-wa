package main

// webp.go — decode stiker WebP (statis & animasi) dan encode stiker baru.
//
// Decode memakai golang.org/x/image/webp (pure Go). Paket itu tidak mendukung
// WebP animasi, jadi container animasi (ANMF) diurai sendiri: tiap frame dibungkus
// ulang jadi WebP statis, di-decode, lalu dikomposisikan ke kanvas.
//
// Encode memakai libwebp sistem lewat dlopen (libwebp.so.7 selalu ada di Mint
// karena dipakai libtiff/gdk-pixbuf), jadi tidak menambah dependensi build.

/*
#cgo LDFLAGS: -ldl
#include <dlfcn.h>
#include <stdint.h>
#include <stdlib.h>

typedef size_t (*obrol_enc_fn)(const uint8_t*, int, int, int, float, uint8_t**);
typedef void (*obrol_free_fn)(void*);
static void *obrol_webp_h;
static obrol_enc_fn obrol_enc;
static obrol_free_fn obrol_wfree;

static int obrol_webp_init(void) {
	if (obrol_enc) return 1;
	const char *names[] = {"libwebp.so.7", "libwebp.so.6", "libwebp.so", NULL};
	for (int i = 0; names[i] && !obrol_webp_h; i++) obrol_webp_h = dlopen(names[i], RTLD_LAZY);
	if (!obrol_webp_h) return 0;
	obrol_enc = (obrol_enc_fn)dlsym(obrol_webp_h, "WebPEncodeRGBA");
	obrol_wfree = (obrol_free_fn)dlsym(obrol_webp_h, "WebPFree");
	if (!obrol_wfree) obrol_wfree = free;
	return obrol_enc != NULL;
}

static size_t obrol_webp_encode(const uint8_t *rgba, int w, int h, float q, uint8_t **out) {
	if (!obrol_webp_init()) return 0;
	return obrol_enc(rgba, w, h, w * 4, q, out);
}

static void obrol_webp_free(void *p) { if (p) obrol_wfree(p); }
*/
import "C"

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/draw"
	"unsafe"

	"golang.org/x/image/webp"
)

// webpEncoderAvailable: true bila libwebp bisa dimuat (untuk membuat stiker).
func webpEncoderAvailable() bool { return C.obrol_webp_init() != 0 }

// encodeWebP meng-encode RGBA (non-premultiplied, stride w*4) jadi WebP lossy.
func encodeWebP(rgba []byte, w, h int, quality float32) ([]byte, error) {
	if len(rgba) < w*h*4 {
		return nil, errors.New("buffer gambar tidak valid")
	}
	var out *C.uint8_t
	n := C.obrol_webp_encode((*C.uint8_t)(unsafe.Pointer(&rgba[0])), C.int(w), C.int(h), C.float(quality), &out)
	if n == 0 || out == nil {
		if !webpEncoderAvailable() {
			return nil, errors.New("libwebp tidak ditemukan (sudo apt install libwebp7)")
		}
		return nil, errors.New("encode WebP gagal")
	}
	defer C.obrol_webp_free(unsafe.Pointer(out))
	return C.GoBytes(unsafe.Pointer(out), C.int(n)), nil
}

// encodeSticker mencari kualitas tertinggi yang masih di bawah batas ukuran stiker WhatsApp.
func encodeSticker(rgba []byte, w, h int) ([]byte, error) {
	var best []byte
	for _, q := range []float32{85, 75, 65, 55, 45, 35} {
		b, err := encodeWebP(rgba, w, h, q)
		if err != nil {
			return nil, err
		}
		best = b
		if len(b) <= 100*1024 {
			break
		}
	}
	return best, nil
}

// ------------------------------------------------------------ decode

type webpChunk struct {
	id   string
	data []byte
}

func webpChunks(data []byte) ([]webpChunk, error) {
	if len(data) < 12 || string(data[0:4]) != "RIFF" || string(data[8:12]) != "WEBP" {
		return nil, errors.New("bukan file WebP")
	}
	var out []webpChunk
	p := 12
	for p+8 <= len(data) {
		id := string(data[p : p+4])
		n := int(binary.LittleEndian.Uint32(data[p+4 : p+8]))
		p += 8
		if n < 0 || p+n > len(data) {
			n = len(data) - p
		}
		out = append(out, webpChunk{id, data[p : p+n]})
		p += n + n&1
	}
	return out, nil
}

func u24(b []byte) int { return int(b[0]) | int(b[1])<<8 | int(b[2])<<16 }

func put24(b []byte, v int) { b[0], b[1], b[2] = byte(v), byte(v>>8), byte(v>>16) }

func riff(chunks ...webpChunk) []byte {
	var body bytes.Buffer
	body.WriteString("WEBP")
	for _, c := range chunks {
		body.WriteString(c.id)
		binary.Write(&body, binary.LittleEndian, uint32(len(c.data)))
		body.Write(c.data)
		if len(c.data)&1 == 1 {
			body.WriteByte(0)
		}
	}
	var out bytes.Buffer
	out.WriteString("RIFF")
	binary.Write(&out, binary.LittleEndian, uint32(body.Len()))
	out.Write(body.Bytes())
	return out.Bytes()
}

// webpInfo: ukuran kanvas dan apakah animasi.
func webpInfo(data []byte) (w, h int, animated bool) {
	ch, err := webpChunks(data)
	if err != nil || len(ch) == 0 {
		return 0, 0, false
	}
	if ch[0].id == "VP8X" && len(ch[0].data) >= 10 {
		d := ch[0].data
		return 1 + u24(d[4:7]), 1 + u24(d[7:10]), d[0]&0x02 != 0
	}
	if cfg, err := webp.DecodeConfig(bytes.NewReader(data)); err == nil {
		return cfg.Width, cfg.Height, false
	}
	return 0, 0, false
}

type animFrame struct {
	img   *image.RGBA // premultiplied, sudah diskalakan
	delay int         // ms
}

// decodeWebPFrames men-decode WebP (statis/animasi) dan menskalakan tiap frame
// agar muat dalam kotak size×size. maxFrames membatasi memori.
func decodeWebPFrames(data []byte, size, maxFrames int) ([]animFrame, error) {
	ch, err := webpChunks(data)
	if err != nil {
		return nil, err
	}
	animated := len(ch) > 0 && ch[0].id == "VP8X" && len(ch[0].data) >= 10 && ch[0].data[0]&0x02 != 0
	if !animated {
		img, err := webp.Decode(bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
		return []animFrame{{img: fitRGBA(toRGBA(img), size)}}, nil
	}
	cw, chh := 1+u24(ch[0].data[4:7]), 1+u24(ch[0].data[7:10])
	if cw > 2048 || chh > 2048 {
		return nil, errors.New("kanvas terlalu besar")
	}
	canvas := image.NewRGBA(image.Rect(0, 0, cw, chh))
	var frames []animFrame
	var prevRect image.Rectangle
	prevDispose := false
	for _, c := range ch {
		if c.id != "ANMF" || len(c.data) < 16 {
			continue
		}
		d := c.data
		fx, fy := 2*u24(d[0:3]), 2*u24(d[3:6])
		fw, fh := 1+u24(d[6:9]), 1+u24(d[9:12])
		dur := u24(d[12:15])
		noBlend := d[15]&0x02 != 0
		dispose := d[15]&0x01 != 0
		sub, _ := webpChunks(append([]byte("RIFF\x00\x00\x00\x00WEBP"), d[16:]...))
		var alph, bits *webpChunk
		for i := range sub {
			switch sub[i].id {
			case "ALPH":
				alph = &sub[i]
			case "VP8 ", "VP8L":
				bits = &sub[i]
			}
		}
		if bits == nil {
			continue
		}
		var single []byte
		if alph != nil && bits.id == "VP8 " {
			x := make([]byte, 10)
			x[0] = 0x10 // alpha
			put24(x[4:7], fw-1)
			put24(x[7:10], fh-1)
			single = riff(webpChunk{"VP8X", x}, *alph, *bits)
		} else {
			single = riff(*bits)
		}
		fimg, err := webp.Decode(bytes.NewReader(single))
		if err != nil {
			continue
		}
		if prevDispose {
			draw.Draw(canvas, prevRect, image.Transparent, image.Point{}, draw.Src)
		}
		r := image.Rect(fx, fy, fx+fw, fy+fh)
		op := draw.Over
		if noBlend {
			op = draw.Src
		}
		draw.Draw(canvas, r, fimg, fimg.Bounds().Min, op)
		prevRect, prevDispose = r, dispose
		if dur <= 10 {
			dur = 100
		}
		frames = append(frames, animFrame{img: fitRGBA(canvas, size), delay: dur})
		if len(frames) >= maxFrames {
			break
		}
	}
	if len(frames) == 0 {
		return nil, errors.New("animasi kosong")
	}
	return frames, nil
}

func toRGBA(img image.Image) *image.RGBA {
	if r, ok := img.(*image.RGBA); ok {
		return r
	}
	b := img.Bounds()
	out := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(out, out.Bounds(), img, b.Min, draw.Src)
	return out
}

// fitRGBA memperkecil (rata-rata area, premultiplied) agar muat dalam size×size.
// Selalu mengembalikan salinan baru.
func fitRGBA(src *image.RGBA, size int) *image.RGBA {
	b := src.Bounds()
	W, H := b.Dx(), b.Dy()
	tw, th := W, H
	if W >= H && W > size {
		tw, th = size, max(1, H*size/W)
	} else if H > W && H > size {
		tw, th = max(1, W*size/H), size
	}
	dst := image.NewRGBA(image.Rect(0, 0, tw, th))
	if tw == W && th == H {
		draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Src)
		return dst
	}
	for y := 0; y < th; y++ {
		y0, y1 := y*H/th, max((y+1)*H/th, y*H/th+1)
		for x := 0; x < tw; x++ {
			x0, x1 := x*W/tw, max((x+1)*W/tw, x*W/tw+1)
			var r, g, bl, a, n uint32
			for yy := y0; yy < y1; yy++ {
				o := src.PixOffset(b.Min.X+x0, b.Min.Y+yy)
				for xx := x0; xx < x1; xx++ {
					r += uint32(src.Pix[o])
					g += uint32(src.Pix[o+1])
					bl += uint32(src.Pix[o+2])
					a += uint32(src.Pix[o+3])
					n++
					o += 4
				}
			}
			o := dst.PixOffset(x, y)
			dst.Pix[o], dst.Pix[o+1], dst.Pix[o+2], dst.Pix[o+3] = uint8(r/n), uint8(g/n), uint8(bl/n), uint8(a/n)
		}
	}
	return dst
}
