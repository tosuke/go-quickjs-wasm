/* Freestanding libc supplied by wasm2go, without WASI or QuickJS std/os. */
#include "pthread.h"
#define STB_SPRINTF_IMPLEMENTATION
#include "stb_sprintf.h"
#include "libc.c"

/* Enable dlmalloc's locking boundaries with Wasm wait/notify mutexes. The
   supported custom-lock interface avoids spinning while another worker runs. */
#define USE_LOCKS 2
#define MLOCK_T pthread_mutex_t
#define INITIAL_LOCK(lk) (atomic_init((lk), 0), 0)
#define DESTROY_LOCK(lk) ((void)(lk), 0)
#define ACQUIRE_LOCK(lk) pthread_mutex_lock(lk)
#define RELEASE_LOCK(lk) pthread_mutex_unlock(lk)
#define TRY_LOCK(lk) (!pthread_mutex_trylock(lk))
static MLOCK_T malloc_global_mutex = PTHREAD_MUTEX_INITIALIZER;

#define malloc_getpagesize ((size_t)65536)
#define DEFAULT_GRANULARITY ((size_t)65536)
#include "malloc_sbrk.c"

/* New calls this once per memory, before publishing the first worker. */
__attribute__((export_name("qjs_init_allocator")))
void qjs_init_allocator(void) { init_allocator(); }

/* Expose the C TLS address for embedders using errno. */
int *__errno_location(void) { return &errno; }
