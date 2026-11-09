package quickjs_wasm_test

import (
	"testing"
	"testing/synctest"
	"time"
	"unsafe"

	qjs "github.com/tosuke/go-quickjs-wasm"
)

func TestMemoryGrowthAndIsolation(t *testing.T) {
	m, _, ctx := context(t)
	before := m.Xmemory().Grow(0, 65536)
	base := unsafe.SliceData(*m.Xmemory().Slice())
	v := eval(t, m, ctx, "globalThis.privateValue = 7; new Uint8Array(4 * 1024 * 1024).fill(123).at(-1)")
	defer m.JS_FreeValue(ctx, v)
	if m.Xmemory().Grow(0, 65536) <= before {
		t.Fatal("expected linear memory to grow")
	}
	if unsafe.SliceData(*m.Xmemory().Slice()) != base {
		t.Fatal("linear memory moved during growth")
	}
	if got := valueString(t, m, ctx, v); got != "123" {
		t.Fatalf("array value = %q", got)
	}
	other, _, otherCtx := context(t)
	result := eval(t, other, otherCtx, "typeof privateValue")
	defer other.JS_FreeValue(otherCtx, result)
	if got := valueString(t, other, otherCtx, result); got != "undefined" {
		t.Fatalf("module state leaked: %q", got)
	}
}

func TestHostAllocationMemoryLimit(t *testing.T) {
	m, rt, ctx := context(t)
	m.JS_SetMemoryLimit(rt, 1)
	// Small requests may reuse an existing QuickJS arena without increasing
	// backing memory. Force a new backing allocation to exercise the limit.
	if p := m.JS_malloc(ctx, 1<<20); p != 0 {
		m.JS_free(ctx, p)
		t.Fatal("host allocation exceeded the runtime memory limit")
	}
	if !m.JS_HasException(ctx) {
		t.Fatal("failed host allocation did not raise an exception")
	}
	exc := m.JS_GetException(ctx)
	m.JS_FreeValue(ctx, exc)
	m.JS_SetMemoryLimit(rt, ^uint32(0))
	p := m.JS_malloc(ctx, 64)
	if p == 0 {
		t.Fatal("host allocation failed after restoring the memory limit")
	}
	m.JS_free(ctx, p)
}

func TestStackLimit(t *testing.T) {
	m, rt, ctx := context(t)
	m.JS_SetMaxStackSize(rt, 64*1024)
	source := "function recurse() { return recurse() + 1; } recurse()"
	v := m.JS_Eval(ctx, cstring(t, m, ctx, source), int32(len(source)), cstring(t, m, ctx, "stack.js"), 0)
	if !qjs.JS_IsException(v) {
		m.JS_FreeValue(ctx, v)
		t.Fatal("expected stack overflow exception")
	}
	exc := m.JS_GetException(ctx)
	defer m.JS_FreeValue(ctx, exc)
	if got := valueString(t, m, ctx, exc); got != "InternalError: stack overflow" {
		t.Fatalf("unexpected exception: %q", got)
	}
}

func TestPendingJob(t *testing.T) {
	m, rt, ctx := context(t)
	v := eval(t, m, ctx, "globalThis.result = 0; Promise.resolve(42).then(v => result = v); undefined")
	m.JS_FreeValue(ctx, v)
	gotCtx, status := m.JS_ExecutePendingJob(rt)
	if status != 1 {
		t.Fatalf("ExecutePendingJob = %d, want 1", status)
	}
	if gotCtx != ctx {
		t.Fatalf("job context = %d, want %d", gotCtx, ctx)
	}
	if _, got := m.JS_ExecutePendingJob(rt); got != 0 {
		t.Fatalf("ExecutePendingJob = %d, want 0", got)
	}
	v = eval(t, m, ctx, "result")
	defer m.JS_FreeValue(ctx, v)
	if got := valueString(t, m, ctx, v); got != "42" {
		t.Fatalf("result = %q, want 42", got)
	}
}

func TestDateNow(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		m, _, ctx := context(t)
		for _, advance := range []time.Duration{0, 1234 * time.Millisecond} {
			time.Sleep(advance)
			v := eval(t, m, ctx, "Date.now()")
			number, status := m.JS_ToFloat64(ctx, v)
			m.JS_FreeValue(ctx, v)
			if status != 0 {
				t.Fatal("JS_ToFloat64 failed")
			}
			if want := float64(time.Now().UnixMilli()); number != want {
				t.Fatalf("Date.now() = %v, want %v", number, want)
			}
		}
	})
}
