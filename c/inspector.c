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

static void append_stream_json(strbuf *sb, GstDiscovererStreamInfo *stream, gboolean *first) {
    GstCaps *caps = gst_discoverer_stream_info_get_caps(stream);
    if (caps == NULL) {
        return;
    }

    const GstStructure *s = gst_caps_get_structure(caps, 0);
    const gchar *media_type = gst_structure_get_name(s);

    if (!*first) {
        sb_append(sb, ",");
    }
    *first = FALSE;

    sb_append(sb, "{");
    sb_append(sb, "\"codec\":\"");
    sb_append(sb, media_type);
    sb_append(sb, "\"");

    if (GST_IS_DISCOVERER_VIDEO_INFO(stream)) {
        GstDiscovererVideoInfo *vinfo = GST_DISCOVERER_VIDEO_INFO(stream);
        guint width = gst_discoverer_video_info_get_width(vinfo);
        guint height = gst_discoverer_video_info_get_height(vinfo);
        guint fps_n = gst_discoverer_video_info_get_framerate_num(vinfo);
        guint fps_d = gst_discoverer_video_info_get_framerate_denom(vinfo);

        sb_append(sb, ",\"type\":\"video\"");
        sb_appendf(sb, ",\"width\":%u,\"height\":%u", width, height);
        if (fps_d > 0) {
            sb_appendf(sb, ",\"fps\":\"%u/%u\"", fps_n, fps_d);
        }
    } else if (GST_IS_DISCOVERER_AUDIO_INFO(stream)) {
        GstDiscovererAudioInfo *ainfo = GST_DISCOVERER_AUDIO_INFO(stream);
        guint channels = gst_discoverer_audio_info_get_channels(ainfo);
        guint sample_rate = gst_discoverer_audio_info_get_sample_rate(ainfo);

        sb_append(sb, ",\"type\":\"audio\"");
        sb_appendf(sb, ",\"channels\":%u,\"sample_rate\":%u", channels, sample_rate);
    } else {
        sb_append(sb, ",\"type\":\"unknown\"");
    }

    sb_append(sb, "}");
    gst_caps_unref(caps);
}

static void collect_streams(strbuf *sb, GstDiscovererStreamInfo *info, gboolean *first) {
    if (info == NULL) {
        return;
    }

    if (GST_IS_DISCOVERER_CONTAINER_INFO(info)) {
        GstDiscovererContainerInfo *cinfo = GST_DISCOVERER_CONTAINER_INFO(info);
        GList *children = gst_discoverer_container_info_get_streams(cinfo);
        for (GList *l = children; l != NULL; l = l->next) {
            collect_streams(sb, GST_DISCOVERER_STREAM_INFO(l->data), first);
        }
        gst_discoverer_stream_info_list_free(children);
    } else {
        append_stream_json(sb, info, first);
    }

    GstDiscovererStreamInfo *next = gst_discoverer_stream_info_get_next(info);
    if (next != NULL) {
        collect_streams(sb, next, first);
        gst_discoverer_stream_info_unref(next);
    }
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

    sb_append(&sb, ",\"streams\":[");
    gboolean first_stream = TRUE;
    collect_streams(&sb, stream_info, &first_stream);
    sb_append(&sb, "]");

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