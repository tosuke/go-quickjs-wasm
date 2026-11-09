package quickjs_wasm

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
)

const wasmPageSize = 65536
const workerStackSize = 2 * 1024 * 1024

// SharedMemory owns the linear memory, allocator and atomic waiter table.
// NewMemory configures it for one active Module backed by Go memory;
// NewSharedMemory uses mmap and permits concurrent workers. Each worker needs
// its own Module and QuickJS runtime. Calls on a Module must be serialized.
// SharedArrayBuffer data can be shared between runtimes using this owner.
type SharedMemory struct {
	view                 []byte
	size                 atomic.Int64
	region               *memoryRegion
	mu                   sync.Mutex
	waiters              sync.Map
	cleanup              runtime.Cleanup
	modulesMu            sync.Mutex
	workers              int
	bootstrapTLS         int32
	allocatorInitialized bool
}

// NewSharedMemory reserves fixed-address mmap memory for concurrent workers.
// The optional maximum is in 64 KiB Wasm pages; omitting it uses the generated
// maximum (normally 65536 pages / 4 GiB). The initial size is the generated minimum.
func NewSharedMemory(maximumPages ...uint32) (*SharedMemory, error) {
	if len(maximumPages) > 1 {
		return nil, fmt.Errorf("quickjs_wasm: expected at most one maximum page count")
	}
	maximum := int64(memoryMaxPages)
	if len(maximumPages) == 1 {
		maximum = int64(maximumPages[0])
	}
	return newLinearMemory(memoryInitialPages, maximum, true)
}

// NewMemory allocates ordinary Go memory for a single active Module. maximumPages
// is a required maximum in 64 KiB Wasm pages, at least the generated initial size.
// The backing array is allocated at its maximum capacity to keep atomic addresses
// and raw views stable. Only the accessible prefix grows. It does not use mmap.
func NewMemory(maximumPages uint32) (*SharedMemory, error) {
	return newLinearMemory(memoryInitialPages, int64(maximumPages), false)
}

func newLinearMemory(initial, maximum int64, mapped bool) (*SharedMemory, error) {
	if initial < 0 || initial > maximum || maximum > memoryMaxPages || maximum <= 0 {
		return nil, fmt.Errorf("invalid Wasm memory limits: %d, %d", initial, maximum)
	}
	capacity := maximum * wasmPageSize
	if capacity > int64(^uint(0)>>1) {
		return nil, fmt.Errorf("quickjs_wasm: memory capacity exceeds the host address space")
	}
	var region *memoryRegion
	if mapped {
		var err error
		region, err = reserveMemory(capacity)
		if err != nil {
			return nil, err
		}
		if err := region.enable(0, int(initial*wasmPageSize)); err != nil {
			_ = region.close()
			return nil, err
		}
	} else {
		region = &memoryRegion{data: make([]byte, int(capacity))}
	}
	m := &SharedMemory{view: region.data[:int(initial*wasmPageSize)], region: region}
	m.size.Store(initial * wasmPageSize)
	if mapped {
		m.cleanup = runtime.AddCleanup(m, func(r *memoryRegion) { _ = r.close() }, region)
	}
	return m, nil
}

