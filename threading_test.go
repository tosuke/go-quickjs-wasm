package quickjs_wasm_test

import (
	"bytes"
	"sync"
	"testing"
	"testing/synctest"

	qjs "github.com/tosuke/go-quickjs-wasm"
)

func threadContext(t *testing.T, m *qjs.Module) (qjs.JSRuntimePtr, qjs.JSContextPtr) {
	t.Helper()
	rt := m.JS_NewRuntime()
	if rt == 0 {
		t.Fatal("JS_NewRuntime failed")
	}
	t.Cleanup(func() { m.JS_FreeRuntime(rt) })
	m.JS_SetDefaultSharedArrayBufferFunctions(rt)
	ctx := m.JS_NewContext(rt)
	if ctx == 0 {
		t.Fatal("JS_NewContext failed")
	}
	t.Cleanup(func() { m.JS_FreeContext(ctx) })
	return rt, ctx
}

func TestSharedAllocatorWorkers(t *testing.T) {
	shared := sharedMemory(t)
	const workers, blocks = 4, 64
	modules := make([]*qjs.Module, workers)
	var wg sync.WaitGroup
	// Exercise construction against the same memory, including first use.
	for i := range modules {
		wg.Go(func() { modules[i] = worker(t, shared) })
	}
	wg.Wait()
	if err := shared.Close(); err == nil {
		t.Fatal("closed memory with live workers")
	}
	type allocation struct {
		ptr, size int32
		pattern   byte
	}
	var allocations [workers][blocks]allocation
	before := len(shared.Bytes())
	start := make(chan struct{})
	for i, m := range modules {
		wg.Go(func() {
			<-start
			for j := range blocks {
				size := int32(128 + (j%11)*4096)
				p := m.Xmalloc(size)
				if p == 0 {
					t.Error("malloc failed")
					return
				}
				pattern := byte(i*blocks + j)
				m.StoreBytes(p, bytes.Repeat([]byte{pattern}, int(size)))
				allocations[i][j] = allocation{p, size, pattern}
			}
		})
	}
	close(start)
	wg.Wait()
	if len(shared.Bytes()) <= before {
		t.Fatal("allocator stress did not grow shared memory")
	}
	seen := make(map[int32]bool)
	for _, group := range allocations {
		for _, a := range group {
			if a.ptr == 0 || seen[a.ptr] {
				t.Fatalf("duplicate/null live allocation %d", a.ptr)
			}
			seen[a.ptr] = true
		}
	}
	// Reallocations run on separate C stacks while contending on one heap.
	for i, m := range modules {
		wg.Go(func() {
			for j, a := range allocations[i] {
				p := m.Xrealloc(a.ptr, a.size*2)
				if p == 0 {
					t.Error("realloc failed")
					return
				}
				got := make([]byte, int(a.size))
				m.LoadBytes(got, p, len(got))
				if !bytes.Equal(got, bytes.Repeat([]byte{a.pattern}, len(got))) {
					t.Error("realloc/neighbor allocation corrupted data")
				}
				allocations[i][j].ptr = p
			}
		})
	}
	wg.Wait()
	// Free each block from a worker other than its allocating worker.
	for i, m := range modules {
		wg.Go(func() {
			for _, a := range allocations[(i+1)%workers] {
				m.Xfree(a.ptr)
			}
		})
	}
	wg.Wait()
}

func TestWorkerTLSAndClassIDs(t *testing.T) {
	shared := sharedMemory(t)
	a, b := worker(t, shared), worker(t, shared)
	pa, pb := a.X__errno_location(), b.X__errno_location()
	if pa == 0 || pa == pb {
		t.Fatal("workers share C TLS")
	}
	a.StoreI32(pa, 111)
	b.StoreI32(pb, 222)
	if a.LoadI32(pa) != 111 || b.LoadI32(pb) != 222 {
		t.Fatal("TLS values leaked")
	}
	if *a.X__stack_pointer() == *b.X__stack_pointer() {
		t.Fatal("workers share C stacks")
	}
	const count = 1000
	ids := make(chan int32, 2*count)
	var wg sync.WaitGroup
	for _, m := range []*qjs.Module{a, b} {
		wg.Go(func() {
			p := m.Xmalloc(4)
			defer m.Xfree(p)
			for range count {
				m.StoreI32(p, 0)
				ids <- m.JS_NewClassID(p)
			}
		})
	}
	wg.Wait()
	close(ids)
	seen := make(map[int32]bool)
	for id := range ids {
		if id == 0 || seen[id] {
			t.Fatalf("duplicate/invalid class ID %d", id)
		}
		seen[id] = true
	}
}

// Use temporary raw allocations so a producing runtime can be destroyed before
// the receiving workers execute. No test cleanup may retain producer-owned data.
func rawEval(t *testing.T, m *qjs.Module, ctx int32, source string) qjs.JSValue {
	t.Helper()
	code := m.Xmalloc(int32(len(source) + 1))
	defer m.Xfree(code)
	m.StoreString(code, source)
	v := m.JS_Eval(ctx, code, int32(len(source)), code, 0)
	if qjs.JS_IsException(v) {
		e := m.JS_GetException(ctx)
		defer m.JS_FreeValue(ctx, e)
		t.Fatalf("JS_Eval: %s", valueString(t, m, ctx, e))
	}
	return v
}

