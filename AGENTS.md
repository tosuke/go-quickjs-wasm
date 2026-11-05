# Repository Guidelines

## Purpose and Organization

This repository translates QuickJS from WebAssembly into Go and exposes its C API. Preserve QuickJS semantics while making the boundary between Go values, Wasm memory, and function-table entries explicit.

- Handwritten library code lives at the repository root; `quickjs.go` defines the API and `mem.go` provides memory access utilities.
- `internal/generate/` owns generation tools, C adapters, and the Docker pipeline. Its Go module is separate from the library module.
- Root tests use package `quickjs_wasm_test`; shared helpers live in `helpers_test.go`. Test-only C adapters belong in `internal/generate/shim_test.c`.
- Generated files are named `*_generated.go` or `quickjs_generated.dat`. Third-party notices live in `licenses/`.

## Build and Validation

Use the Go versions declared in the module files and the generation toolchain pinned in the Dockerfile.

- `go build ./...`: compile the library.
- `go test ./...`: run library tests and fuzz seeds.
- `go test -race ./...`: check for data races.
- `go -C internal build ./generate/structgen`: compile the generator; root-module checks do not enter the nested module.
- `go test -run '^$' -fuzz '^FuzzHeaderFloat64$' -fuzztime=30s -parallel=2 .`: example focused fuzz run; select the target relevant to the change.
- `docker buildx bake generate`: regenerate artifacts using the pinned toolchain. Requires Docker Buildx and dependency downloads.

Choose validation proportional to the change. Run focused checks while developing, then the relevant module's tests before finishing. Use fuzzing for behavior with large input spaces or representation-sensitive edge cases.

## API and ABI Principles

Preserve QuickJS naming and behavior, including exception handling, ownership, and resource lifetimes. Make changes to public signatures and return conventions deliberately.

Treat Wasm addresses, function-table indices, and tagged values as distinct concepts even when their Go representations coincide. Preserve their bits across conversions, and choose signed or unsigned arithmetic according to the operation's semantics.

Callbacks must match the exact Go function types expected by translated indirect calls. Equal bit widths do not imply compatible function types; validate the actual calling boundary.

Keep API intent and layout facts under separate authorities: Go definitions specify exposed fields, types, and documentation; the C compiler determines sizes, offsets, and alignment for the generation target. Fail generation when a definition cannot be matched or represented correctly.

Keep public C-memory representations opaque and expose typed access to their fields. Avoid coupling them to Go's native struct layout. Treat unions, bitfields, and other special representations according to their C semantics, with explicit handling where ordinary field addressing is insufficient.

Use shared memory access utilities in implementation code and tests. Document semantics, ownership, and non-obvious constraints; avoid comments that merely repeat signatures.

## Testing Principles

Retain tests only where inspection, builds, or generation are insufficient; omit implementation-mirroring and routine C API wrapper checks. Temporary experimental tests are exempt; review or remove them before finalizing. Prefer independent evidence, such as actual C accesses or calls.

Include numeric boundaries, exceptions, and ownership transitions where relevant. For memory writes, check that neighboring fields and unrelated bytes remain intact. Reuse existing helpers and release owned resources with their matching free functions.

Keep test adapters focused on active tests and remove them when their consumers disappear.

## Generated Code and Style

Change generation inputs rather than editing generated artifacts directly. Regenerate affected outputs, review their diffs, and include inputs and outputs together in commits.

Format handwritten Go with `gofmt` and use standard import grouping. Keep production code in `quickjs_wasm` and follow existing conventions for names and file organization.

## Commits and Pull Requests

Use short, lowercase imperative commit subjects and keep commits focused. PR descriptions should explain the problem, resulting behavior, and validation results. Describe public API or generation effects when relevant, and link related issues.
