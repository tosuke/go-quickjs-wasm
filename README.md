# go-quickjs-wasm

[QuickJS](https://bellard.org/quickjs) compiled as WASM and translated to Go by [ncruces/wasm2go](https://github.com/ncruces/wasm2go).

## Single-thread startup

Use `NewMemory()` for a single active Module backed by a resizable Go slice.
Only the generated initial size (currently 64 Wasm pages / 4 MiB) is allocated.
Grow extends the slice on demand and reallocates/copies it when necessary.

```go
memory, err := quickjs_wasm.NewMemory()
if err != nil {
    panic(err)
}
defer memory.Close()

module := quickjs_wasm.New(memory)
defer module.Close()
// Create the QuickJS runtime and context on module; free them before module.Close.
```

There is no per-instance maximum setting or maximum-capacity reservation for the
Go backend. It grows as needed within the generated Wasm address range (normally
4 GiB) and host address space. Grow uses 64 KiB Wasm pages.

The Go backend requires no mmap and uses the initial C stack/TLS and initial heap.
It can start a runtime using the initial 64 pages (4 MiB), without an additional
worker stack allocation. Creating a second active Module on the same
Go memory panics. After closing that Module, a replacement can reuse the memory.
All Go-backed accesses, including Bytes/Grow and Module calls, must be serialized
on the same worker. Raw Go-backed views must be reacquired after Grow or any
Module call that may allocate; Wasm pointer offsets remain valid across relocation.

## Shared-memory workers

Create the memory explicitly, then create one `Module` per worker. Workers sharing
that memory can execute concurrently. Each has its own C stack, C TLS, function
table, and QuickJS runtime. Calls on a single Module must be serialized. QuickJS
objects, contexts, and runtimes must remain with their owning worker; only SAB
backing stores are shared between runtimes.

```go
memory, err := quickjs_wasm.NewSharedMemory()
if err != nil {
    panic(err)
}
defer memory.Close()

first := quickjs_wasm.New(memory)
defer first.Close()
second := quickjs_wasm.New(memory)
defer second.Close()
// Use first and second from separate goroutines, with one runtime per worker.
```

`New(memory)` on mmap-backed memory allocates a private 2 MiB C stack and
initializes C TLS for each worker. The shared dlmalloc allocator uses its built-in
locking boundaries with Wasm atomic wait/notify mutexes (`USE_LOCKS=2`). QuickJS
class-ID allocation uses the same mutex primitive. Module initialization preserves
the shared heap when subsequent workers are created. The mmap slice header stays
fixed at the maximum reserved size; Grow enables additional pages and updates the
logical page count without changing that header.

`NewSharedMemory(maximumPages)` accepts a smaller maximum for mmap-backed workers;
omitting the argument retains the generated maximum of 65536 pages (4 GiB).
The mmap backend currently supports Linux and macOS when the selected capacity
fits the host's address space. It initially exposes 4 MiB and enables additional
pages on demand without moving the backing memory. The virtual reservation does
not allocate its maximum size in physical memory. Allow room for each worker's
2 MiB private stack as well as the runtimes and their allocations. Other platforms
return an error from `NewSharedMemory()`; use `NewMemory` for single-thread startup.

## SharedArrayBuffer ownership

SharedArrayBuffer and Atomics are enabled. Before creating or reading SABs that
will be transferred between workers, call
`module.JS_SetDefaultSharedArrayBufferFunctions(rt)` on each participating runtime.
The default ownership policy uses an atomic reference count on the backing store,
so the creating runtime can be destroyed while other runtimes still use it.
Transfer SABs using `JS_WriteObject2` with `JS_WRITE_OBJ_SAB` and `JS_ReadObject`
with `JS_READ_OBJ_SAB`. The serialized data contains memory addresses and is only
valid within the same SharedMemory. Keep the source SAB alive until all receivers
have read it; free the serialized buffer and SAB table with `JS_free` afterwards.

Install this ownership policy before any SAB use in that runtime. For externally
supplied SAB storage, use `JS_SetSharedArrayBufferFunctions` with an ownership
policy matching that storage instead. The default policy must not be used with
arbitrary external pointers.

`JS_SetCanBlock(rt, true)` enables blocking `Atomics.wait` in that runtime;
blocking is disabled by default. Other workers can continue evaluating JavaScript
and call `Atomics.notify` while it waits.

## Lifetime and memory access

Free values, contexts, and runtimes with their matching QuickJS functions before
calling `Module.Close()`. On the mmap backend that releases the worker's private
C stack and TLS; other workers remain usable. On the Go backend it relinquishes
the initial C stack/TLS for reuse by the next Module. Then call
`SharedMemory.Close()` to release the backing memory.
Closing a memory with live workers returns an error. Go cleanups also reclaim
unreachable worker stacks and mappings. Explicit Close returns worker allocations
and unmaps mmap-backed memory promptly; Go arrays are reclaimed by Go garbage
collection once their views become unreachable.

`SharedMemory.Bytes()` returns a snapshot of the currently accessible memory.
Mmap-backed views remain valid across growth; obtain a new snapshot to access
added pages. Go-backed views may become obsolete when Grow reallocates the slice;
reacquire them after calls that may allocate. Keep the owner or a Module reachable
while using raw views; call
`runtime.KeepAlive` after their last use if necessary. Closing the memory
invalidates every view and requires that no raw access remains in progress.
`Xmemory().Slice()` exposes the full reservation for mmap-backed memory and its
header remains fixed across Grow. Use `Bytes()` for a snapshot of the accessible
prefix. Go-backed Grow updates its slice header, so Go-backed access must remain
on the single worker. Callers must not modify either backend's slice header.
Synchronize conflicting accesses to shared bytes with the corresponding atomic
operations.

## License

This project is licensed under the [MIT License](LICENSE).
Third-party license notices are in [licenses/](licenses/).
