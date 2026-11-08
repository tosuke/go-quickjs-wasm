package quickjs_wasm_test

import (
	"reflect"
	"testing"

	qjs "github.com/tosuke/go-quickjs-wasm"
)

func TestCallbackFunctionTypes(t *testing.T) {
	m := module(t)
	c := newTestModule(t, m)
	cases := []struct {
		name      string
		signature any
		call      func(int32)
	}{
		{"JSCFunction", (qjs.JSCFunction)(nil), c.Xtest_call_JSCFunction},
		{"JSCFunctionMagic", (qjs.JSCFunctionMagic)(nil), c.Xtest_call_JSCFunctionMagic},
		{"JSCFunctionData", (qjs.JSCFunctionData)(nil), c.Xtest_call_JSCFunctionData},
		{"JS_MarkFunc", (qjs.JS_MarkFunc)(nil), c.Xtest_call_JS_MarkFunc},
		{"JSClassFinalizer", (qjs.JSClassFinalizer)(nil), c.Xtest_call_JSClassFinalizer},
		{"JSClassGCMark", (qjs.JSClassGCMark)(nil), c.Xtest_call_JSClassGCMark},
		{"JSClassCall", (qjs.JSClassCall)(nil), c.Xtest_call_JSClassCall},
		{"JSFreeArrayBufferDataFunc", (qjs.JSFreeArrayBufferDataFunc)(nil), c.Xtest_call_JSFreeArrayBufferDataFunc},
		{"JSHostPromiseRejectionTracker", (qjs.JSHostPromiseRejectionTracker)(nil), c.Xtest_call_JSHostPromiseRejectionTracker},
		{"JSInterruptHandler", (qjs.JSInterruptHandler)(nil), c.Xtest_call_JSInterruptHandler},
		{"JSModuleNormalizeFunc", (qjs.JSModuleNormalizeFunc)(nil), c.Xtest_call_JSModuleNormalizeFunc},
		{"JSModuleLoaderFunc", (qjs.JSModuleLoaderFunc)(nil), c.Xtest_call_JSModuleLoaderFunc},
		{"JSModuleLoaderFunc2", (qjs.JSModuleLoaderFunc2)(nil), c.Xtest_call_JSModuleLoaderFunc2},
		{"JSModuleCheckSupportedImportAttributes", (qjs.JSModuleCheckSupportedImportAttributes)(nil), c.Xtest_call_JSModuleCheckSupportedImportAttributes},
		{"JSJobFunc", (qjs.JSJobFunc)(nil), c.Xtest_call_JSJobFunc},
		{"JSModuleInitFunc", (qjs.JSModuleInitFunc)(nil), c.Xtest_call_JSModuleInitFunc},
		{"JSPrintValueWrite", (qjs.JSPrintValueWrite)(nil), c.Xtest_call_JSPrintValueWrite},
		{"JSMallocFunctionsMalloc", (qjs.JSMallocFunctionsMalloc)(nil), c.Xtest_call_JSMallocFunctionsMalloc},
		{"JSMallocFunctionsFree", (qjs.JSMallocFunctionsFree)(nil), c.Xtest_call_JSMallocFunctionsFree},
		{"JSMallocFunctionsRealloc", (qjs.JSMallocFunctionsRealloc)(nil), c.Xtest_call_JSMallocFunctionsRealloc},
		{"JSMallocFunctionsMallocUsableSize", (qjs.JSMallocFunctionsMallocUsableSize)(nil), c.Xtest_call_JSMallocFunctionsMallocUsableSize},
		{"JSClassExoticMethodGetOwnProperty", (qjs.JSClassExoticMethodGetOwnProperty)(nil), c.Xtest_call_JSClassExoticMethodGetOwnProperty},
		{"JSClassExoticMethodGetOwnPropertyNames", (qjs.JSClassExoticMethodGetOwnPropertyNames)(nil), c.Xtest_call_JSClassExoticMethodGetOwnPropertyNames},
		{"JSClassExoticMethodDeleteProperty", (qjs.JSClassExoticMethodDeleteProperty)(nil), c.Xtest_call_JSClassExoticMethodDeleteProperty},
		{"JSClassExoticMethodDefineOwnProperty", (qjs.JSClassExoticMethodDefineOwnProperty)(nil), c.Xtest_call_JSClassExoticMethodDefineOwnProperty},
		{"JSClassExoticMethodHasProperty", (qjs.JSClassExoticMethodHasProperty)(nil), c.Xtest_call_JSClassExoticMethodHasProperty},
		{"JSClassExoticMethodGetProperty", (qjs.JSClassExoticMethodGetProperty)(nil), c.Xtest_call_JSClassExoticMethodGetProperty},
		{"JSClassExoticMethodSetProperty", (qjs.JSClassExoticMethodSetProperty)(nil), c.Xtest_call_JSClassExoticMethodSetProperty},
		{"JSClassExoticMethodGetPrototype", (qjs.JSClassExoticMethodGetPrototype)(nil), c.Xtest_call_JSClassExoticMethodGetPrototype},
		{"JSClassExoticMethodSetPrototype", (qjs.JSClassExoticMethodSetPrototype)(nil), c.Xtest_call_JSClassExoticMethodSetPrototype},
		{"JSClassExoticMethodIsExtensible", (qjs.JSClassExoticMethodIsExtensible)(nil), c.Xtest_call_JSClassExoticMethodIsExtensible},
		{"JSClassExoticMethodPreventExtensions", (qjs.JSClassExoticMethodPreventExtensions)(nil), c.Xtest_call_JSClassExoticMethodPreventExtensions},
		{"JSCFunctionGenericMagic", (qjs.JSCFunctionGenericMagic)(nil), c.Xtest_call_JSCFunctionGenericMagic},
		{"JSCFunctionConstructorMagic", (qjs.JSCFunctionConstructorMagic)(nil), c.Xtest_call_JSCFunctionConstructorMagic},
		{"JSCFunctionFF", (qjs.JSCFunctionFF)(nil), c.Xtest_call_JSCFunctionFF},
		{"JSCFunctionFFF", (qjs.JSCFunctionFFF)(nil), c.Xtest_call_JSCFunctionFFF},
		{"JSCFunctionGetter", (qjs.JSCFunctionGetter)(nil), c.Xtest_call_JSCFunctionGetter},
		{"JSCFunctionSetter", (qjs.JSCFunctionSetter)(nil), c.Xtest_call_JSCFunctionSetter},
		{"JSCFunctionGetterMagic", (qjs.JSCFunctionGetterMagic)(nil), c.Xtest_call_JSCFunctionGetterMagic},
		{"JSCFunctionSetterMagic", (qjs.JSCFunctionSetterMagic)(nil), c.Xtest_call_JSCFunctionSetterMagic},
		{"JSCFunctionIteratorNext", (qjs.JSCFunctionIteratorNext)(nil), c.Xtest_call_JSCFunctionIteratorNext},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			signature := reflect.TypeOf(tc.signature)
			callback := reflect.MakeFunc(signature, func(args []reflect.Value) []reflect.Value {
				called = true
				for i, arg := range args {
					if !arg.IsZero() {
						t.Errorf("argument %d = %v, want zero", i, arg.Interface())
					}
				}
				results := make([]reflect.Value, signature.NumOut())
				for i := range results {
					results[i] = reflect.Zero(signature.Out(i))
				}
				return results
			}).Interface()
			table := m.X__indirect_function_table()
			index := int32(len(*table))
			*table = append(*table, callback)
			tc.call(index)
			if !called {
				t.Fatal("C did not call the Go callback")
			}
		})
	}
}

