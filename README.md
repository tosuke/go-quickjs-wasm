# go-quickjs-wasm

[QuickJS](https://bellard.org/quickjs) compiled as WASM and translated to Go by [ncruces/wasm2go](https://github.com/ncruces/wasm2go).

`New()` imports an mmap-backed shared linear memory on 64-bit Linux and macOS.
Each module reserves 4 GiB of virtual address space, initially exposes 4 MiB,
and enables additional pages on demand.
Growing memory preserves existing addresses and does not copy the heap. The
reservation does not allocate 4 GiB of physical memory. `New()` panics if the
platform is unsupported or memory cannot be reserved.

SharedArrayBuffer and Atomics are enabled, including Wasm-backed wait/notify.
`JS_SetCanBlock(rt, true)` enables `Atomics.wait`; blocking is disabled by default.
Module API calls must be serialized; enabling shared memory does not make a
QuickJS runtime safe to call concurrently. The module's linear memory remains
isolated from other modules.

Free all QuickJS values, contexts, and runtimes before calling `Module.Close()`
to release the mapping. Existing host memory views remain valid across growth,
but become invalid after Close. Do not resize or replace the slice returned by
`Xmemory().Slice()`. Unclosed mappings are also released by a Go cleanup once
their memory owner becomes unreachable. Keep the module or its Memory owner
reachable while using raw host views; call `runtime.KeepAlive` after the last
access if necessary.

## License

This project is licensed under the [MIT License](LICENSE).
Third-party license notices are in [licenses/](licenses/).
