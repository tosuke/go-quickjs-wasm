/* Freestanding libc supplied by wasm2go, without WASI or QuickJS std/os. */
#include "pthread.h"
#define STB_SPRINTF_IMPLEMENTATION
#include "stb_sprintf.h"
#include "libc.c"

/* Rename internal calls too: realloc/calloc must not acquire the lock twice. */
#define malloc tlsf_malloc
#define free tlsf_free
#define realloc tlsf_realloc
#define memalign tlsf_memalign
#define calloc tlsf_calloc
#define aligned_alloc tlsf_aligned_alloc
#define reallocarray tlsf_reallocarray
#define malloc_usable_size tlsf_malloc_usable_size
#define malloc_good_size tlsf_malloc_good_size
#include "malloc_tlsf.c"
#undef malloc
#undef free
#undef realloc
#undef memalign
#undef calloc
#undef aligned_alloc
#undef reallocarray
#undef malloc_usable_size
#undef malloc_good_size

static pthread_mutex_t allocator_mutex = PTHREAD_MUTEX_INITIALIZER;

void * malloc(size_t size)
{
    pthread_mutex_lock(&allocator_mutex);
    void * result = tlsf_malloc(size);
    pthread_mutex_unlock(&allocator_mutex);
    return result;
}

void free(void *ptr)
{
    pthread_mutex_lock(&allocator_mutex);
    tlsf_free(ptr);
    pthread_mutex_unlock(&allocator_mutex);
}

void * realloc(void *ptr, size_t size)
{
    pthread_mutex_lock(&allocator_mutex);
    void * result = tlsf_realloc(ptr, size);
    pthread_mutex_unlock(&allocator_mutex);
    return result;
}

void * memalign(size_t align, size_t size)
{
    pthread_mutex_lock(&allocator_mutex);
    void * result = tlsf_memalign(align, size);
    pthread_mutex_unlock(&allocator_mutex);
    return result;
}

void * calloc(size_t count, size_t size)
{
    pthread_mutex_lock(&allocator_mutex);
    void * result = tlsf_calloc(count, size);
    pthread_mutex_unlock(&allocator_mutex);
    return result;
}

void * aligned_alloc(size_t align, size_t size)
{
    pthread_mutex_lock(&allocator_mutex);
    void * result = tlsf_aligned_alloc(align, size);
    pthread_mutex_unlock(&allocator_mutex);
    return result;
}

void * reallocarray(void *ptr, size_t count, size_t size)
{
    pthread_mutex_lock(&allocator_mutex);
    void * result = tlsf_reallocarray(ptr, count, size);
    pthread_mutex_unlock(&allocator_mutex);
    return result;
}

size_t malloc_usable_size(void *ptr)
{
    pthread_mutex_lock(&allocator_mutex);
    size_t result = tlsf_malloc_usable_size(ptr);
    pthread_mutex_unlock(&allocator_mutex);
    return result;
}

size_t malloc_good_size(size_t size)
{
    pthread_mutex_lock(&allocator_mutex);
    size_t result = tlsf_malloc_good_size(size);
    pthread_mutex_unlock(&allocator_mutex);
    return result;
}

/* Expose the C TLS address for embedders using errno. */
int *__errno_location(void) { return &errno; }