func TestJSCFunctionDataCallback(t *testing.T) {
	m, _, ctx := context(t)
	called := false
	callback := qjs.JSCFunctionData(func(gotCtx qjs.JSContextPtr, this qjs.JSValue, argc int32, argv qjs.Ptr[qjs.JSValue], magic int32, data qjs.Ptr[qjs.JSValue]) qjs.JSValue {
		called = true
		if gotCtx != ctx || this != qjs.JS_UNDEFINED || argc != 0 || magic != 42 {
			t.Fatalf("callback arguments = (%d, %x, %d, %d)", gotCtx, this, argc, magic)
		}
		return m.LoadJSValue(data)
	})
	table := m.X__indirect_function_table()
	index := int32(len(*table))
	*table = append(*table, callback)
	data := alloc(t, m, ctx, 8)
	want := qjs.JS_NAN // A negative int64 representation must retain all bits.
	m.StoreJSValue(data, want)
	fn := requireValue(t, m, ctx, m.JS_NewCFunctionData(ctx, index, 0, 42, 1, data))
	defer m.JS_FreeValue(ctx, fn)
	result := requireValue(t, m, ctx, m.JS_Call(ctx, fn, qjs.JS_UNDEFINED, 0, 0))
	defer m.JS_FreeValue(ctx, result)
	if !called || result != want {
		t.Fatalf("callback result = %x, want %x (called=%v)", result, want, called)
	}
}

func TestVariadicThrowABI(t *testing.T) {
	m, _, ctx := context(t)
	args := alloc(t, m, ctx, 16)
	// wasm32 varargs align i32/pointers to 4 bytes and double/i64 to 8.
	m.StoreI32(args, 42)
	qjs.StorePtr[byte](m, args+4, cstring(t, m, ctx, "answer"))
	m.StoreF64(args+8, 1.5)
	v := m.JS_ThrowTypeError(ctx, cstring(t, m, ctx, "%d %s %.1f"), args)
	if !qjs.JS_IsException(v) {
		t.Fatal("ThrowTypeError did not return JS_EXCEPTION")
	}
	exc := m.JS_GetException(ctx)
	defer m.JS_FreeValue(ctx, exc)
	if got := valueString(t, m, ctx, exc); got != "TypeError: 42 answer 1.5" {
		t.Fatalf("varargs mismatch: %q", got)
	}
}
