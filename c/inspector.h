#ifndef INSPECTOR_H
#define INSPECTOR_H

/*
 * Ownership rule:
 * inspect_media() allocates its return string with malloc().
 * The caller must free it by calling inspect_media_free() —
 * never call free() directly on the result.
 *
 * Return contract:
 * Always returns a JSON object on success OR failure — the caller
 * should check for the presence of an "error" key to distinguish
 * the two. NULL is reserved for catastrophic failures only (e.g.
 * malloc itself failing), which should be treated as fatal.
 */
char *inspect_media(const char *filepath);
void  inspect_media_free(char *result);

#endif