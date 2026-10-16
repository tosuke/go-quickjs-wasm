package quickjs_wasm_test

import (
	"bytes"
	"encoding/binary"
	"testing"

	qjs "github.com/tosuke/go-quickjs-wasm"
)

func TestGoCallbacksAndFunctionListMacros(t *testing.T) {
	m, _, ctx := context(t)
	cb := registerCallback(m, func(ctx int32, this int64, argc, argv int32) int64 {
		if argc != 1 {
			return qjs.JS_EXCEPTION
		}
		arg := int64(binary.LittleEndian.Uint64((*m.Xmemory().Slice())[uint32(argv):]))
		// Force growth from inside the callback, then call the module again.
		p := m.Xjs_malloc(ctx, 4<<20)
		m.Xjs_free(ctx, p)
		return m.XJS_NewInt32(ctx, m.XJS_VALUE_GET_INT(arg)+2)
	})
	entry := alloc(t, m, ctx, qjs.SizeofJSCFunctionListEntry)
	m.XJS_CFUNC_DEF(entry, cstring(t, m, ctx, "addTwo"), 1, cb)
	global := m.XJS_GetGlobalObject(ctx)
	defer m.XJS_FreeValue(ctx, global)
	if m.XJS_SetPropertyFunctionList(ctx, global, entry, 1) != 0 {
		t.Fatal("SetPropertyFunctionList failed")
	}
	v := eval(t, m, ctx, "addTwo(40)")
	defer m.XJS_FreeValue(ctx, v)
	if m.XJS_VALUE_GET_INT(v) != 42 {
		t.Fatal("Go callback result mismatch")
	}
	m.XJS_PROP_DOUBLE_DEF(entry, cstring(t, m, ctx, "half"), 0.5, qjs.JS_PROP_C_W_E)
	if m.XJS_SetPropertyFunctionList(ctx, global, entry, 1) != 0 {
		t.Fatal("double initializer failed")
	}
	v2 := eval(t, m, ctx, "half * 84")
	defer m.XJS_FreeValue(ctx, v2)
	if valueString(t, m, ctx, v2) != "42" {
		t.Fatal("double initializer result mismatch")
	}
}

func TestCFunctionCallbackVariants(t *testing.T) {
	m, _, ctx := context(t)
	global := m.XJS_GetGlobalObject(ctx)
	defer m.XJS_FreeValue(ctx, global)
	entry := alloc(t, m, ctx, qjs.SizeofJSCFunctionListEntry)
	floatCallback := registerCallback(m, func(v float64) float64 { return v * v })
	m.XJS_CFUNC_SPECIAL_DEF(entry, cstring(t, m, ctx, "square"), 1, qjs.JS_CFUNC_f_f, floatCallback)
	if m.XJS_SetPropertyFunctionList(ctx, global, entry, 1) != 0 {
		t.Fatal("special function initializer failed")
	}
	magic := registerCallback(m, func(gotCtx int32, this int64, argc, argv, magic int32) int64 {
		return m.XJS_NewInt32(gotCtx, magic)
	})
	fn := requireValue(t, m, ctx, m.XJS_NewCFunctionMagic(ctx, magic, cstring(t, m, ctx, "magic"), 0, qjs.JS_CFUNC_generic_magic, 42))
	if m.XJS_SetPropertyStr(ctx, global, cstring(t, m, ctx, "magic"), fn) < 0 {
		t.Fatal("setting magic function failed")
	}
	data := alloc(t, m, ctx, 8)
	binary.LittleEndian.PutUint64((*m.Xmemory().Slice())[uint32(data):], uint64(m.XJS_NewInt32(ctx, 42)))
	dataCallback := registerCallback(m, func(gotCtx int32, this int64, argc, argv, magic, data int32) int64 {
		return m.XJS_DupValue(gotCtx, int64(binary.LittleEndian.Uint64((*m.Xmemory().Slice())[uint32(data):])))
	})
	fn = requireValue(t, m, ctx, m.XJS_NewCFunctionData(ctx, dataCallback, 0, 0, 1, data))
	if m.XJS_SetPropertyStr(ctx, global, cstring(t, m, ctx, "data"), fn) < 0 {
		t.Fatal("setting data function failed")
	}
	getter := registerCallback(m, func(gotCtx int32, this int64) int64 {
		return m.XJS_NewInt32(gotCtx, 42)
	})
	// Function-list entries can be retained for lazy property initialization.
	entry = alloc(t, m, ctx, qjs.SizeofJSCFunctionListEntry)
	m.XJS_CGETSET_DEF(entry, cstring(t, m, ctx, "fromGetter"), getter, 0)
	if m.XJS_SetPropertyFunctionList(ctx, global, entry, 1) != 0 {
		t.Fatal("getter initializer failed")
	}
	v := eval(t, m, ctx, "[square(1.5), magic(), data(), fromGetter].join(',')")
	defer m.XJS_FreeValue(ctx, v)
	if got := valueString(t, m, ctx, v); got != "2.25,42,42,42" {
		t.Fatalf("callback variants: %s", got)
	}
}

