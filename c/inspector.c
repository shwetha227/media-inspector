#include "inspector.h"
#include <stdlib.h>
#include <string.h>

char *inspect_media(const char *filepath) {
    if (filepath == NULL || filepath[0] == '\0') {
        return NULL;
    }

    const char *placeholder = "{\"status\":\"stub - not yet implemented\"}";
    char *result = malloc(strlen(placeholder) + 1);
    if (result == NULL) {
        return NULL;
    }

    strcpy(result, placeholder);
    return result;
}

void inspect_media_free(char *result) {
    if (result != NULL) {
        free(result);
    }
}