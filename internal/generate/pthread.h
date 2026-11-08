/* The mutex subset used by QuickJS and the shared allocator. No OS threads. */
#pragma once
#include <stdint.h>
#include <stdatomic.h>

typedef _Atomic(uint32_t) pthread_mutex_t;
#define PTHREAD_MUTEX_INITIALIZER 0

static inline int pthread_mutex_lock(pthread_mutex_t *m)
{
    while (atomic_exchange_explicit(m, 1, memory_order_acquire))
        __builtin_wasm_memory_atomic_wait32((int32_t *)m, 1, -1);
    return 0;
}

static inline int pthread_mutex_unlock(pthread_mutex_t *m)
{
    atomic_store_explicit(m, 0, memory_order_release);
    __builtin_wasm_memory_atomic_notify((int32_t *)m, 1);
    return 0;
}
