// sni.c — implementasi StatusNotifierItem di atas GDBus (lihat sni.go).
#include "sni.h"
#include <string.h>
#include <stdlib.h>
#include <stdarg.h>
#include <stdio.h>

/* Log diagnostik lapangan: aktif HANYA bila env OBROL_SNI_LOG diset. */
static FILE *sni_logf = NULL;
static void snilog(const char *fmt, ...) {
	if (!sni_logf) return;
	va_list ap;
	fprintf(sni_logf, "[%lld] ", (long long)(g_get_monotonic_time() / 1000));
	va_start(ap, fmt);
	vfprintf(sni_logf, fmt, ap);
	va_end(ap);
	fputc('\n', sni_logf);
	fflush(sni_logf);
}

extern void goSniMethod(char *name, gint x, gint y);

static unsigned char *sni_pixels = NULL;
static gint sni_pw = 0, sni_ph = 0, sni_plen = 0;
static char *sni_title = NULL, *sni_tooltip = NULL, *sni_status = NULL;

void obrol_sni_set_pixels(const void *p, int len, int w, int h) {
	free(sni_pixels);
	sni_pixels = NULL;
	if (p && len > 0) {
		sni_pixels = (unsigned char *)g_memdup2(p, (gsize)len);
		sni_plen = len; sni_pw = w; sni_ph = h;
	}
}

void obrol_sni_set_strings(const char *title, const char *tooltip, const char *status) {
	g_free(sni_title); g_free(sni_tooltip); g_free(sni_status);
	sni_title = g_strdup(title ? title : "");
	sni_tooltip = g_strdup(tooltip ? tooltip : "");
	sni_status = g_strdup(status ? status : "Active");
}

static void sni_method_call(GDBusConnection *c, const gchar *sender, const gchar *path,
		const gchar *iface, const gchar *method, GVariant *params,
		GDBusMethodInvocation *inv, gpointer ud) {
	gint x = 0, y = 0;
	if (params && g_variant_n_children(params) >= 2) {
		g_variant_get_child(params, 0, "i", &x);
		g_variant_get_child(params, 1, "i", &y);
	}
	snilog("sni method=%s x=%d y=%d", method, x, y);
	goSniMethod((char *)method, x, y);
	g_dbus_method_invocation_return_value(inv, NULL);
}

static GVariant *sni_prop_get(GDBusConnection *c, const gchar *sender, const gchar *path,
		const gchar *iface, const gchar *prop, GError **err, gpointer ud) {
	snilog("sni prop get: %s", prop);
	if (g_strcmp0(prop, "Category") == 0) return g_variant_new_string("Communications");
	if (g_strcmp0(prop, "Id") == 0) return g_variant_new_string("obrol");
	if (g_strcmp0(prop, "Title") == 0) return g_variant_new_string(sni_title ? sni_title : "Obrol");
	if (g_strcmp0(prop, "Status") == 0) return g_variant_new_string(sni_status ? sni_status : "Active");
	if (g_strcmp0(prop, "ItemIsMenu") == 0) return g_variant_new_boolean(FALSE);
	if (g_strcmp0(prop, "MenuObjectPath") == 0) return g_variant_new_object_path("/MenuBar");
	if (g_strcmp0(prop, "IconName") == 0) return g_variant_new_string("");
	if (g_strcmp0(prop, "IconPixmap") == 0) {
		GVariantBuilder b;
		g_variant_builder_init(&b, G_VARIANT_TYPE("a(iiay)"));
		if (sni_pixels && sni_plen > 0) {
			GVariantBuilder px;
			g_variant_builder_init(&px, G_VARIANT_TYPE("(iiay)"));
			g_variant_builder_add_value(&px, g_variant_new_int32(sni_pw));
			g_variant_builder_add_value(&px, g_variant_new_int32(sni_ph));
			g_variant_builder_add_value(&px, g_variant_new_fixed_array(G_VARIANT_TYPE_BYTE, sni_pixels, (gsize)sni_plen, 1));
			g_variant_builder_add_value(&b, g_variant_builder_end(&px));
		}
		return g_variant_builder_end(&b);
	}
	if (g_strcmp0(prop, "ToolTip") == 0) {
		GVariantBuilder empty;
		g_variant_builder_init(&empty, G_VARIANT_TYPE("a(iiay)"));
		return g_variant_new("(sa(iiay)ss)", "", &empty,
			sni_title ? sni_title : "Obrol",
			sni_tooltip ? sni_tooltip : "");
	}
	snilog("sni prop UNKNOWN: %s", prop);
	g_set_error(err, G_DBUS_ERROR, G_DBUS_ERROR_UNKNOWN_PROPERTY, "unknown: %s", prop);
	return NULL;
}

