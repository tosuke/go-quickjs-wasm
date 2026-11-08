package quickjs_wasm_test

import (
	"testing"

	qjs "github.com/tosuke/go-quickjs-wasm"
)

type testModuleEnv struct {
	*qjs.Module
	memoryBase int32
	tableBase  int32
}

func (e *testModuleEnv) X__memory_base() *int32 { return &e.memoryBase }
func (e *testModuleEnv) X__table_base() *int32  { return &e.tableBase }
func (e *testModuleEnv) Xmemory() Memory        { return e.Module.Xmemory() }

func (m *Module) bytes() []byte { return m.memImp.(*qjs.SharedMemory).Bytes() }

func newTestModule(t testing.TB, main *qjs.Module) *Module {
	t.Helper()
	if memorySize, _, tableSize, _ := DylinkInfo(); memorySize != 0 || tableSize != 0 {
		t.Fatalf("test shim requires memory size %d and table size %d", memorySize, tableSize)
	}
	return New(&testModuleEnv{Module: main})
}

func sharedMemory(t testing.TB) *qjs.SharedMemory {
	t.Helper()
	shared, err := qjs.NewSharedMemory()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := shared.Close(); err != nil {
			t.Error(err)
		}
	})
	return shared
}

func worker(t testing.TB, shared *qjs.SharedMemory) *qjs.Module {
	t.Helper()
	m := qjs.New(shared)
	t.Cleanup(func() {
		if err := m.Close(); err != nil {
			t.Error(err)
		}
	})
	return m
}

func module(t testing.TB) *qjs.Module {
	t.Helper()
	return worker(t, sharedMemory(t))
}

func context(t *testing.T) (*qjs.Module, qjs.JSRuntimePtr, qjs.JSContextPtr) {
	t.Helper()
	m := module(t)
	rt := m.JS_NewRuntime()
	if rt == 0 {
		t.Fatal("JS_NewRuntime returned NULL")
	}
	t.Cleanup(func() { m.JS_FreeRuntime(rt) })
	ctx := m.JS_NewContext(rt)
	if ctx == 0 {
		t.Fatal("JS_NewContext returned NULL")
	}
	t.Cleanup(func() { m.JS_FreeContext(ctx) })
	return m, rt, ctx
}

func cstring(t *testing.T, m *qjs.Module, ctx qjs.JSContextPtr, s string) qjs.Ptr[byte] {
	t.Helper()
	p := m.JS_malloc(ctx, uint32(len(s)+1))
	if p == 0 {
		t.Fatal("malloc returned NULL")
	}
	m.StoreString(p, s)
	t.Cleanup(func() { m.JS_free(ctx, p) })
	return p
}

func valueString(t *testing.T, m *qjs.Module, ctx qjs.JSContextPtr, v qjs.JSValue) string {
	t.Helper()
	p, n := m.JS_ToCStringLen(ctx, v)
	if p == 0 {
		t.Fatal("JS_ToCString returned NULL")
	}
	defer m.JS_FreeCString(ctx, p)
	b := make([]byte, int(n))
	m.LoadBytes(b, p, len(b))
	return string(b)
}

func eval(t *testing.T, m *qjs.Module, ctx qjs.JSContextPtr, source string) qjs.JSValue {
	t.Helper()
	v := m.JS_Eval(ctx, cstring(t, m, ctx, source), int32(len(source)), cstring(t, m, ctx, "test.js"), 0)
	if qjs.JS_IsException(v) {
		exc := m.JS_GetException(ctx)
		defer m.JS_FreeValue(ctx, exc)
		t.Fatalf("JS_Eval: %s", valueString(t, m, ctx, exc))
	}
	return v
}

func alloc(t *testing.T, m *qjs.Module, ctx qjs.JSContextPtr, n uint32) qjs.Ptr[[0]byte] {
	t.Helper()
	p := m.JS_mallocz(ctx, n)
	if p == 0 {
		t.Fatal("allocation returned NULL")
	}
	t.Cleanup(func() { m.JS_free(ctx, p) })
	return p
}

func requireValue(t *testing.T, m *qjs.Module, ctx qjs.JSContextPtr, v qjs.JSValue) qjs.JSValue {
	t.Helper()
	if qjs.JS_IsException(v) {
		exc := m.JS_GetException(ctx)
		defer m.JS_FreeValue(ctx, exc)
		t.Fatalf("unexpected exception: %s", valueString(t, m, ctx, exc))
	}
	return v
}
