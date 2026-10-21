#include <stdint.h>
void *nil_entry_start(uintptr_t callback, int32_t *status);
int32_t nil_entry_configure(void *entry, int enabled, uint32_t key, uint32_t modifiers);
void nil_entry_close(void *entry);