static GDBusInterfaceVTable sni_vtable = { sni_method_call, sni_prop_get, NULL };

static const char sni_xml[] =
	"<node>"
	"<interface name='org.kde.StatusNotifierItem'>"
	"<property name='Category' type='s' access='read'/>"
	"<property name='Id' type='s' access='read'/>"
	"<property name='Title' type='s' access='read'/>"
	"<property name='Status' type='s' access='read'/>"
	"<property name='IconName' type='s' access='read'/>"
	"<property name='IconPixmap' type='a(iiay)' access='read'/>"
	"<property name='ToolTip' type='(sa(iiay)ss)' access='read'/>"
	"<property name='ItemIsMenu' type='b' access='read'/>"
	"<property name='MenuObjectPath' type='o' access='read'/>"
	"<method name='Activate'><arg type='i' direction='in'/><arg type='i' direction='in'/></method>"
	"<method name='SecondaryActivate'><arg type='i' direction='in'/><arg type='i' direction='in'/></method>"
	"<method name='ContextMenu'><arg type='i' direction='in'/><arg type='i' direction='in'/></method>"
	"<method name='Scroll'><arg type='i' direction='in'/><arg type='i' direction='in'/><arg type='s' direction='in'/></method>"
	"<signal name='NewIcon'/><signal name='NewToolTip'/><signal name='NewTitle'/><signal name='NewStatus'/>"
	"</interface></node>";

static GDBusConnection *sni_conn = NULL;
static guint sni_reg = 0;
static char *sni_bus_name = NULL;

static char *sni_last_watcher_owner = NULL;

static void sni_watcher_appeared(GDBusConnection *c, const gchar *name,
		const gchar *owner, gpointer ud) {
	if (!owner || !sni_conn || !sni_bus_name) return;
	if (sni_last_watcher_owner && g_strcmp0(owner, sni_last_watcher_owner) == 0)
		return; /* watcher yang sama sudah kita daftari */
	g_free(sni_last_watcher_owner);
	sni_last_watcher_owner = g_strdup(owner);
	snilog("watcher appeared: %s -> re-register", owner);
	GError *werr = NULL;
	GVariant *r = g_dbus_connection_call_sync(sni_conn,
		"org.kde.StatusNotifierWatcher", "/StatusNotifierWatcher",
		"org.kde.StatusNotifierWatcher", "RegisterStatusNotifierItem",
		g_variant_new("(s)", sni_bus_name), NULL, G_DBUS_CALL_FLAGS_NONE, 3000, NULL, &werr);
	if (r) g_variant_unref(r);
	else if (werr) g_error_free(werr);
}

/* ---------- com.canonical.dbusmenu: menu klik-kanan native host SNI ----------
 * Host SNI (Cinnamon/GNOME/KDE) merender menu klik-kanan dari antarmuka ini,
 * BUKAN dari ContextMenu() — tanpa ini host menyusun menu sendiri dan opsi
 * "Keluar" tidak pernah muncul (issue tray v1.15.0). Layout statis:
 *   1 = tampilkan jendela, 2 = pemisah, 3 = keluar.
 * Label didorong dari sisi Go (i18n) lewat obrol_sni_set_menu_labels. */

extern void goSniMenuEvent(int id);

static char *menu_show_label = NULL, *menu_quit_label = NULL;
static guint32 menu_rev = 1;
static long sni_menu_calls = 0;

long obrol_sni_menu_calls(void) { return sni_menu_calls; }

void obrol_sni_set_menu_labels(const char *show, const char *quit) {
	g_free(menu_show_label); g_free(menu_quit_label);
	menu_show_label = g_strdup(show ? show : "");
	menu_quit_label = g_strdup(quit ? quit : "");
	menu_rev++;
	if (sni_conn)
		g_dbus_connection_emit_signal(sni_conn, NULL, "/MenuBar",
			"com.canonical.dbusmenu", "LayoutUpdated",
			g_variant_new("(ui)", menu_rev, 0), NULL);
}

