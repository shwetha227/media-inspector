#include "inspector.h"

#include <gst/gst.h>
#include <gst/pbutils/pbutils.h>

#include <stdarg.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

/* Growable string buffer so we can build JSON without a JSON library. */
typedef struct {
    char *data;
    size_t len;
    size_t cap;
} strbuf;

static void sb_init(strbuf *sb) {
    sb->cap = 256;
    sb->len = 0;
    sb->data = malloc(sb->cap);
    sb->data[0] = '\0';
}

static void sb_append(strbuf *sb, const char *s) {
    size_t s_len = strlen(s);
    if (sb->len + s_len + 1 > sb->cap) {
        while (sb->len + s_len + 1 > sb->cap) {
            sb->cap *= 2;
        }
        sb->data = realloc(sb->data, sb->cap);
    }
    memcpy(sb->data + sb->len, s, s_len);
    sb->len += s_len;
    sb->data[sb->len] = '\0';
}

static void sb_appendf(strbuf *sb, const char *fmt, ...) {
    char tmp[256];
    va_list args;
    va_start(args, fmt);
    vsnprintf(tmp, sizeof(tmp), fmt, args);
    va_end(args);
    sb_append(sb, tmp);
}

char *inspect_media(const char *filepath) {
    if (filepath == NULL || filepath[0] == '\0') {
        return NULL;
    }

    if (!gst_is_initialized()) {
        gst_init(NULL, NULL);
    }

    GError *err = NULL;
    gchar *abs_path = filepath[0] == '/'
                           ? g_strdup(filepath)
                           : g_build_filename(g_get_current_dir(), filepath, NULL);
    gchar *uri = gst_filename_to_uri(abs_path, &err);
    g_free(abs_path);
    if (uri == NULL) {
        if (err) g_error_free(err);
        return NULL;
    }

    GstDiscoverer *discoverer = gst_discoverer_new(10 * GST_SECOND, &err);
    if (discoverer == NULL) {
        g_free(uri);
        if (err) g_error_free(err);
        return NULL;
    }
    if (err) { g_error_free(err); err = NULL; }

    GstDiscovererInfo *info = gst_discoverer_discover_uri(discoverer, uri, &err);
    g_free(uri);

    GstDiscovererResult result =
        info != NULL ? gst_discoverer_info_get_result(info) : GST_DISCOVERER_ERROR;

    if (info == NULL || result != GST_DISCOVERER_OK) {
        if (info != NULL) gst_discoverer_info_unref(info);
        if (err != NULL) g_error_free(err);
        g_object_unref(discoverer);
        return NULL;
    }

    strbuf sb;
    sb_init(&sb);

    GstClockTime duration = gst_discoverer_info_get_duration(info);
    double duration_seconds = (double)duration / (double)GST_SECOND;

    sb_append(&sb, "{");
    sb_appendf(&sb, "\"duration_seconds\":%.3f", duration_seconds);

    GstDiscovererStreamInfo *stream_info = gst_discoverer_info_get_stream_info(info);
    GstCaps *container_caps = stream_info ? gst_discoverer_stream_info_get_caps(stream_info) : NULL;

    sb_append(&sb, ",\"container\":\"");
    if (container_caps != NULL) {
        gchar *cstr = gst_caps_to_string(container_caps);
        sb_append(&sb, cstr);
        g_free(cstr);
        gst_caps_unref(container_caps);
    } else {
        sb_append(&sb, "unknown");
    }
    sb_append(&sb, "\"");

    if (stream_info != NULL) {
        gst_discoverer_stream_info_unref(stream_info);
    }

    sb_append(&sb, "}");

    gst_discoverer_info_unref(info);
    g_object_unref(discoverer);
    if (err != NULL) g_error_free(err);

    return sb.data;
}

void inspect_media_free(char *result) {
    if (result != NULL) {
        free(result);
    }
}