// New creates a Module over caller-owned memory. Mmap workers get private C
// stacks/TLS and can execute concurrently on different Modules. Go-backed memory
// uses the initial C stack/TLS and permits one active Module. Calls on a Module
// must be serialized. It panics on a closed memory, a second Go-backed worker,
// or failure to allocate a shared worker's stack/TLS.
func New(memory *SharedMemory) *Module {
	memory.modulesMu.Lock()
	defer memory.modulesMu.Unlock()
	if memory.region.data == nil {
		panic("quickjs_wasm: closed shared memory")
	}
	if !memory.region.mapped && memory.workers != 0 {
		panic("quickjs_wasm: Go memory allows only one active worker")
	}
	env := &moduleEnv{shared: memory}
	m := newModule(env)
	if !memory.allocatorInitialized {
		// Data and dlmalloc's initial heap must be initialized before any worker
		// allocates. modulesMu also protects construction against live workers.
		m.Xqjs_init_allocator()
		memory.allocatorInitialized = true
	}
	var ptr int32
	if memory.region.mapped {
		tlsSize := uint32(*m.X__tls_size())
		tlsAlign := max(uint32(*m.X__tls_align()), 16)
		if tlsAlign&(tlsAlign-1) != 0 {
			panic("quickjs_wasm: invalid TLS alignment")
		}
		ptr = m.Xmalloc(int32(workerStackSize + tlsSize + tlsAlign + 16))
		if ptr == 0 {
			panic("quickjs_wasm: cannot allocate worker stack and TLS")
		}
		tlsBase := (uint32(ptr) + tlsAlign - 1) & ^(tlsAlign - 1)
		stackBase := (tlsBase + tlsSize + 15) & ^uint32(15)
		*m.X__stack_pointer() = int32(stackBase + workerStackSize)
		m.X__wasm_init_tls(int32(tlsBase))
	} else {
		// The generated data initializer sets the bootstrap TLS address on its
		// first invocation only. Reuse it after the previous Module is closed.
		if memory.bootstrapTLS == 0 {
			memory.bootstrapTLS = *m.X__tls_base()
		}
		m.X__wasm_init_tls(memory.bootstrapTLS)
	}
	memory.workers++
	env.allocation = &workerAllocation{memory: memory, ptr: ptr}
	env.cleanup = runtime.AddCleanup(env, func(a *workerAllocation) { a.close() }, env.allocation)
	return m
}

type moduleEnv struct {
	shared     *SharedMemory
	allocation *workerAllocation
	cleanup    runtime.Cleanup
}

func (e *moduleEnv) Xmemory() Memory { return e.shared }
func (m *Module) bytes() []byte      { return m.memImp.(*SharedMemory).Bytes() }

// Bytes returns an atomic snapshot of the accessible memory. The backing address
// is stable across growth. Keep this owner or a worker reachable while using raw
// views (runtime.KeepAlive after the last access if necessary). Close invalidates
// every view. Reacquire a view to access pages added after the snapshot.
func (m *SharedMemory) Bytes() []byte {
	defer runtime.KeepAlive(m)
	size := m.size.Load()
	return m.region.data[:size:size]
}

// Slice implements wasm2go's legacy memory interface. Do not read or mutate this
// slice header concurrently with Grow; use Bytes for concurrent memory views.
func (m *SharedMemory) Slice() *[]byte     { return &m.view }
func (m *SharedMemory) Waiters() *sync.Map { return &m.waiters }

func (m *SharedMemory) Grow(delta, maximum int64) int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.region.data == nil {
		return -1
	}
	old := m.size.Load() / wasmPageSize
	maximum = min(maximum, int64(len(m.region.data))/wasmPageSize)
	if delta < 0 || maximum < old || delta > maximum-old {
		return -1
	}
	if delta == 0 {
		return old
	}
	next := (old + delta) * wasmPageSize
	if err := m.region.enable(int(old*wasmPageSize), int(next)); err != nil {
		return -1
	}
	m.view = m.region.data[:next]
	m.size.Store(next)
	return old
}

// Close releases the backing memory after every worker has been closed. Values, contexts,
// and runtimes must be freed before closing their worker. No raw view may remain
// in use during Close. Closing an owner with live workers returns an error.
func (m *SharedMemory) Close() error {
	m.modulesMu.Lock()
	defer m.modulesMu.Unlock()
	if m.workers != 0 {
		return fmt.Errorf("quickjs_wasm: shared memory has %d live workers", m.workers)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.region.close(); err != nil {
		return err
	}
	m.view = nil
	m.size.Store(0)
	m.cleanup.Stop()
	return nil
}

type workerAllocation struct {
	memory *SharedMemory
	ptr    int32
	closed bool
}

func (a *workerAllocation) close() {
	a.memory.modulesMu.Lock()
	defer a.memory.modulesMu.Unlock()
	if a.closed {
		return
	}
	// Free from the serialized bootstrap stack, not the stack being released.
	if a.ptr != 0 {
		bootstrap := newModule(&moduleEnv{shared: a.memory})
		bootstrap.Xfree(a.ptr)
	}
	a.ptr = 0
	a.closed = true
	a.memory.workers--
}

// Close releases a shared worker's private C stack/TLS, or relinquishes a
// Go-backed Module's initial stack/TLS, after its runtimes have been freed.
// The memory owner remains usable. Calls are idempotent.
func (m *Module) Close() error {
	env := m._env.(*moduleEnv)
	env.allocation.close()
	env.cleanup.Stop()
	return nil
}