/* isi properti satu item menu (dipakai GetLayout & GetGroupProperties) */
static void menu_item_props(GVariantBuilder *props, gint id) {
	if (id == 2) {
		g_variant_builder_add(props, "{sv}", "type", g_variant_new_string("separator"));
		return;
	}
	const char *label = (id == 3) ? menu_quit_label : menu_show_label;
	g_variant_builder_add(props, "{sv}", "label", g_variant_new_string(label ? label : ""));
	g_variant_builder_add(props, "{sv}", "enabled", g_variant_new_boolean(TRUE));
}

static void menu_add_item(GVariantBuilder *kids, gint id) {
	GVariantBuilder props, node, ch;
	g_variant_builder_init(&props, G_VARIANT_TYPE("a{sv}"));
	menu_item_props(&props, id);
	g_variant_builder_init(&node, G_VARIANT_TYPE("(ia{sv}av)"));
	g_variant_builder_add_value(&node, g_variant_new_int32(id));
	g_variant_builder_add_value(&node, g_variant_builder_end(&props));
	g_variant_builder_init(&ch, G_VARIANT_TYPE("av"));
	g_variant_builder_add_value(&node, g_variant_builder_end(&ch));
	g_variant_builder_add_value(kids, g_variant_new_variant(g_variant_builder_end(&node)));
}

static void menu_method_call(GDBusConnection *c, const gchar *sender, const gchar *path,
		const gchar *iface, const gchar *method, GVariant *params,
		GDBusMethodInvocation *inv, gpointer ud) {
	sni_menu_calls++;
	snilog("menu method=%s", method);
	if (g_strcmp0(method, "GetLayout") == 0) {
		GVariantBuilder root, rprops, kids;
		g_variant_builder_init(&root, G_VARIANT_TYPE("(ia{sv}av)"));
		g_variant_builder_add_value(&root, g_variant_new_int32(0));
		g_variant_builder_init(&rprops, G_VARIANT_TYPE("a{sv}"));
		g_variant_builder_add(&rprops, "{sv}", "label",
			g_variant_new_string(sni_title ? sni_title : "Obrol"));
		g_variant_builder_add_value(&root, g_variant_builder_end(&rprops));
		g_variant_builder_init(&kids, G_VARIANT_TYPE("av"));
		menu_add_item(&kids, 1);
		menu_add_item(&kids, 2);
		menu_add_item(&kids, 3);
		g_variant_builder_add_value(&root, g_variant_builder_end(&kids));
		g_dbus_method_invocation_return_value(inv,
			g_variant_new("(u@(ia{sv}av))", menu_rev, g_variant_builder_end(&root)));
		return;
	}
	if (g_strcmp0(method, "GetGroup") == 0) {
		g_dbus_method_invocation_return_value(inv, g_variant_new("(a(ia{sv}av))", NULL));
		return;
	}
	if (g_strcmp0(method, "AboutToShow") == 0) {
		g_dbus_method_invocation_return_value(inv, g_variant_new("(b)", FALSE));
		return;
	}
	if (g_strcmp0(method, "AboutToShowGroup") == 0) {
		g_dbus_method_invocation_return_value(inv, g_variant_new("(auau)", NULL, NULL));
		return;
	}
	if (g_strcmp0(method, "GetGroupProperties") == 0) {
		/* (ai ids, as propertyNames) -> a(ia{sv}); wajib ada agar klien
		 * libdbusmenu (host Xfce dkk.) mau membangun anak menu. */
		GVariantBuilder out;
		g_variant_builder_init(&out, G_VARIANT_TYPE("a(ia{sv})"));
		GVariant *ids = g_variant_get_child_value(params, 0);
		gsize n = ids ? g_variant_n_children(ids) : 0;
		for (gsize k = 0; k < n; k++) {
			gint32 id = 0;
			g_variant_get_child(ids, k, "i", &id);
			GVariantBuilder props;
			g_variant_builder_init(&props, G_VARIANT_TYPE("a{sv}"));
			menu_item_props(&props, id);
			g_variant_builder_add_value(&out,
				g_variant_new("(i@a{sv})", id, g_variant_builder_end(&props)));
		}
		if (ids) g_variant_unref(ids);
		g_dbus_method_invocation_return_value(inv,
			g_variant_new("(a(ia{sv}))", &out));
		return;
	}
	if (g_strcmp0(method, "GetProperty") == 0) {
		gint32 id = 0;
		const gchar *name = NULL;
		g_variant_get(params, "(i&s)", &id, &name);
		GVariantBuilder props;
		g_variant_builder_init(&props, G_VARIANT_TYPE("a{sv}"));
		menu_item_props(&props, id);
		GVariant *dict = g_variant_builder_end(&props);
		GVariant *val = g_variant_lookup_value(dict, name, NULL);
		g_variant_unref(dict);
		if (!val) val = g_variant_new_string("");
		g_dbus_method_invocation_return_value(inv, g_variant_new("(v)", val));
		return;
	}
	if (g_strcmp0(method, "Event") == 0) {
		gint id = 0;
		const gchar *ev = NULL;
		if (params && g_variant_n_children(params) >= 2) {
			g_variant_get_child(params, 0, "i", &id);
			g_variant_get_child(params, 1, "&s", &ev);
		}
		if (ev && g_strcmp0(ev, "clicked") == 0)
			goSniMenuEvent(id);
		g_dbus_method_invocation_return_value(inv, NULL);
		return;
	}
	if (g_strcmp0(method, "EventGroup") == 0) {
		g_dbus_method_invocation_return_value(inv, g_variant_new("(au)", NULL));
		return;
	}
	g_dbus_method_invocation_return_error(inv, G_DBUS_ERROR,
		G_DBUS_ERROR_UNKNOWN_METHOD, "unknown: %s", method);
}

