package main

// emoji.go — v1.9.8: pemilih emoji milik obrol sendiri, menggantikan pemilih
// bawaan GTK (GtkEmojiChooser) yang dipicu sinyal "insert-emoji".
//
// Alasan RAM: pemilih GTK membangun ribu-an widget emoji di TextView pada
// popup pertama dan mempertahankannya selamanya, plus memuat cache glyph
// font emoji berwarna secara masif saat digulir — kontributor besar lonjakan
// RSS pemakaian berat. Pemilih ini: kategori malas (sel dibangun saat tab
// diklik) dan semua sel dihancurkan saat popover tertutup.

import (
	"strings"

	"github.com/gotk3/gotk3/gtk"
)

type emojiCat struct {
	icon string
	list []string
}

var emojiCats = []emojiCat{
	{"😀", strings.Fields(`😀 😃 😄  😆 😅 🤣 😂 🙂 🙃 😉 😊 😇 🥰 😍  😘 😗  😙 🥲  😛 😜 🤪 😝 🤗  🤫 🤐  😑 😶 😏 😒 🙄 😬 😌 😔 😪 🤤 😴 😷 🤒 🤕 🤢 🤮 🥵 🥶 🥴 😵 🤠 🥳 🥸 😎 🤓 😕 😟 🙁 ☹️ 😮 😯 😲 😳 🥺 😦  😨 😰 😥 😢 😭 😱 😖 😣 😞 😓 😩 😫 🥱 😤 😡 🤬 😈 👿  💩 🤡 👹 👺 👻 👽 👾 🤖 😺 😸 😹 😻 😼 😽 🙀 😿 😾`)},
	{"👍", strings.Fields(`👋 🤚 🖐 ✋ 🖖 👌 🤌 🤏 ✌️ 🤞 🤟 🤘 🤙 👈 👉 👆 👇 ☝️ 👍 👎 ✊ 👊 🤛 🤜 👏 🙌 👐 🤲 🤝 🙏 ✍️ 💅 🤳 💪  👃 🧠 🫀 🫁 🦷 🦴 👀 👁 👅 👄 🙈 🙉 🙊`)},
	{"❤", strings.Fields(`❤️ 🧡 💛 💚 💙 💜 🖤 🤍 🤎 💔 ❣️ 💕 💞 💓 💗 💖 💘 💝 💯 💢 💥 💫 💦 💨 🕳 💬 💭 🗯 ♨️ ✨ ⭐ 🌟  ☄️ 💥 🌈 ☀️ 🌤 ⛅ 🌧 ⛈ ❄️ ☃️  🌊`)},
	{"🐱", strings.Fields(`🐶 🐱  🐹 🐰 🦊 🐻 🐼 🐨 🐯 🦁 🐮 🐷 🐸 🐵 🐔 🐧 🐦 🐤  🦅 🦉  🐺 🐗 🐴  🐝  🐛  🐌 🐞 🐜 🪰 🪲 🦟 🐢 🐍 🦎  🦑 🦐 🦞 🦀 🐡 🐠 🐟 🐬 🐳 🐋 🦈 🐊 🐅 🐆 🦓  🦧 🐘 🦛 🦏 🐪 🐫 🦒 🦬 🐂 🐄 🐎 🐖 🐏 🐑 🦙 🐐 🦌  🐩  🐈 🐓 🦃 🦚 🦜 🦢 🦩 🕊  🦝 🦨 🦡 🦫 🦦 🦥 🐁 🐀 🐿 🦔`)},
	{"🍜", strings.Fields(`🍏 🍎  🍊 🍋 🍌 🍉 🍇 🍓 🫐 🍈 🍒 🍑 🥭 🍍 🥥 🥝  🍆 🥑 🥦 🥬 🥒 🌶  🌽  🫒 🧄 🧅 🥔  🥐 🍞 🥖  🧀  🍳 🧈 🥞  🥓  🍗 🍖 🦴 🌭 🍔 🍟 🍕 🫓 🥪 🥙  🌮  🫔 🥗 🥘  🥫 🍝 🍜 🍲 🍛 🍣  🥟  🍤 🍙 🍚 🍘 🍥 🥠 🥮 🍢 🍡 🍧 🍨 🍦  🧁 🍰  🍮 🍭 🍬 🍫 🍿 🍩  🌰 🥜 🍯  🍼  ☕ 🍵  🥤  🍶 🍺 🍻 🥂  🥃 🍸 🍹 🧉 🍾 🧊 🥄  🍽`)},
	{"⚽", strings.Fields(`⚽  🏈 ⚾ 🥎  🏐 🏉 🥏  🪀 🏓 🏸 🏒 🏑 🥍 🏏 🪃 🥅 ⛳ 🪁 🏹 🎣 🤿  🥋  🛹  🛷 ⛸ 🥌 🎿 ⛷ 🏂  🏋  🤸  🤺 🤾 🏌  🧘 🏄 🏊 🤽  🧗  🥇 🥈 🥉 🏅 🎖 🏆 🎗 🎫 🎟 🎪 🤹 🎭 🩰 🎨 🎬  🎧 🎼 🎹 🥁 🪘 🎷 🎺 🪗 🎸 🎻 🎲 ♟ 🎯  🎮 🎰`)},
	{"💡", strings.Fields(`⌚ 📱  💻 ⌨️ 🖥 🖨  💽 💾  📀 📷 📸 📹 🎥 📞 ☎️ 📟  📺  🎙 🎚 🎛 🧭 ⏰ ⏱ ⏲  ⌛ 📡 🔋 🔌 💡 🔦 🕯 🪔 🧯 🛢 💸 💵 💴 💶 💷 🪙 💰 💳 💎 ⚖️  🧰 🔧 🔩 ⚙️ 🗜 ⛓ 🧲 🔫 💣 🧨 🪓 🔪 🗡 ️  🔭 🔬  📚 📖 🔖 🔗  🖊 🖋 ✒️ 🖌 🖍 📝`)},
	{"✅", strings.Fields(`✅ ❌ ❎ ➕ ➖ ➗ ➰ ➿ 〽️ ✳️ ✴️ ❇️ ©️ ®️ ™️  🔠 🔡 🔤 🅰️ 🆎 ️ 🆑 🆒 🆓 ℹ️ 🆕 🆖 Ⓜ️ 🆗 ️ 🆘 🆙 🆚 🈲 🈶 🈺 🈷 🔴 🟠  🟢 🔵  🟤  ⚪  🟧 🟨 🟩 🟦 🟪 🟫 ⬛ ⬜ ◼️ ◻️ ◾ ◽ ▪️ ▫️ 🔶 🔷 🔸 🔹 🔺 🔻 💠  🔳 🔲`)},
}

