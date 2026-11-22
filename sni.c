// sni.c — implementasi StatusNotifierItem di atas GDBus (lihat sni.go).
#include "sni.h"
#include <string.h>
#include <stdlib.h>

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
	goSniMethod((char *)method, x, y);
	g_dbus_method_invocation_return_value(inv, NULL);
}

static GVariant *sni_prop_get(GDBusConnection *c, const gchar *sender, const gchar *path,
		const gchar *iface, const gchar *prop, GError **err, gpointer ud) {
	if (g_strcmp0(prop, "Category") == 0) return g_variant_new_string("Communications");
	if (g_strcmp0(prop, "Id") == 0) return g_variant_new_string("obrol");
	if (g_strcmp0(prop, "Title") == 0) return g_variant_new_string(sni_title ? sni_title : "Obrol");
	if (g_strcmp0(prop, "Status") == 0) return g_variant_new_string(sni_status ? sni_status : "Active");
	if (g_strcmp0(prop, "ItemIsMenu") == 0) return g_variant_new_boolean(FALSE);
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
	"<method name='Activate'><arg type='i' direction='in'/><arg type='i' direction='in'/></method>"
	"<method name='SecondaryActivate'><arg type='i' direction='in'/><arg type='i' direction='in'/></method>"
	"<method name='ContextMenu'><arg type='i' direction='in'/><arg type='i' direction='in'/></method>"
	"<method name='Scroll'><arg type='i' direction='in'/><arg type='i' direction='in'/><arg type='s' direction='in'/></method>"
	"<signal name='NewIcon'/><signal name='NewToolTip'/><signal name='NewTitle'/><signal name='NewStatus'/>"
	"</interface></node>";

static GDBusConnection *sni_conn = NULL;
static guint sni_reg = 0;

int obrol_sni_start(const char *bus_name) {
	GError *err = NULL;
	sni_conn = g_bus_get_sync(G_BUS_TYPE_SESSION, NULL, &err);
	if (!sni_conn) { if (err) g_error_free(err); return 0; }
	GDBusNodeInfo *node = g_dbus_node_info_new_for_xml(sni_xml, &err);
	if (!node) { if (err) g_error_free(err); return 0; }
	sni_reg = g_dbus_connection_register_object(sni_conn, "/StatusNotifierItem",
		node->interfaces[0], &sni_vtable, NULL, NULL, &err);
	if (!sni_reg) { if (err) g_error_free(err); return 0; }
	g_bus_own_name_on_connection(sni_conn, bus_name,
		G_BUS_NAME_OWNER_FLAGS_DO_NOT_QUEUE, NULL, NULL, NULL, NULL);
	GVariant *r = g_dbus_connection_call_sync(sni_conn,
		"org.kde.StatusNotifierWatcher", "/StatusNotifierWatcher",
		"org.kde.StatusNotifierWatcher", "RegisterStatusNotifierItem",
		g_variant_new("(s)", bus_name), NULL, G_DBUS_CALL_FLAGS_NONE, 3000, NULL, &err);
	if (!r) { if (err) g_error_free(err); return 0; }
	g_variant_unref(r);
	return 1;
}

void obrol_sni_emit(const char *signal) {
	if (!sni_conn) return;
	g_dbus_connection_emit_signal(sni_conn, NULL, "/StatusNotifierItem",
		"org.kde.StatusNotifierItem", signal, NULL, NULL);
}