static GVariant *menu_prop_get(GDBusConnection *c, const gchar *sender, const gchar *path,
		const gchar *iface, const gchar *prop, GError **err, gpointer ud) {
	if (g_strcmp0(prop, "Version") == 0) return g_variant_new_uint32(3);
	if (g_strcmp0(prop, "TextDirection") == 0) return g_variant_new_string("ltr");
	if (g_strcmp0(prop, "Status") == 0) return g_variant_new_string("normal");
	if (g_strcmp0(prop, "IconThemePath") == 0) return g_variant_new("as", NULL);
	g_set_error(err, G_DBUS_ERROR, G_DBUS_ERROR_UNKNOWN_PROPERTY, "unknown: %s", prop);
	return NULL;
}

static GDBusInterfaceVTable menu_vtable = { menu_method_call, menu_prop_get, NULL };

static const char menu_xml[] =
	"<node>"
	"<interface name='com.canonical.dbusmenu'>"
	"<property name='Version' type='u' access='read'/>"
	"<property name='TextDirection' type='s' access='read'/>"
	"<property name='Status' type='s' access='read'/>"
	"<property name='IconThemePath' type='as' access='read'/>"
	"<method name='GetLayout'><arg type='i' direction='in'/><arg type='i' direction='in'/><arg type='as' direction='in'/><arg type='u' direction='out'/><arg type='(ia{sv}av)' direction='out'/></method>"
	"<method name='GetGroup'><arg type='ai' direction='in'/><arg type='a(ia{sv}av)' direction='out'/></method>"
	"<method name='GetGroupProperties'><arg type='ai' direction='in'/><arg type='as' direction='in'/><arg type='a(ia{sv})' direction='out'/></method>"
	"<method name='GetProperty'><arg type='i' direction='in'/><arg type='s' direction='in'/><arg type='v' direction='out'/></method>"
	"<method name='AboutToShow'><arg type='i' direction='in'/><arg type='b' direction='out'/></method>"
	"<method name='AboutToShowGroup'><arg type='ai' direction='in'/><arg type='au' direction='out'/><arg type='au' direction='out'/></method>"
	"<method name='Event'><arg type='i' direction='in'/><arg type='s' direction='in'/><arg type='v' direction='in'/><arg type='u' direction='in'/></method>"
	"<method name='EventGroup'><arg type='a(isvu)' direction='in'/><arg type='au' direction='out'/></method>"
	"<signal name='LayoutUpdated'><arg type='u'/><arg type='i'/></signal>"
	"</interface></node>";

