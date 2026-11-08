package quickjs_wasm_test

import (
	"math"
	"testing"
	"unsafe"

	qjs "github.com/tosuke/go-quickjs-wasm"
)

func goMemory(t *testing.T, maximumPages uint32) *qjs.SharedMemory {
	t.Helper()
	memory, err := qjs.NewMemory(maximumPages)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := memory.Close(); err != nil {
			t.Error(err)
		}
	})
	return memory
}

func TestGoMemoryMinimumStartup(t *testing.T) {
	// The initial 4 MiB must suffice for one runtime: do not require an extra
	// private worker stack or a memory.grow before using the initial heap.
	memory := goMemory(t, 64)
	m := worker(t, memory)
	_, ctx := threadContext(t, m)
	v := eval(t, m, ctx, "40+2")
	defer m.JS_FreeValue(ctx, v)
	if got := qjs.JS_VALUE_GET_INT(v); got != 42 {
		t.Fatalf("result=%d", got)
	}
	if len(memory.Bytes()) != 4*1024*1024 {
		t.Fatal("minimum startup grew memory")
	}
	if p := m.Xmalloc(4 * 1024 * 1024); p != 0 {
		m.Xfree(p)
		t.Fatal("allocation exceeded the configured maximum")
	}
	v = eval(t, m, ctx, "6*7")
	defer m.JS_FreeValue(ctx, v)
	if got := qjs.JS_VALUE_GET_INT(v); got != 42 {
		t.Fatal("failed allocation broke the runtime")
	}
}

func TestGoMemoryGrowthAndSAB(t *testing.T) {
	const maximumPages = 256 // 16 MiB, no mmap reservation.
	memory := goMemory(t, maximumPages)
	m := worker(t, memory)
	_, ctx := threadContext(t, m)
	before := memory.Bytes()
	base := unsafe.SliceData(before)
	v := eval(t, m, ctx, `globalThis.kept = new Int32Array(new SharedArrayBuffer(8));
        Atomics.store(kept,0,41); new Uint8Array(5*1024*1024).fill(123).at(-1)`)
	defer m.JS_FreeValue(ctx, v)
	if got := qjs.JS_VALUE_GET_INT(v); got != 123 {
		t.Fatalf("array tail=%d", got)
	}
	if len(memory.Bytes()) <= len(before) {
		t.Fatal("allocation did not grow Go-backed memory")
	}
	if unsafe.SliceData(memory.Bytes()) != base {
		t.Fatal("growth moved atomic/SAB addresses")
	}
	v = eval(t, m, ctx, "Atomics.add(kept,0,1);Atomics.load(kept,0)")
	defer m.JS_FreeValue(ctx, v)
	if got := qjs.JS_VALUE_GET_INT(v); got != 42 {
		t.Fatalf("retained SAB=%d", got)
	}
	old := memory.Grow(0, math.MaxInt64)
	if p := m.Xmalloc(32 * 1024 * 1024); p != 0 {
		m.Xfree(p)
		t.Fatal("Wasm allocator exceeded the Go memory maximum")
	}
	if got := memory.Grow(0, math.MaxInt64); got != old {
		t.Fatal("failed allocation changed memory size")
	}
	if got := memory.Grow(maximumPages-old, math.MaxInt64); got != old {
		t.Fatalf("Grow to maximum=%d", got)
	}
	for _, b := range memory.Bytes()[old*65536:] {
		if b != 0 {
			t.Fatal("new pages were not zero initialized")
		}
	}
	if memory.Grow(1, math.MaxInt64) != -1 {
		t.Fatal("Grow exceeded configured maximum")
	}
}

func expectPanic(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Error("expected panic")
		}
	}()
	f()
}

func TestGoMemoryWorkerLifecycle(t *testing.T) {
	memory := goMemory(t, 128)
	first := worker(t, memory)
	tls := first.X__errno_location()
	first.StoreI32(tls, 123)
	expectPanic(t, func() { qjs.New(memory) })
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	second := worker(t, memory)
	if second.X__errno_location() != tls || second.LoadI32(tls) != 0 {
		t.Fatal("bootstrap TLS was not reset on reopening")
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	if err := memory.Close(); err == nil {
		t.Fatal("closed memory while the replacement worker was alive")
	}
	if err := second.Close(); err != nil {
		t.Fatal(err)
	}
	if err := memory.Close(); err != nil {
		t.Fatal(err)
	}
	if memory.Grow(0, math.MaxInt64) != -1 || len(memory.Bytes()) != 0 {
		t.Fatal("closed Go memory is still accessible")
	}
	expectPanic(t, func() { qjs.New(memory) })
}

func TestConfiguredMemoryMaximum(t *testing.T) {
	// Test owner bounds before creating a Module: the allocator does not own
	// these pages, so reading the zero-filled tail cannot disturb heap metadata.
	memory, err := qjs.NewSharedMemory(128)
	if err != nil {
		t.Fatal(err)
	}
	defer memory.Close()
	initial := memory.Grow(0, math.MaxInt64)
	if got := memory.Grow(128-initial, math.MaxInt64); got != initial {
		t.Fatalf("Grow=%d", got)
	}
	if len(memory.Bytes()) != 128*65536 || memory.Grow(1, math.MaxInt64) != -1 {
		t.Fatal("mmap memory did not honor configured maximum")
	}
	for _, pages := range []uint32{0, 63, 65537, math.MaxUint32} {
		if m, err := qjs.NewMemory(pages); err == nil {
			m.Close()
			t.Fatalf("Go memory accepted invalid limit %d", pages)
		}
		if m, err := qjs.NewSharedMemory(pages); err == nil {
			m.Close()
			t.Fatalf("mmap memory accepted invalid limit %d", pages)
		}
	}
}
