package quickjs_wasm_test

import (
	"bytes"
	"encoding/binary"
	"testing"

	qjs "github.com/tosuke/go-quickjs-wasm"
)

func context(t *testing.T) (*qjs.Module, int32, int32) {
	t.Helper()
	m := qjs.New()
	rt := m.XJS_NewRuntime()
	if rt == 0 {
		t.Fatal("JS_NewRuntime returned NULL")
	}
	t.Cleanup(func() { m.XJS_FreeRuntime(rt) })
	ctx := m.XJS_NewContext(rt)
	if ctx == 0 {
		t.Fatal("JS_NewContext returned NULL")
	}
	t.Cleanup(func() { m.XJS_FreeContext(ctx) })
	return m, rt, ctx
}

func cstring(t *testing.T, m *qjs.Module, ctx int32, s string) int32 {
	t.Helper()
	p := m.Xjs_malloc(ctx, int32(len(s)+1))
	if p == 0 {
		t.Fatal("malloc returned NULL")
	}
	mem := *m.Xmemory().Slice()
	copy(mem[uint32(p):], s)
	mem[uint32(p)+uint32(len(s))] = 0
	t.Cleanup(func() { m.Xjs_free(ctx, p) })
	return p
}

func valueString(t *testing.T, m *qjs.Module, ctx int32, v int64) string {
	t.Helper()
	p := m.XJS_ToCString(ctx, v)
	if p == 0 {
		t.Fatal("JS_ToCString returned NULL")
	}
	defer m.XJS_FreeCString(ctx, p)
	b := (*m.Xmemory().Slice())[uint32(p):]
	n := bytes.IndexByte(b, 0)
	if n < 0 {
		t.Fatal("unterminated C string")
	}
	return string(b[:n])
}

func eval(t *testing.T, m *qjs.Module, ctx int32, source string) int64 {
	t.Helper()
	v := m.XJS_Eval(ctx, cstring(t, m, ctx, source), int32(len(source)), cstring(t, m, ctx, "test.js"), 0)
	if m.XJS_IsException(v) != 0 {
		exc := m.XJS_GetException(ctx)
		defer m.XJS_FreeValue(ctx, exc)
		t.Fatalf("JS_Eval: %s", valueString(t, m, ctx, exc))
	}
	return v
}

func alloc(t *testing.T, m *qjs.Module, ctx int32, n int32) int32 {
	t.Helper()
	p := m.Xjs_mallocz(ctx, n)
	if p == 0 {
		t.Fatal("allocation returned NULL")
	}
	t.Cleanup(func() { m.Xjs_free(ctx, p) })
	return p
}

func read32(m *qjs.Module, p int32) int32 {
	return int32(binary.LittleEndian.Uint32((*m.Xmemory().Slice())[uint32(p):]))
}

func write32(m *qjs.Module, p, value int32) {
	binary.LittleEndian.PutUint32((*m.Xmemory().Slice())[uint32(p):], uint32(value))
}

func registerCallback(m *qjs.Module, fn any) int32 {
	table := m.X__indirect_function_table()
	pointer := int32(len(*table))
	*table = append(*table, fn)
	return pointer
}

func requireValue(t *testing.T, m *qjs.Module, ctx int32, v int64) int64 {
	t.Helper()
	if m.XJS_IsException(v) != 0 {
		exc := m.XJS_GetException(ctx)
		defer m.XJS_FreeValue(ctx, exc)
		t.Fatalf("unexpected exception: %s", valueString(t, m, ctx, exc))
	}
	return v
}