int obrol_sni_start(const char *bus_name) {
	GError *err = NULL;
	{
		const char *lp = g_getenv("OBROL_SNI_LOG");
		if (lp) sni_logf = fopen(lp, "a");
	}
	snilog("=== sni start ===");
	sni_conn = g_bus_get_sync(G_BUS_TYPE_SESSION, NULL, &err);
	if (!sni_conn) { if (err) g_error_free(err); return 0; }
	GDBusNodeInfo *node = g_dbus_node_info_new_for_xml(sni_xml, &err);
	if (!node) { if (err) g_error_free(err); return 0; }
	sni_reg = g_dbus_connection_register_object(sni_conn, "/StatusNotifierItem",
		node->interfaces[0], &sni_vtable, NULL, NULL, &err);
	if (!sni_reg) { if (err) g_error_free(err); return 0; }
	{
		GDBusNodeInfo *mnode = g_dbus_node_info_new_for_xml(menu_xml, NULL);
		if (mnode)
			g_dbus_connection_register_object(sni_conn, "/MenuBar",
				mnode->interfaces[0], &menu_vtable, NULL, NULL, NULL);
	}
	/* v1.15.0: kepemilikan nama WAJIB selesai sebelum daftar ke watcher —
	 * own_name async meninggalkan jendela race saat item terdaftar runtime
	 * (panel sudah jalan): host membaca properti ke nama yang belum owned. */
	{
		GVariant *own = g_dbus_connection_call_sync(sni_conn,
			"org.freedesktop.DBus", "/org/freedesktop/DBus", "org.freedesktop.DBus",
			"RequestName", g_variant_new("(su)", bus_name, 0x4u),
			NULL, G_DBUS_CALL_FLAGS_NONE, 3000, NULL, &err);
		if (!own) { if (err) g_error_free(err); return 0; }
		guint32 res = 0;
		g_variant_get(own, "(u)", &res);
		g_variant_unref(own);
		snilog("RequestName result=%u", res);
		if (res != 1 && res != 4) return 0;
	}
	g_free(sni_bus_name);
	sni_bus_name = g_strdup(bus_name);
	/* watcher ada? tidak ada -> jatuh kembali ke GtkStatusIcon (XEmbed) */
	{
		GVariant *o = g_dbus_connection_call_sync(sni_conn,
			"org.freedesktop.DBus", "/org/freedesktop/DBus", "org.freedesktop.DBus",
			"GetNameOwner", g_variant_new("(s)", "org.kde.StatusNotifierWatcher"),
			NULL, G_DBUS_CALL_FLAGS_NONE, 2000, NULL, &err);
		if (!o) { snilog("no watcher -> fallback XEmbed"); if (err) g_error_free(err); return 0; }
		const gchar *owner = NULL;
		g_variant_get(o, "(s)", &owner);
		g_free(sni_last_watcher_owner);
		sni_last_watcher_owner = g_strdup(owner);
		g_variant_unref(o);
	}
	/* daftar ulang otomatis tiap watcher (panel) (re)start */
	g_bus_watch_name_on_connection(sni_conn, "org.kde.StatusNotifierWatcher",
		G_BUS_NAME_WATCHER_FLAGS_NONE, sni_watcher_appeared, NULL, NULL, NULL);
	GVariant *r = g_dbus_connection_call_sync(sni_conn,
		"org.kde.StatusNotifierWatcher", "/StatusNotifierWatcher",
		"org.kde.StatusNotifierWatcher", "RegisterStatusNotifierItem",
		g_variant_new("(s)", bus_name), NULL, G_DBUS_CALL_FLAGS_NONE, 3000, NULL, &err);
	if (!r) { snilog("register FAIL: %s", err ? err->message : "?"); if (err) g_error_free(err); return 0; }
	g_variant_unref(r);
	snilog("register OK");
	return 1;
}

void obrol_sni_emit(const char *signal) {
	if (!sni_conn) return;
	g_dbus_connection_emit_signal(sni_conn, NULL, "/StatusNotifierItem",
		"org.kde.StatusNotifierItem", signal, NULL, NULL);
}