// showEmoji membuka pemilih emoji relatif terhadap widget pemicu.
func (a *App) showEmoji(rel gtk.IWidget) {
	if a.emojiPop == nil {
		pop, _ := gtk.PopoverNew(nil)
		pop.SetPosition(gtk.POS_TOP)
		v, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 6)
		addClass(v, "stickerpanel")
		cats, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 2)
		cats.SetHAlign(gtk.ALIGN_CENTER)
		for i := range emojiCats {
			idx := i
			b, _ := gtk.ButtonNew()
			b.SetRelief(gtk.RELIEF_NONE)
			addClass(b, "iconbtn")
			l, _ := gtk.LabelNew(emojiCats[i].icon)
			b.Add(l)
			b.SetTooltipText(T("Kategori emoji"))
			b.Connect("clicked", func() { a.fillEmojiCat(idx) })
			cats.PackStart(b, false, false, 0)
		}
		sw, _ := gtk.ScrolledWindowNew(nil, nil)
		sw.SetPolicy(gtk.POLICY_NEVER, gtk.POLICY_AUTOMATIC)
		sw.SetSizeRequest(332, 252)
		flow, _ := gtk.FlowBoxNew()
		flow.SetSelectionMode(gtk.SELECTION_NONE)
		flow.SetHomogeneous(true)
		flow.SetMinChildrenPerLine(8)
		flow.SetMaxChildrenPerLine(8)
		flow.SetRowSpacing(2)
		flow.SetColumnSpacing(2)
		sw.Add(flow)
		v.PackStart(cats, false, false, 0)
		v.PackStart(sw, true, true, 0)
		v.ShowAll()
		pop.Add(v)
		a.emojiPop = pop
		a.emojiFlow = flow
		pop.Connect("closed", func() {
			a.clearEmojiCells(true)
			a.emojiMore = nil
		})
	}
	a.emojiPop.SetRelativeTo(rel)
	a.fillEmojiCat(0)
	a.emojiPop.Popup()
}

