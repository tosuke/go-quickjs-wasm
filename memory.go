package quickjs_wasm

import (
	"fmt"
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
	maximumPages         int64
	size                 atomic.Int64
	region               *memoryRegion
	mu                   sync.Mutex
	waiters              sync.Map
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

// NewMemory creates Go-backed memory for one active Module, initially allocating
// only the generated minimum size. Grow resizes the slice and may relocate it.
// No maximum-capacity reservation or per-instance limit is configured. Growth is
// bounded by the Wasm address range and host address space. No mmap is required.
func NewMemory() (*SharedMemory, error) {
	return newLinearMemory(memoryInitialPages, memoryMaxPages, false)
}

func newLinearMemory(initial, maximum int64, mapped bool) (*SharedMemory, error) {
	if initial < 0 || initial > maximum || maximum > memoryMaxPages || maximum <= 0 {
		return nil, fmt.Errorf("invalid Wasm memory limits: %d, %d", initial, maximum)
	}
	capacity := maximum * wasmPageSize
	if initial*wasmPageSize > int64(^uint(0)>>1) || mapped && capacity > int64(^uint(0)>>1) {
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
		region = &memoryRegion{data: make([]byte, int(initial*wasmPageSize))}
	}
	m := &SharedMemory{view: region.data, region: region, maximumPages: maximum}
	m.size.Store(initial * wasmPageSize)
	return m, nil
}

// Xmemory supplies the imported memory to wasm2go's New constructor.
func (memory *SharedMemory) Xmemory() Memory { return memory }

// Init is wasm2go's construction hook, called by New(memory). It initializes
// shared data and the allocator, then assigns this worker its C stack/TLS.
// Go-backed memory permits only one active worker. It panics on closed memory,
// a second Go-backed worker, or failure to allocate a shared worker's stack/TLS.
func (memory *SharedMemory) Init(instance any) {
	m := instance.(*Module)
	memory.modulesMu.Lock()
	defer memory.modulesMu.Unlock()
	if memory.region.data == nil {
		panic("quickjs_wasm: closed shared memory")
	}
	if !memory.region.mapped && memory.workers != 0 {
		panic("quickjs_wasm: Go memory allows only one active worker")
	}
	env := &moduleEnv{shared: memory}
	m._env = env
	// The construction hook runs before the Wasm start function. Initialize
	// shared data now so allocator and TLS setup can use it. The start function
	// will observe completed initialization when New resumes.
	m.___wasm_init_memory()
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
	env.stackAllocation = ptr
}

type moduleEnv struct {
	shared          *SharedMemory
	stackAllocation int32
	closed          bool
}

func (e *moduleEnv) Xmemory() Memory { return e.shared }

// Bytes returns the accessible memory. Mmap-backed addresses remain stable across
// growth. Go-backed views must be reacquired after any Grow or Module call that
// may allocate. Go-backed Bytes/Grow and Module calls must be serialized on one
// worker. Close invalidates all views; finish every access before calling it.
func (m *SharedMemory) Bytes() []byte {
	size := m.size.Load()
	return m.region.data[:size:size]
}

// Slice implements wasm2go's memory interface. The mmap-backed header spans the
// entire reservation and stays fixed across Grow; only the accessible prefix is
// enabled. Go-backed Grow updates the header, with all accesses serialized on its
// single worker. Callers must not mutate the header. Bytes returns the current
// accessible prefix for either backend.
func (m *SharedMemory) Slice() *[]byte     { return &m.view }
func (m *SharedMemory) Waiters() *sync.Map { return &m.waiters }

func (m *SharedMemory) Grow(delta, maximum int64) int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.region.data == nil {
		return -1
	}
	old := m.size.Load() / wasmPageSize
	maximum = min(maximum, m.maximumPages, int64(^uint(0)>>1)/wasmPageSize)
	if delta < 0 || maximum < old || delta > maximum-old {
		return -1
	}
	if delta == 0 {
		return old
	}
	next := (old + delta) * wasmPageSize
	if m.region.mapped {
		if err := m.region.enable(int(old*wasmPageSize), int(next)); err != nil {
			return -1
		}
	} else {
		// C pointers are Wasm offsets. Copy the entire prefix, including the
		// allocator state, C stack and TLS, and zero-initialize added pages.
		m.region.data = append(m.region.data, make([]byte, int(next)-len(m.region.data))...)
		m.view = m.region.data
	}
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
	return nil
}

// Close releases a shared worker's private C stack/TLS, or relinquishes a
// Go-backed Module's initial stack/TLS, after its runtimes have been freed.
// The memory owner remains usable. Calls are idempotent.
func (m *Module) Close() error {
	env := m._env.(*moduleEnv)
	memory := env.shared
	memory.modulesMu.Lock()
	defer memory.modulesMu.Unlock()
	if env.closed {
		return nil
	}
	// Free from the serialized bootstrap stack, not the stack being released.
	if env.stackAllocation != 0 {
		bootstrap := New(&moduleEnv{shared: memory})
		bootstrap.Xfree(env.stackAllocation)
	}
	env.stackAllocation = 0
	env.closed = true
	memory.workers--
	return nil
}
