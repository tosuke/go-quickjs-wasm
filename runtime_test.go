package quickjs_wasm_test

import (
	"encoding/binary"
	"testing"
)

func TestMemoryGrowthAndIsolation(t *testing.T) {
	m, _, ctx := context(t)
	before := len(*m.Xmemory().Slice())
	v := eval(t, m, ctx, "globalThis.privateValue = 7; new Uint8Array(4 * 1024 * 1024).fill(123).at(-1)")
	defer m.XJS_FreeValue(ctx, v)
	if len(*m.Xmemory().Slice()) <= before {
		t.Fatal("expected linear memory to grow")
	}
	if got := valueString(t, m, ctx, v); got != "123" {
		t.Fatalf("array value = %q", got)
	}
	other, _, otherCtx := context(t)
	result := eval(t, other, otherCtx, "typeof privateValue")
	defer other.XJS_FreeValue(otherCtx, result)
	if got := valueString(t, other, otherCtx, result); got != "undefined" {
		t.Fatalf("module state leaked: %q", got)
	}
}

func TestHostAllocationMemoryLimit(t *testing.T) {
	m, rt, ctx := context(t)
	m.XJS_SetMemoryLimit(rt, 1)
	// Small requests may reuse an existing QuickJS arena without increasing
	// backing memory. Force a new backing allocation to exercise the limit.
	if p := m.Xjs_malloc(ctx, 1<<20); p != 0 {
		m.Xjs_free(ctx, p)
		t.Fatal("host allocation exceeded the runtime memory limit")
	}
	if m.XJS_HasException(ctx) == 0 {
		t.Fatal("failed host allocation did not raise an exception")
	}
	exc := m.XJS_GetException(ctx)
	m.XJS_FreeValue(ctx, exc)
	m.XJS_SetMemoryLimit(rt, -1)
	p := m.Xjs_malloc(ctx, 64)
	if p == 0 {
		t.Fatal("host allocation failed after restoring the memory limit")
	}
	m.Xjs_free(ctx, p)
}

func TestStackLimit(t *testing.T) {
	m, rt, ctx := context(t)
	m.XJS_SetMaxStackSize(rt, 64*1024)
	source := "function recurse() { return recurse() + 1; } recurse()"
	v := m.XJS_Eval(ctx, cstring(t, m, ctx, source), int32(len(source)), cstring(t, m, ctx, "stack.js"), 0)
	if m.XJS_IsException(v) == 0 {
		m.XJS_FreeValue(ctx, v)
		t.Fatal("expected stack overflow exception")
	}
	exc := m.XJS_GetException(ctx)
	defer m.XJS_FreeValue(ctx, exc)
	if got := valueString(t, m, ctx, exc); got != "InternalError: stack overflow" {
		t.Fatalf("unexpected exception: %q", got)
	}
}

func TestPendingJob(t *testing.T) {
	m, rt, ctx := context(t)
	v := eval(t, m, ctx, "globalThis.result = 0; Promise.resolve(42).then(v => result = v); undefined")
	m.XJS_FreeValue(ctx, v)
	out := m.Xjs_malloc(ctx, 4)
	defer m.Xjs_free(ctx, out)
	if got := m.XJS_ExecutePendingJob(rt, out); got != 1 {
		t.Fatalf("ExecutePendingJob = %d, want 1", got)
	}
	if got := int32(binary.LittleEndian.Uint32((*m.Xmemory().Slice())[uint32(out):])); got != ctx {
		t.Fatalf("job context = %d, want %d", got, ctx)
	}
	if got := m.XJS_ExecutePendingJob(rt, out); got != 0 {
		t.Fatalf("ExecutePendingJob = %d, want 0", got)
	}
	v = eval(t, m, ctx, "result")
	defer m.XJS_FreeValue(ctx, v)
	if got := valueString(t, m, ctx, v); got != "42" {
		t.Fatalf("result = %q, want 42", got)
	}
}