// emojiPageSize: v1.10.0 — sel emoji dimuat per halaman. Menggulir kategori
// tidak lagi menyentuh ratusan glyph font emoji berwarna sekaligus (tiap
// glyph yang dirender memetakan sebagian file font warna ke RSS proses).
const emojiPageSize = 64

// fillEmojiCat membangun halaman pertama satu kategori (malas).
func (a *App) fillEmojiCat(idx int) {
	if a.emojiFlow == nil || idx < 0 || idx >= len(emojiCats) {
		return
	}
	a.clearEmojiCells(false)
	a.emojiOff = 0
	a.addEmojiPage(idx)
}

// addEmojiPage menambahkan satu halaman sel berikut (+ tombol «⋯» bila masih
// ada sisa), tanpa membuang halaman sebelumnya.
func (a *App) addEmojiPage(idx int) {
	list := emojiCats[idx].list
	if a.emojiOff >= len(list) {
		return
	}
	if a.emojiMore != nil {
		a.emojiFlow.Remove(a.emojiMore)
		a.emojiMore.Destroy()
		a.emojiMore = nil
	}
	end := a.emojiOff + emojiPageSize
	if end > len(list) {
		end = len(list)
	}
	for _, e := range list[a.emojiOff:end] {
		em := e
		b, _ := gtk.ButtonNew()
		b.SetRelief(gtk.RELIEF_NONE)
		addClass(b, "stickercell")
		l, _ := gtk.LabelNew(em)
		b.Add(l)
		b.Connect("clicked", func() {
			a.insertEmoji(em)
			a.emojiPop.Popdown()
		})
		a.emojiFlow.Insert(b, -1)
	}
	a.emojiOff = end
	if end < len(list) {
		mb, _ := gtk.ButtonNew()
		mb.SetRelief(gtk.RELIEF_NONE)
		addClass(mb, "stickercell")
		ml, _ := gtk.LabelNew("⋯")
		mb.Add(ml)
		mb.SetTooltipText(T("Muat lebih banyak emoji"))
		mb.Connect("clicked", func() { a.addEmojiPage(idx) })
		a.emojiFlow.Insert(mb, -1)
		a.emojiMore = &mb.Widget
	}
	a.emojiFlow.ShowAll()
}

// clearEmojiCells menghancurkan semua sel emoji supaya glyph & widget lepas.
func (a *App) clearEmojiCells(trim bool) {
	if a.emojiFlow == nil {
		return
	}
	cl := a.emojiFlow.GetChildren()
	if cl == nil {
		return
	}
	var dead []*gtk.Widget
	cl.Foreach(func(i interface{}) {
		if w, ok := i.(*gtk.Widget); ok {
			dead = append(dead, w)
		}
	})
	cl.Free()
	for _, w := range dead {
		a.emojiFlow.Remove(w)
		w.Destroy()
	}
	if trim {
		trimMem() // v1.9.8: kembalikan memori C sel emoji ke OS
	}
}

func (a *App) insertEmoji(e string) {
	buf, _ := a.input.GetBuffer()
	buf.InsertAtCursor(e)
	a.input.GrabFocus()
}