func cloneSharedBuffer(t *testing.T, origin *qjs.Module, ctx int32, value qjs.JSValue, modules []*qjs.Module, contexts []int32) {
	t.Helper()
	data, size, table, _ := origin.JS_WriteObject2(ctx, value, qjs.JS_WRITE_OBJ_SAB)
	if data == 0 {
		t.Fatal("cannot serialize SAB")
	}
	defer origin.JS_free(ctx, data)
	defer origin.JS_free(ctx, table)
	for i, m := range modules {
		cloned := m.JS_ReadObject(contexts[i], data, size, qjs.JS_READ_OBJ_SAB)
		if qjs.JS_IsException(cloned) {
			t.Fatal("cannot clone SAB")
		}
		global := m.JS_GetGlobalObject(contexts[i])
		prop := m.Xmalloc(7)
		m.StoreString(prop, "shared")
		status := m.JS_SetPropertyStr(contexts[i], global, prop, cloned)
		m.Xfree(prop)
		m.JS_FreeValue(contexts[i], global)
		if status < 0 {
			t.Fatal("cannot install SAB")
		}
	}
}

func TestParallelQuickJSWorkers(t *testing.T) {
	shared := sharedMemory(t)
	origin := worker(t, shared)
	rt := origin.JS_NewRuntime()
	origin.JS_SetDefaultSharedArrayBufferFunctions(rt)
	ctx := origin.JS_NewContext(rt)
	defer func() {
		if rt != 0 {
			origin.JS_FreeContext(ctx)
			origin.JS_FreeRuntime(rt)
		}
	}()
	buffer := rawEval(t, origin, ctx, "new SharedArrayBuffer(8)")
	const workers, iterations = 4, 1000
	modules := make([]*qjs.Module, workers)
	contexts := make([]int32, workers)
	for i := range modules {
		modules[i] = worker(t, shared)
		_, contexts[i] = threadContext(t, modules[i])
	}
	cloneSharedBuffer(t, origin, ctx, buffer, modules, contexts)
	origin.JS_FreeValue(ctx, buffer)
	origin.JS_FreeContext(ctx)
	origin.JS_FreeRuntime(rt)
	rt = 0
	if err := origin.Close(); err != nil {
		t.Fatal(err)
	}
	source := "globalThis.hold=new Uint8Array(4*1024*1024).fill(37);for(let i=0;i<1000;i++){Atomics.add(new Int32Array(shared),0,1);let a=new Uint8Array((i%31+1)*4096);a.fill(i&255);if(a.at(-1)!=(i&255))throw Error('heap corruption');}"
	start := make(chan struct{})
	var wg sync.WaitGroup
	before := len(shared.Bytes())
	for i, m := range modules {
		code, file := cstring(t, m, contexts[i], source), cstring(t, m, contexts[i], "worker.js")
		wg.Go(func() {
			<-start
			v := m.JS_Eval(contexts[i], code, int32(len(source)), file, 0)
			if qjs.JS_IsException(v) {
				e := m.JS_GetException(contexts[i])
				t.Errorf("worker %d: %s", i, valueString(t, m, contexts[i], e))
				m.JS_FreeValue(contexts[i], e)
			}
			m.JS_FreeValue(contexts[i], v)
		})
	}
	close(start)
	wg.Wait()
	if len(shared.Bytes()) <= before {
		t.Fatal("parallel workers did not exercise memory growth")
	}
	result := eval(t, modules[0], contexts[0], "Atomics.load(new Int32Array(shared),0)")
	defer modules[0].JS_FreeValue(contexts[0], result)
	if got := qjs.JS_VALUE_GET_INT(result); got != workers*iterations {
		t.Fatalf("shared counter=%d, want %d", got, workers*iterations)
	}
}

func TestWorkersWaitNotify(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		shared := sharedMemory(t)
		a, b := worker(t, shared), worker(t, shared)
		ra, ca := threadContext(t, a)
		_, cb := threadContext(t, b)
		a.JS_SetCanBlock(ra, true)
		buf := eval(t, a, ca, "globalThis.shared=new SharedArrayBuffer(8);shared")
		defer a.JS_FreeValue(ca, buf)
		cloneSharedBuffer(t, a, ca, buf, []*qjs.Module{b}, []int32{cb})
		source := "Atomics.wait(new Int32Array(shared),0,0)"
		code, file := cstring(t, a, ca, source), cstring(t, a, ca, "wait.js")
		done := make(chan qjs.JSValue, 1)
		go func() { done <- a.JS_Eval(ca, code, int32(len(source)), file, 0) }()
		synctest.Wait()
		result := eval(t, b, cb, "Atomics.store(new Int32Array(shared),0,42);Atomics.notify(new Int32Array(shared),0,1)")
		defer b.JS_FreeValue(cb, result)
		if qjs.JS_VALUE_GET_INT(result) != 1 {
			t.Fatal("did not wake waiting worker")
		}
		waited := <-done
		defer a.JS_FreeValue(ca, waited)
		if got := valueString(t, a, ca, waited); got != "ok" {
			t.Fatalf("wait=%s", got)
		}
	})
}
