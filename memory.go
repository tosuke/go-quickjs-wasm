package quickjs_wasm

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"
)

const wasmPageSize = 65536

// New creates an isolated module with a fixed-address linear memory. On supported
// Unix systems it reserves the Wasm address space without allocating its maximum
// size in physical memory. It panics if the reservation cannot be made.
//
// Module API calls must be serialized. Call Close after freeing all QuickJS
// contexts and runtimes to release the reservation promptly.
func New() *Module {
	mem, err := newLinearMemory(memoryInitialPages, memoryMaxPages)
	if err != nil {
		panic(fmt.Errorf("quickjs_wasm: %w", err))
	}
	return newModule(mem)
}

// Close releases the module's linear memory. All contexts, runtimes, and values
// must be freed first, and no module calls or memory accesses may be in progress.
// Memory slices and pointers become invalid after Close. Repeated calls are safe.
// A cleanup also releases the mapping when an unclosed module becomes unreachable.
func (m *Module) Close() error {
	return m.memImp.(*linearMemory).close()
}

type linearMemory struct {
	view    []byte
	region  *memoryRegion
	mu      sync.Mutex
	waiters sync.Map
	cleanup runtime.Cleanup
}

func newLinearMemory(initial, maximum int64) (*linearMemory, error) {
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
	m := &linearMemory{view: region.data[:int(initial*wasmPageSize)], region: region}
	m.cleanup = runtime.AddCleanup(m, func(r *memoryRegion) { _ = r.close() }, region)
	return m, nil
}

func (m *linearMemory) Xmemory() Memory    { return m }
func (m *linearMemory) Slice() *[]byte     { return &m.view }
func (m *linearMemory) Waiters() *sync.Map { return &m.waiters }

// length addresses the slice length word, matching wasm2go's shared-memory ABI.
// The data pointer and capacity never change; publish length only after the new
// pages are accessible. Grow(0) is also used for Wasm memory.size.
func (m *linearMemory) length() *atomic.Uintptr {
	type header struct {
		data unsafe.Pointer
		len  atomic.Uintptr
		cap  uintptr
	}
	return &(*header)(unsafe.Pointer(&m.view)).len
}

func (m *linearMemory) Grow(delta, maximum int64) int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.region.data == nil {
		return -1
	}
	old := int64(m.length().Load()) / wasmPageSize
	maximum = min(maximum, int64(len(m.region.data))/wasmPageSize)
	if delta < 0 || maximum < old || delta > maximum-old {
		return -1
	}
	if delta == 0 {
		return old
	}
	next := old + delta
	if err := m.region.enable(int(old*wasmPageSize), int(next*wasmPageSize)); err != nil {
		return -1
	}
	m.length().Store(uintptr(next * wasmPageSize))
	return old
}

func (m *linearMemory) close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.region.close(); err != nil {
		return err
	}
	m.length().Store(0)
	m.cleanup.Stop()
	return nil
}
