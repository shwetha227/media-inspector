#include "inspector.h"
#include <stdio.h>

int main(int argc, char **argv) {
    if (argc != 2) {
        fprintf(stderr, "usage: %s <media file>\n", argv[0]);
        return 2;
    }
    char *json = inspect_media(argv[1]);
    if (json == NULL) {
        fprintf(stderr, "error: could not inspect \"%s\"\n", argv[1]);
        return 1;
    }
    printf("%s\n", json);
    inspect_media_free(json);
    return 0;
}
