package quickjs_wasm_test

import (
	"encoding/binary"
	"math"
	"testing"
	"unsafe"

	qjs "github.com/tosuke/go-quickjs-wasm"
)

func goMemory(t *testing.T) *qjs.SharedMemory {
	t.Helper()
	memory, err := qjs.NewMemory()
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

func TestGoMemoryDefaultGrowth(t *testing.T) {
	memory := goMemory(t)
	before := memory.Bytes()
	initial := len(before)
	if initial != 4*1024*1024 || cap(*memory.Slice()) != initial {
		t.Fatal("Go memory preallocated beyond the generated initial size")
	}
	before[0], before[initial-1] = 123, 45
	if got := memory.Grow(1, math.MaxInt64); got != int64(initial/65536) {
		t.Fatalf("Grow=%d", got)
	}
	after := memory.Bytes()
	if len(after) != initial+65536 || unsafe.SliceData(after) == unsafe.SliceData(before) {
		t.Fatal("Grow did not reallocate and extend the initial slice")
	}
	if after[0] != 123 || after[initial-1] != 45 {
		t.Fatal("relocation lost old data")
	}
	for _, b := range after[initial:] {
		if b != 0 {
			t.Fatal("new pages are not zero initialized")
		}
	}
	after[0] = 99
	if before[0] != 123 {
		t.Fatal("old view unexpectedly aliases relocated memory")
	}
}

func TestGoMemoryStoreAfterReentrantGrowth(t *testing.T) {
	memory := goMemory(t)
	m := worker(t, memory)
	p := m.Xmalloc(12)
	defer m.Xfree(p)
	m.StoreI32(p, 7)
	m.StoreI32(p+4, 111)
	m.StoreI32(p+8, 9)
	before := memory.Bytes()
	callback := func() int32 {
		if memory.Grow(1, math.MaxInt64) < 0 {
			t.Fatal("Grow failed")
		}
		return 777
	}
	table := m.X__indirect_function_table()
	index := int32(len(*table))
	*table = append(*table, callback)
	c := newTestModule(t, m)
	if got := c.Xtest_store_after_grow(p+4, index); got != 777 {
		t.Fatalf("C store/read after Grow=%d", got)
	}
	if m.LoadI32(p+4) != 777 || m.LoadI32(p) != 7 || m.LoadI32(p+8) != 9 {
		t.Fatal("store used stale memory or changed neighboring words")
	}
	if got := binary.LittleEndian.Uint32(before[uint32(p+4):uint32(p+8)]); got != 111 {
		t.Fatal("C wrote into the obsolete backing array")
	}
}

func TestGoMemoryMinimumStartup(t *testing.T) {
	// The initial 4 MiB must suffice for one runtime: do not require an extra
	// private worker stack or a memory.grow before using the initial heap.
	memory := goMemory(t)
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
	p := m.Xmalloc(4 * 1024 * 1024)
	if p == 0 {
		t.Fatal("allocation could not grow beyond the initial size")
	}
	m.Xfree(p)
	v = eval(t, m, ctx, "6*7")
	defer m.JS_FreeValue(ctx, v)
	if got := qjs.JS_VALUE_GET_INT(v); got != 42 {
		t.Fatal("growth broke the runtime")
	}
}

func TestGoMemoryGrowthAndSAB(t *testing.T) {
	memory := goMemory(t)
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
	if unsafe.SliceData(memory.Bytes()) == base {
		t.Fatal("allocation did not exercise relocation")
	}
	v = eval(t, m, ctx, "Atomics.add(kept,0,1);Atomics.load(kept,0)")
	defer m.JS_FreeValue(ctx, v)
	if got := qjs.JS_VALUE_GET_INT(v); got != 42 {
		t.Fatalf("retained SAB=%d", got)
	}
	// Larger allocations must also grow on demand and preserve existing SAB data.
	v = eval(t, m, ctx, "globalThis.large=new Uint8Array(20*1024*1024).fill(91);Atomics.add(kept,0,1);large.at(-1)")
	defer m.JS_FreeValue(ctx, v)
	if qjs.JS_VALUE_GET_INT(v) != 91 {
		t.Fatal("large allocation lost data after further growth")
	}
	v = eval(t, m, ctx, "Atomics.load(kept,0)")
	defer m.JS_FreeValue(ctx, v)
	if qjs.JS_VALUE_GET_INT(v) != 43 {
		t.Fatal("retained SAB was corrupted by further growth")
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
	memory := goMemory(t)
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
		if m, err := qjs.NewSharedMemory(pages); err == nil {
			m.Close()
			t.Fatalf("mmap memory accepted invalid limit %d", pages)
		}
	}
}
