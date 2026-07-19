#include <gst/gst.h>
#include <stdio.h>

int main(int argc, char *argv[]) {
    gst_init(&argc, &argv);

    guint major, minor, micro, nano;
    gst_version(&major, &minor, &micro, &nano);

    printf("GStreamer initialized successfully.\n");
    printf("Version: %u.%u.%u\n", major, minor, micro);

    return 0;
}