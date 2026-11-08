# go-quickjs-wasm

[QuickJS](https://bellard.org/quickjs) compiled as WASM and translated to Go by [ncruces/wasm2go](https://github.com/ncruces/wasm2go).

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

`New(memory)` allocates a private 2 MiB C stack and initializes C TLS for each
worker. The shared TLSF allocator and QuickJS class-ID allocation use Wasm atomic
mutexes. Module initialization preserves the shared heap when subsequent workers
are created. Translated memory accesses obtain atomic size snapshots so another
worker can grow memory while they execute.

The mmap backend currently supports 64-bit Linux and macOS. Each memory reserves
4 GiB of virtual address space, initially exposes 4 MiB, and enables additional
pages on demand without moving the backing memory. The reservation does not
allocate 4 GiB of physical memory. Other platforms return an error from
`NewSharedMemory()`.

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
calling `Module.Close()`. That releases the worker's C stack and TLS; other
workers remain usable. Then call `SharedMemory.Close()` to release the mapping.
Closing a memory with live workers returns an error. Go cleanups also reclaim
unreachable worker stacks and mappings, but explicit Close releases them promptly.

`SharedMemory.Bytes()` returns a snapshot of the currently accessible memory.
Existing views remain valid across growth; obtain a new snapshot to access added
pages. Keep the owner or a Module reachable while using raw views; call
`runtime.KeepAlive` after their last use if necessary. Closing the memory
invalidates every view and requires that no raw access remains in progress.
The legacy `Xmemory().Slice()` header must not be read or changed concurrently
with Grow; use `Bytes()` for concurrent access. Synchronize conflicting accesses
to shared bytes with the corresponding atomic operations.

## License

This project is licensed under the [MIT License](LICENSE).
Third-party license notices are in [licenses/](licenses/).