func TestInterruptAndJobCallbacks(t *testing.T) {
	m, rt, ctx := context(t)
	calls := 0
	interrupt := registerCallback(m, func(gotRT, opaque int32) int32 {
		if gotRT != rt || opaque != 123 {
			t.Fatal("interrupt callback arguments mismatch")
		}
		calls++
		return 1
	})
	m.XJS_SetInterruptHandler(rt, interrupt, 123)
	source := "for (;;) {}"
	v := m.XJS_Eval(ctx, cstring(t, m, ctx, source), int32(len(source)), cstring(t, m, ctx, "interrupt.js"), 0)
	if m.XJS_IsException(v) == 0 || calls == 0 {
		t.Fatal("execution was not interrupted")
	}
	exc := m.XJS_GetException(ctx)
	m.XJS_FreeValue(ctx, exc)
	m.XJS_SetInterruptHandler(rt, 0, 0)
	(*m.X__indirect_function_table())[uint32(interrupt)] = nil
	jobCalls := 0
	job := registerCallback(m, func(gotCtx, argc, argv int32) int64 {
		if gotCtx != ctx || argc != 0 {
			t.Fatal("job callback arguments mismatch")
		}
		jobCalls++
		return qjs.JS_UNDEFINED
	})
	if m.XJS_EnqueueJob(ctx, job, 0, 0) != 0 || m.XJS_IsJobPending(rt) == 0 {
		t.Fatal("job was not queued")
	}
	out := alloc(t, m, ctx, 4)
	if m.XJS_ExecutePendingJob(rt, out) != 1 || jobCalls != 1 {
		t.Fatal("Go job did not execute")
	}
	(*m.X__indirect_function_table())[uint32(job)] = nil
}

func TestArrayBufferFreeAndPromiseTrackerCallbacks(t *testing.T) {
	m, rt, ctx := context(t)
	p := m.Xjs_malloc_rt(rt, 16)
	if p == 0 {
		t.Fatal("malloc failed")
	}
	freed := false
	free := registerCallback(m, func(gotRT, opaque, ptr int32) {
		if gotRT != rt || opaque != 123 || ptr != p {
			t.Fatal("buffer free callback arguments mismatch")
		}
		freed = true
		m.Xjs_free_rt(gotRT, ptr)
	})
	buf := requireValue(t, m, ctx, m.XJS_NewArrayBuffer(ctx, p, 16, free, 123, 0))
	m.XJS_FreeValue(ctx, buf)
	if !freed {
		t.Fatal("ArrayBuffer free callback did not run")
	}
	tracked := false
	tracker := registerCallback(m, func(gotCtx int32, promise, reason int64, handled, opaque int32) {
		if gotCtx != ctx || opaque != 456 || m.XJS_VALUE_GET_INT(reason) != 42 {
			t.Fatal("promise tracker arguments mismatch")
		}
		tracked = true
	})
	m.XJS_SetHostPromiseRejectionTracker(rt, tracker, 456)
	v := eval(t, m, ctx, "Promise.reject(42)")
	m.XJS_FreeValue(ctx, v)
	if !tracked {
		t.Fatal("promise rejection tracker did not run")
	}
}

func TestCustomAllocatorCallbacks(t *testing.T) {
	m := qjs.New()
	// A default runtime backs the custom allocator without exposing libc.
	backingRT := m.XJS_NewRuntime()
	if backingRT == 0 {
		t.Fatal("backing runtime allocation failed")
	}
	t.Cleanup(func() { m.XJS_FreeRuntime(backingRT) })
	blocks := map[int32]int32{}
	account := func(state, countDelta, sizeDelta int32) {
		write32(m, state+qjs.OffsetofJSMallocState_malloc_count, read32(m, state+qjs.OffsetofJSMallocState_malloc_count)+countDelta)
		write32(m, state+qjs.OffsetofJSMallocState_malloc_size, read32(m, state+qjs.OffsetofJSMallocState_malloc_size)+sizeDelta)
	}
	malloc := func(state, size int32) int32 {
		p := m.Xjs_malloc_rt(backingRT, size)
		if p != 0 {
			blocks[p] = size
			account(state, 1, size)
		}
		return p
	}
	free := func(state, ptr int32) {
		if ptr != 0 {
			account(state, -1, -blocks[ptr])
			delete(blocks, ptr)
			m.Xjs_free_rt(backingRT, ptr)
		}
	}
	realloc := func(state, ptr, size int32) int32 {
		if ptr == 0 {
			return malloc(state, size)
		}
		if size == 0 {
			free(state, ptr)
			return 0
		}
		oldSize := blocks[ptr]
		p := m.Xjs_realloc_rt(backingRT, ptr, size)
		if p != 0 {
			delete(blocks, ptr)
			blocks[p] = size
			account(state, 0, size-oldSize)
		}
		return p
	}
	mf := m.Xjs_mallocz_rt(backingRT, qjs.SizeofJSMallocFunctions)
	if mf == 0 {
		t.Fatal("allocator descriptor allocation failed")
	}
	t.Cleanup(func() { m.Xjs_free_rt(backingRT, mf) })
	t.Cleanup(func() {
		if len(blocks) != 0 {
			t.Errorf("custom allocator leaked %d blocks", len(blocks))
		}
	})
	write32(m, mf+qjs.OffsetofJSMallocFunctions_js_malloc, registerCallback(m, malloc))
	write32(m, mf+qjs.OffsetofJSMallocFunctions_js_free, registerCallback(m, free))
	write32(m, mf+qjs.OffsetofJSMallocFunctions_js_realloc, registerCallback(m, realloc))
	write32(m, mf+qjs.OffsetofJSMallocFunctions_js_malloc_usable_size, registerCallback(m, func(ptr int32) int32 { return blocks[ptr] }))
	rt := m.XJS_NewRuntime2(mf, 123)
	if rt == 0 {
		t.Fatal("NewRuntime2 failed")
	}
	t.Cleanup(func() { m.XJS_FreeRuntime(rt) })
	ctx := m.XJS_NewContext(rt)
	if ctx == 0 {
		t.Fatal("NewContext failed")
	}
	t.Cleanup(func() { m.XJS_FreeContext(ctx) })
	v := eval(t, m, ctx, "Array.from({length: 1000}, (_, i) => i).at(-1)")
	if valueString(t, m, ctx, v) != "999" {
		t.Fatal("custom allocator execution failed")
	}
	m.XJS_FreeValue(ctx, v)
}

