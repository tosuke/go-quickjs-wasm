package quickjs_wasm

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
)

const wasmPageSize = 65536
const workerStackSize = 2 * 1024 * 1024

// SharedMemory owns the linear memory, allocator state and atomic waiter table
// shared by workers. Each worker needs its own Module and QuickJS runtime.
// A Module, runtime, context, or JSValue must not be used by concurrent workers.
// SharedArrayBuffer data can be shared between their runtimes.
type SharedMemory struct {
	view      []byte
	size      atomic.Int64
	region    *memoryRegion
	mu        sync.Mutex
	waiters   sync.Map
	cleanup   runtime.Cleanup
	modulesMu sync.Mutex
	workers   int
}

// NewSharedMemory reserves a fixed-address wasm32 memory with the generated
// initial and maximum sizes. Memory pages become accessible as workers grow it.
func NewSharedMemory() (*SharedMemory, error) {
	return newLinearMemory(memoryInitialPages, memoryMaxPages)
}

func newLinearMemory(initial, maximum int64) (*SharedMemory, error) {
	if initial < 0 || initial > maximum || maximum > 65536 || maximum <= 0 {
		return nil, fmt.Errorf("invalid Wasm memory limits: %d, %d", initial, maximum)
	}
	region, err := reserveMemory(maximum * wasmPageSize)
	if err != nil {
		return nil, err
	}
	if err := region.enable(0, int(initial*wasmPageSize)); err != nil {
		_ = region.close()
		return nil, err
	}
	m := &SharedMemory{view: region.data[:int(initial*wasmPageSize)], region: region}
	m.size.Store(initial * wasmPageSize)
	m.cleanup = runtime.AddCleanup(m, func(r *memoryRegion) { _ = r.close() }, region)
	return m, nil
}

// New creates a worker over caller-owned shared memory. It allocates a private
// C stack and initializes its own C TLS. It panics on a closed memory or OOM.
// Different Modules over the same memory can execute concurrently. Calls on one
// Module must be serialized, including calls using different JSRuntime objects.
func New(memory *SharedMemory) *Module {
	memory.modulesMu.Lock()
	defer memory.modulesMu.Unlock()
	if memory.region.data == nil {
		panic("quickjs_wasm: closed shared memory")
	}
	env := &moduleEnv{shared: memory}
	m := newModule(env)
	tlsSize := uint32(*m.X__tls_size())
	tlsAlign := max(uint32(*m.X__tls_align()), 16)
	if tlsAlign&(tlsAlign-1) != 0 {
		panic("quickjs_wasm: invalid TLS alignment")
	}
	ptr := m.Xmalloc(int32(workerStackSize + tlsSize + tlsAlign + 16))
	if ptr == 0 {
		panic("quickjs_wasm: cannot allocate worker stack and TLS")
	}
	tlsBase := (uint32(ptr) + tlsAlign - 1) & ^(tlsAlign - 1)
	stackBase := (tlsBase + tlsSize + 15) & ^uint32(15)
	*m.X__stack_pointer() = int32(stackBase + workerStackSize)
	m.X__wasm_init_tls(int32(tlsBase))
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
	return m.region.data[:m.size.Load()]
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

// Close releases the mapping after every worker has been closed. Values, contexts,
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
}

func (a *workerAllocation) close() {
	a.memory.modulesMu.Lock()
	defer a.memory.modulesMu.Unlock()
	if a.ptr == 0 {
		return
	}
	// Free from the serialized bootstrap stack, not the stack being released.
	bootstrap := newModule(&moduleEnv{shared: a.memory})
	bootstrap.Xfree(a.ptr)
	a.ptr = 0
	a.memory.workers--
}

// Close releases this worker's C stack/TLS after its runtimes have been freed.
// Other workers and the shared mapping remain usable. Calls are idempotent.
func (m *Module) Close() error {
	env := m._env.(*moduleEnv)
	env.allocation.close()
	env.cleanup.Stop()
	return nil
}
