#ifndef INSPECTOR_H
#define INSPECTOR_H

/*
 * Ownership rule:
 * inspect_media() allocates its return string with malloc().
 * The caller must free it by calling inspect_media_free() —
 * never call free() directly on the result.
 * Returns NULL on failure.
 */
char *inspect_media(const char *filepath);
void  inspect_media_free(char *result);

#endif