func TestModuleLoaderCallbacks(t *testing.T) {
	m, rt, ctx := context(t)
	loaded, initialized := false, false
	init := registerCallback(m, func(gotCtx, module int32) int32 {
		initialized = true
		return m.XJS_SetModuleExport(gotCtx, module, cstring(t, m, ctx, "answer"), m.XJS_NewInt32(gotCtx, 42))
	})
	loader := registerCallback(m, func(gotCtx, name, opaque int32) int32 {
		loaded = true
		module := m.XJS_NewCModule(gotCtx, name, init)
		if module == 0 || m.XJS_AddModuleExport(gotCtx, module, cstring(t, m, ctx, "answer")) != 0 {
			return 0
		}
		return module
	})
	m.XJS_SetModuleLoaderFunc(rt, 0, loader, 0)
	source := "import { answer } from 'host'; globalThis.answer = answer;"
	v := requireValue(t, m, ctx, m.XJS_Eval(ctx, cstring(t, m, ctx, source), int32(len(source)), cstring(t, m, ctx, "module.mjs"), qjs.JS_EVAL_TYPE_MODULE))
	defer m.XJS_FreeValue(ctx, v)
	if !loaded || !initialized {
		t.Fatal("module callbacks did not run")
	}
	result := eval(t, m, ctx, "answer")
	defer m.XJS_FreeValue(ctx, result)
	if m.XJS_VALUE_GET_INT(result) != 42 {
		t.Fatal("module export mismatch")
	}
}

func TestClassFinalizerAndPrintCallbacks(t *testing.T) {
	m, rt, ctx := context(t)
	classID := alloc(t, m, ctx, 4)
	id := m.XJS_NewClassID(classID)
	finalized := false
	finalizer := registerCallback(m, func(gotRT int32, value int64) {
		if gotRT != rt || m.XJS_GetOpaque(value, id) != 123 {
			t.Fatal("finalizer arguments mismatch")
		}
		finalized = true
	})
	def := alloc(t, m, ctx, qjs.SizeofJSClassDef)
	write32(m, def+qjs.OffsetofJSClassDef_class_name, cstring(t, m, ctx, "HostClass"))
	write32(m, def+qjs.OffsetofJSClassDef_finalizer, finalizer)
	if m.XJS_NewClass(rt, id, def) != 0 {
		t.Fatal("NewClass failed")
	}
	obj := requireValue(t, m, ctx, m.XJS_NewObjectClass(ctx, id))
	m.XJS_SetOpaque(obj, 123)
	m.XJS_FreeValue(ctx, obj)
	if !finalized {
		t.Fatal("class finalizer did not run")
	}
	var output bytes.Buffer
	printer := registerCallback(m, func(opaque, p, n int32) {
		output.Write((*m.Xmemory().Slice())[uint32(p) : uint32(p)+uint32(n)])
	})
	options := alloc(t, m, ctx, qjs.SizeofJSPrintValueOptions)
	m.XJS_PrintValueSetDefaultOptions(options)
	m.XJSPrintValueOptions_set_show_hidden(options, 1)
	if m.XJSPrintValueOptions_get_show_hidden(options) != 1 {
		t.Fatal("bitfield accessors disagree")
	}
	m.XJS_PrintValue(ctx, printer, 0, m.XJS_NewInt32(ctx, 42), options)
	if output.String() != "42" {
		t.Fatalf("PrintValue = %q", output.String())
	}
}
