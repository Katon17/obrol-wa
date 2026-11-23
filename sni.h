#ifndef OBROL_SNI_H
#define OBROL_SNI_H

#include <gio/gio.h>

void obrol_sni_set_pixels(const void *p, int len, int w, int h);
void obrol_sni_set_strings(const char *title, const char *tooltip, const char *status);
void obrol_sni_set_menu_labels(const char *show, const char *quit);
long obrol_sni_menu_calls(void);
int obrol_sni_start(const char *bus_name);
void obrol_sni_emit(const char *signal);

#endif
