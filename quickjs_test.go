package quickjs_wasm_test

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"
	"time"

	qjs "github.com/tosuke/go-quickjs-wasm"
)

func TestEval(t *testing.T) {
	for _, tc := range []struct{ name, source, want string }{
		{"integer", "6 * 7", "42"},
		{"float", "Math.sqrt(2) ** 2", "2.0000000000000004"},
		{"unicode", `'こんにちは🌍'.normalize('NFC')`, "こんにちは🌍"},
		{"regexp", `JSON.stringify('abc123'.match(/(?<n>\d+)/).groups)`, `{"n":"123"}`},
		{"bigint", "(2n ** 100n).toString()", "1267650600228229401496703205376"},
		{"date", "new Date(0).toISOString()", "1970-01-01T00:00:00.000Z"},
		{"typedarray", "new Uint32Array([1, 2, 3]).reduce((a,b) => a+b)", "6"},
		{"nan", "String(NaN) + ',' + Object.is(-0, -1 / Infinity)", "NaN,true"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, _, ctx := context(t)
			v := eval(t, m, ctx, tc.source)
			defer m.XJS_FreeValue(ctx, v)
			if got := valueString(t, m, ctx, v); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestException(t *testing.T) {
	m, _, ctx := context(t)
	source := "throw new TypeError('boom')"
	v := m.XJS_Eval(ctx, cstring(t, m, ctx, source), int32(len(source)), cstring(t, m, ctx, "throw.js"), 0)
	if m.XJS_IsException(v) == 0 {
		m.XJS_FreeValue(ctx, v)
		t.Fatal("expected exception")
	}
	exc := m.XJS_GetException(ctx)
	defer m.XJS_FreeValue(ctx, exc)
	if got := valueString(t, m, ctx, exc); got != "TypeError: boom" {
		t.Fatalf("unexpected exception: %q", got)
	}
	// Retrieving an exception must leave the context usable.
	v = eval(t, m, ctx, "40 + 2")
	m.XJS_FreeValue(ctx, v)
}

func TestPropertyAndCall(t *testing.T) {
	m, _, ctx := context(t)
	obj := eval(t, m, ctx, "({ base: 40, add(n) { return this.base + n; } })")
	defer m.XJS_FreeValue(ctx, obj)
	fn := m.XJS_GetPropertyStr(ctx, obj, cstring(t, m, ctx, "add"))
	defer m.XJS_FreeValue(ctx, fn)
	arg := eval(t, m, ctx, "2")
	defer m.XJS_FreeValue(ctx, arg)
	argv := m.Xjs_malloc(ctx, 8)
	defer m.Xjs_free(ctx, argv)
	binary.LittleEndian.PutUint64((*m.Xmemory().Slice())[uint32(argv):], uint64(arg))
	v := m.XJS_Call(ctx, fn, obj, 1, argv)
	defer m.XJS_FreeValue(ctx, v)
	if m.XJS_IsException(v) != 0 {
		t.Fatal("JS_Call returned an exception")
	}
	out := m.Xjs_malloc(ctx, 8)
	defer m.Xjs_free(ctx, out)
	if m.XJS_ToFloat64(ctx, out, v) != 0 {
		t.Fatal("JS_ToFloat64 failed")
	}
	if got := math.Float64frombits(binary.LittleEndian.Uint64((*m.Xmemory().Slice())[uint32(out):])); got != 42 {
		t.Fatalf("got %v, want 42", got)
	}
	if m.XJS_SetPropertyStr(ctx, obj, cstring(t, m, ctx, "base"), m.XJS_DupValue(ctx, arg)) < 0 {
		t.Fatal("JS_SetPropertyStr failed")
	}
	base := m.XJS_GetPropertyStr(ctx, obj, cstring(t, m, ctx, "base"))
	defer m.XJS_FreeValue(ctx, base)
	if got := valueString(t, m, ctx, base); got != "2" {
		t.Fatalf("base = %q, want 2", got)
	}
}

func TestDateNow(t *testing.T) {
	m, _, ctx := context(t)
	before := time.Now().UnixMilli()
	v := eval(t, m, ctx, "Date.now()")
	defer m.XJS_FreeValue(ctx, v)
	out := m.Xjs_malloc(ctx, 8)
	defer m.Xjs_free(ctx, out)
	if m.XJS_ToFloat64(ctx, out, v) != 0 {
		t.Fatal("JS_ToFloat64 failed")
	}
	got := int64(math.Float64frombits(binary.LittleEndian.Uint64((*m.Xmemory().Slice())[uint32(out):])))
	if got < before || got > time.Now().UnixMilli() {
		t.Fatalf("Date.now() = %d is outside host time interval", got)
	}
}

func TestRawValues(t *testing.T) {
	m, _, ctx := context(t)
	for _, d := range []float64{0, math.Copysign(0, -1), 42, 2147483648, 1.25, math.Inf(1), math.Inf(-1), math.NaN()} {
		v := m.XJS_NewFloat64(ctx, d)
		if m.XJS_IsNumber(v) == 0 {
			t.Fatal("float value is not a number")
		}
		if m.XJS_VALUE_GET_TAG(v) == qjs.JS_TAG_INT {
			if float64(m.XJS_VALUE_GET_INT(v)) != d {
				t.Fatalf("integer mismatch: %v", d)
			}
		} else if got := m.XJS_VALUE_GET_FLOAT64(v); !(math.IsNaN(d) && math.IsNaN(got)) && math.Float64bits(got) != math.Float64bits(d) {
			t.Fatalf("float mismatch: %v != %v", got, d)
		}
	}
	if m.XJS_NewBool(ctx, 1) != qjs.JS_TRUE || m.XJS_IsNull(qjs.JS_NULL) == 0 || m.XJS_IsUndefined(qjs.JS_UNDEFINED) == 0 {
		t.Fatal("special values do not match the C header")
	}
	if got := m.XJS_MKVAL(qjs.JS_TAG_INT, -42); got != m.XJS_NewInt32(ctx, -42) {
		t.Fatal("JS_MKVAL disagrees with constructor")
	}
	big := requireValue(t, m, ctx, m.XJS_NewBigInt64(ctx, 1<<60))
	defer m.XJS_FreeValue(ctx, big)
	out := alloc(t, m, ctx, 8)
	if m.XJS_IsBigInt(ctx, big) == 0 || m.XJS_ToBigInt64(ctx, out, big) != 0 || int64(binary.LittleEndian.Uint64((*m.Xmemory().Slice())[uint32(out):])) != 1<<60 {
		t.Fatal("BigInt round trip failed")
	}
}

func TestRawStringsAtomsAndProperties(t *testing.T) {
	m, _, ctx := context(t)
	s := "hello\x00世界"
	v := requireValue(t, m, ctx, m.XJS_NewStringLen(ctx, cstring(t, m, ctx, s), int32(len(s))))
	defer m.XJS_FreeValue(ctx, v)
	size := alloc(t, m, ctx, 4)
	p := m.XJS_ToCStringLen(ctx, size, v)
	if p == 0 {
		t.Fatal("ToCStringLen returned NULL")
	}
	got := string((*m.Xmemory().Slice())[uint32(p) : uint32(p)+uint32(read32(m, size))])
	m.XJS_FreeCString(ctx, p)
	if got != s {
		t.Fatalf("string round trip: %q", got)
	}
	obj := requireValue(t, m, ctx, m.XJS_NewObject(ctx))
	defer m.XJS_FreeValue(ctx, obj)
	atom := m.XJS_NewAtom(ctx, cstring(t, m, ctx, "answer"))
	defer m.XJS_FreeAtom(ctx, atom)
	if m.XJS_DefinePropertyValue(ctx, obj, atom, m.XJS_NewInt32(ctx, 42), qjs.JS_PROP_C_W_E) != 1 {
		t.Fatal("DefinePropertyValue failed")
	}
	prop := requireValue(t, m, ctx, m.XJS_GetProperty(ctx, obj, atom))
	defer m.XJS_FreeValue(ctx, prop)
	if m.XJS_VALUE_GET_INT(prop) != 42 {
		t.Fatal("property round trip failed")
	}
	ptab, plen := alloc(t, m, ctx, 4), alloc(t, m, ctx, 4)
	if m.XJS_GetOwnPropertyNames(ctx, ptab, plen, obj, qjs.JS_GPN_STRING_MASK|qjs.JS_GPN_SET_ENUM) != 0 {
		t.Fatal("GetOwnPropertyNames failed")
	}
	tab, count := read32(m, ptab), read32(m, plen)
	defer m.XJS_FreePropertyEnum(ctx, tab, count)
	if count != 1 || read32(m, tab+qjs.OffsetofJSPropertyEnum_atom) != atom {
		t.Fatal("property enum layout or contents disagree")
	}
	encoded := requireValue(t, m, ctx, m.XJS_JSONStringify(ctx, obj, qjs.JS_UNDEFINED, qjs.JS_UNDEFINED))
	defer m.XJS_FreeValue(ctx, encoded)
	if got := valueString(t, m, ctx, encoded); got != `{"answer":42}` {
		t.Fatalf("JSONStringify = %s", got)
	}
}

func TestRawArrayBufferAndBytecode(t *testing.T) {
	m, _, ctx := context(t)
	data := []byte{0, 1, 128, 255}
	p := alloc(t, m, ctx, int32(len(data)))
	copy((*m.Xmemory().Slice())[uint32(p):], data)
	v := requireValue(t, m, ctx, m.XJS_NewArrayBufferCopy(ctx, p, int32(len(data))))
	defer m.XJS_FreeValue(ctx, v)
	size := alloc(t, m, ctx, 4)
	buf := m.XJS_GetArrayBuffer(ctx, size, v)
	if buf == 0 || read32(m, size) != int32(len(data)) || !bytes.Equal((*m.Xmemory().Slice())[uint32(buf):uint32(buf)+uint32(len(data))], data) {
		t.Fatal("ArrayBuffer round trip failed")
	}
	source := "21 * 2"
	compiled := requireValue(t, m, ctx, m.XJS_Eval(ctx, cstring(t, m, ctx, source), int32(len(source)), cstring(t, m, ctx, "bytecode.js"), qjs.JS_EVAL_FLAG_COMPILE_ONLY))
	defer m.XJS_FreeValue(ctx, compiled)
	serialized := m.XJS_WriteObject(ctx, size, compiled, qjs.JS_WRITE_OBJ_BYTECODE)
	if serialized == 0 {
		t.Fatal("WriteObject returned NULL")
	}
	defer m.Xjs_free(ctx, serialized)
	loaded := requireValue(t, m, ctx, m.XJS_ReadObject(ctx, serialized, read32(m, size), qjs.JS_READ_OBJ_BYTECODE))
	result := requireValue(t, m, ctx, m.XJS_EvalFunction(ctx, loaded)) // consumes loaded
	defer m.XJS_FreeValue(ctx, result)
	if m.XJS_VALUE_GET_INT(result) != 42 {
		t.Fatal("bytecode round trip failed")
	}
}

func TestVariadicThrowABI(t *testing.T) {
	m, _, ctx := context(t)
	args := alloc(t, m, ctx, 16)
	// wasm32 varargs align i32/pointers to 4 bytes and double/i64 to 8.
	write32(m, args, 42)
	write32(m, args+4, cstring(t, m, ctx, "answer"))
	binary.LittleEndian.PutUint64((*m.Xmemory().Slice())[uint32(args+8):], math.Float64bits(1.5))
	v := m.XJS_ThrowTypeError(ctx, cstring(t, m, ctx, "%d %s %.1f"), args)
	if m.XJS_IsException(v) == 0 {
		t.Fatal("ThrowTypeError did not return JS_EXCEPTION")
	}
	exc := m.XJS_GetException(ctx)
	defer m.XJS_FreeValue(ctx, exc)
	if got := valueString(t, m, ctx, exc); got != "TypeError: 42 answer 1.5" {
		t.Fatalf("varargs mismatch: %q", got)
	}
}

func TestMultiValueConversions(t *testing.T) {
	m, _, ctx := context(t)
	for _, tc := range []struct {
		name, source string
		convert      func(int32, int64) (int32, int64)
		want         int64
	}{
		{"int32", "-42", func(ctx int32, v int64) (int32, int64) {
			s, n := m.XJS_ToInt32MV(ctx, v)
			return s, int64(n)
		}, -42},
		{"uint32", "4294967295", func(ctx int32, v int64) (int32, int64) {
			s, n := m.XJS_ToUint32MV(ctx, v)
			return s, int64(uint32(n))
		}, 4294967295},
		{"int64", "-(2 ** 40)", m.XJS_ToInt64MV, -1 << 40},
		{"index", "2 ** 40", m.XJS_ToIndexMV, 1 << 40},
		{"bigint64", "-(2n ** 60n)", m.XJS_ToBigInt64MV, -1 << 60},
		{"int64ext", "2n ** 60n", m.XJS_ToInt64ExtMV, 1 << 60},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := eval(t, m, ctx, tc.source)
			defer m.XJS_FreeValue(ctx, v)
			if status, got := tc.convert(ctx, v); status != 0 || got != tc.want {
				t.Fatalf("conversion = (%d, %d), want (0, %d)", status, got, tc.want)
			}
			bad := eval(t, m, ctx, "Symbol('bad')")
			defer m.XJS_FreeValue(ctx, bad)
			if status, got := tc.convert(ctx, bad); status != -1 || m.XJS_HasException(ctx) == 0 {
				t.Fatalf("failed conversion = (%d, %d)", status, got)
			}
			m.XJS_FreeValue(ctx, m.XJS_GetException(ctx))
		})
	}
	for _, want := range []float64{1.25, math.Copysign(0, -1), math.NaN(), math.Inf(1)} {
		v := m.XJS_NewFloat64(ctx, want)
		status, got := m.XJS_ToFloat64MV(ctx, v)
		m.XJS_FreeValue(ctx, v)
		if status != 0 || (math.Float64bits(got) != math.Float64bits(want) && !(math.IsNaN(got) && math.IsNaN(want))) {
			t.Fatalf("float conversion = (%d, %v), want %v", status, got, want)
		}
	}
	bad := eval(t, m, ctx, "Symbol('bad')")
	defer m.XJS_FreeValue(ctx, bad)
	if status, got := m.XJS_ToFloat64MV(ctx, bad); status != -1 || m.XJS_HasException(ctx) == 0 {
		t.Fatalf("failed float conversion = (%d, %v)", status, got)
	}
	m.XJS_FreeValue(ctx, m.XJS_GetException(ctx))
}

func TestMultiValueReentrantConversion(t *testing.T) {
	m, _, ctx := context(t)
	callback := registerCallback(m, func(gotCtx int32, this int64, argc, argv int32) int64 {
		// Re-enter an MV adapter and grow memory while the outer output is on
		// the wasm stack. Neither call may overwrite the other's output.
		status, n := m.XJS_ToInt32MV(gotCtx, m.XJS_NewInt32(gotCtx, 40))
		if status != 0 || n != 40 {
			t.Fatal("nested conversion failed")
		}
		p := m.Xjs_malloc(gotCtx, 4<<20)
		if p == 0 {
			t.Fatal("memory growth allocation failed")
		}
		m.Xjs_free(gotCtx, p)
		return m.XJS_NewInt32(gotCtx, n+2)
	})
	fn := requireValue(t, m, ctx, m.XJS_NewCFunction(ctx, callback, cstring(t, m, ctx, "valueOf"), 0))
	obj := requireValue(t, m, ctx, m.XJS_NewObject(ctx))
	defer m.XJS_FreeValue(ctx, obj)
	if m.XJS_SetPropertyStr(ctx, obj, cstring(t, m, ctx, "valueOf"), fn) < 0 {
		t.Fatal("setting valueOf failed")
	}
	if status, got := m.XJS_ToFloat64MV(ctx, obj); status != 0 || got != 42 {
		t.Fatalf("reentrant conversion = (%d, %v)", status, got)
	}
}

func TestMultiValueStrings(t *testing.T) {
	m, _, ctx := context(t)
	want := "hello\x00世界🌍"
	v := eval(t, m, ctx, `'hello\0世界🌍'`)
	defer m.XJS_FreeValue(ctx, v)
	atom := m.XJS_ValueToAtom(ctx, v)
	defer m.XJS_FreeAtom(ctx, atom)
	for _, convert := range []func() (int32, int32){
		func() (int32, int32) { return m.XJS_ToCStringLenMV(ctx, v) },
		func() (int32, int32) { return m.XJS_ToCStringLen2MV(ctx, v, 0) },
		func() (int32, int32) { return m.XJS_AtomToCStringLenMV(ctx, atom) },
	} {
		p, n := convert()
		if p == 0 {
			t.Fatal("string conversion failed")
		}
		got := string((*m.Xmemory().Slice())[uint32(p) : uint32(p)+uint32(n)])
		m.XJS_FreeCString(ctx, p)
		if got != want {
			t.Fatalf("string = %q, want %q", got, want)
		}
	}
	p, n := m.XJS_ToCStringLen2MV(ctx, v, 1)
	if p == 0 || n != int32(len(want)+2) { // CESU-8 encodes a surrogate pair as six bytes.
		t.Fatalf("CESU-8 conversion = (%d, %d)", p, n)
	}
	m.XJS_FreeCString(ctx, p)
	bad := eval(t, m, ctx, "Symbol('bad')")
	defer m.XJS_FreeValue(ctx, bad)
	if p, n := m.XJS_ToCStringLenMV(ctx, bad); p != 0 || n != 0 || m.XJS_HasException(ctx) == 0 {
		t.Fatalf("failed string conversion = (%d, %d)", p, n)
	}
	m.XJS_FreeValue(ctx, m.XJS_GetException(ctx))
}

func TestMultiValueSerializationFailure(t *testing.T) {
	m, _, ctx := context(t)
	badValue := eval(t, m, ctx, "Symbol('unserializable')")
	defer m.XJS_FreeValue(ctx, badValue)
	if p, n := m.XJS_WriteObjectMV(ctx, badValue, 0); p != 0 || n != 0 || m.XJS_HasException(ctx) == 0 {
		t.Fatal("invalid serialization failure")
	}
	m.XJS_FreeValue(ctx, m.XJS_GetException(ctx))
	if p, n, tab, count := m.XJS_WriteObject2MV(ctx, badValue, 0); p != 0 || n != 0 || tab != 0 || count != 0 || m.XJS_HasException(ctx) == 0 {
		t.Fatal("invalid extended serialization failure")
	}
	m.XJS_FreeValue(ctx, m.XJS_GetException(ctx))
}

func TestMultiValueBuffersAndSerialization(t *testing.T) {
	m, _, ctx := context(t)
	v := eval(t, m, ctx, "new Uint16Array(new ArrayBuffer(16), 4, 3)")
	defer m.XJS_FreeValue(ctx, v)
	buf, offset, length, width := m.XJS_GetTypedArrayBufferMV(ctx, v)
	requireValue(t, m, ctx, buf)
	defer m.XJS_FreeValue(ctx, buf)
	if offset != 4 || length != 6 || width != 2 {
		t.Fatalf("typed array metadata = (%d, %d, %d)", offset, length, width)
	}
	if p, n := m.XJS_GetArrayBufferMV(ctx, buf); p == 0 || n != 16 {
		t.Fatalf("array buffer = (%d, %d)", p, n)
	}
	obj := eval(t, m, ctx, "({answer: 42})")
	defer m.XJS_FreeValue(ctx, obj)
	for _, serialize := range []func() (int32, int32){
		func() (int32, int32) { return m.XJS_WriteObjectMV(ctx, obj, 0) },
		func() (int32, int32) {
			p, n, tab, count := m.XJS_WriteObject2MV(ctx, obj, 0)
			defer m.Xjs_free(ctx, tab)
			if count != 0 {
				t.Fatalf("unexpected shared buffers: %d", count)
			}
			return p, n
		},
	} {
		p, n := serialize()
		if p == 0 || n == 0 {
			t.Fatal("serialization failed")
		}
		loaded := requireValue(t, m, ctx, m.XJS_ReadObject(ctx, p, n, 0))
		m.Xjs_free(ctx, p)
		answer := requireValue(t, m, ctx, m.XJS_GetPropertyStr(ctx, loaded, cstring(t, m, ctx, "answer")))
		m.XJS_FreeValue(ctx, loaded)
		if status, n := m.XJS_ToInt32MV(ctx, answer); status != 0 || n != 42 {
			t.Fatalf("serialized answer = (%d, %d)", status, n)
		}
		m.XJS_FreeValue(ctx, answer)
	}
	bad, offset, length, width := m.XJS_GetTypedArrayBufferMV(ctx, qjs.JS_NULL)
	if m.XJS_IsException(bad) == 0 || offset != 0 || length != 0 || width != 0 {
		t.Fatal("invalid typed array did not return exception and zero metadata")
	}
	m.XJS_FreeValue(ctx, m.XJS_GetException(ctx))
	if p, n := m.XJS_GetArrayBufferMV(ctx, qjs.JS_NULL); p != 0 || n != 0 {
		t.Fatalf("invalid array buffer = (%d, %d)", p, n)
	}
	m.XJS_FreeValue(ctx, m.XJS_GetException(ctx))
}

func TestMultiValueProperties(t *testing.T) {
	m, _, ctx := context(t)
	obj := eval(t, m, ctx, "({answer: {n: 42}, get accessor() { return 42; }})")
	defer m.XJS_FreeValue(ctx, obj)
	status, tab, count := m.XJS_GetOwnPropertyNamesMV(ctx, obj, qjs.JS_GPN_STRING_MASK|qjs.JS_GPN_SET_ENUM)
	if status != 0 || count != 2 {
		t.Fatalf("property names = (%d, %d, %d)", status, tab, count)
	}
	defer m.XJS_FreePropertyEnum(ctx, tab, count)
	for _, name := range []string{"answer", "accessor", "missing"} {
		atom := m.XJS_NewAtom(ctx, cstring(t, m, ctx, name))
		status, flags, value, getter, setter := m.XJS_GetOwnPropertyMV(ctx, obj, atom)
		m.XJS_FreeAtom(ctx, atom)
		switch name {
		case "answer":
			if status != 1 || flags != qjs.JS_PROP_C_W_E || m.XJS_IsObject(value) == 0 || getter != qjs.JS_UNDEFINED || setter != qjs.JS_UNDEFINED {
				t.Fatal("invalid data property descriptor")
			}
		case "accessor":
			if status != 1 || value != qjs.JS_UNDEFINED || m.XJS_IsFunction(ctx, getter) == 0 || setter != qjs.JS_UNDEFINED {
				t.Fatal("invalid accessor property descriptor")
			}
		case "missing":
			if status != 0 || flags != 0 || value != qjs.JS_UNDEFINED || getter != qjs.JS_UNDEFINED || setter != qjs.JS_UNDEFINED {
				t.Fatal("invalid missing property descriptor")
			}
		}
		m.XJS_FreeValue(ctx, value)
		m.XJS_FreeValue(ctx, getter)
		m.XJS_FreeValue(ctx, setter)
	}
	if status, tab, count := m.XJS_GetOwnPropertyNamesMV(ctx, qjs.JS_NULL, 0); status != -1 || tab != 0 || count != 0 {
		t.Fatal("invalid property enumeration failure")
	}
	m.XJS_FreeValue(ctx, m.XJS_GetException(ctx))
	if status, _, _, _, _ := m.XJS_GetOwnPropertyMV(ctx, qjs.JS_NULL, 0); status != -1 {
		t.Fatal("invalid property descriptor failure")
	}
	m.XJS_FreeValue(ctx, m.XJS_GetException(ctx))
}

func TestMultiValuePromiseAndJobs(t *testing.T) {
	m, rt, ctx := context(t)
	promise, resolve, reject := m.XJS_NewPromiseCapabilityMV(ctx)
	requireValue(t, m, ctx, promise)
	defer m.XJS_FreeValue(ctx, promise)
	defer m.XJS_FreeValue(ctx, resolve)
	defer m.XJS_FreeValue(ctx, reject)
	if m.XJS_IsFunction(ctx, resolve) == 0 || m.XJS_IsFunction(ctx, reject) == 0 {
		t.Fatal("resolving functions are not callable")
	}
	requireValue(t, m, ctx, m.XJS_Call(ctx, resolve, qjs.JS_UNDEFINED, 0, 0))
	if m.XJS_PromiseState(ctx, promise) != qjs.JS_PROMISE_FULFILLED {
		t.Fatal("promise did not resolve")
	}
	for _, fail := range []bool{false, true} {
		job := registerCallback(m, func(gotCtx, argc, argv int32) int64 {
			if gotCtx != ctx || argc != 0 {
				t.Fatal("incorrect job arguments")
			}
			if fail {
				return m.XJS_Throw(ctx, qjs.JS_NULL)
			}
			return qjs.JS_UNDEFINED
		})
		if m.XJS_EnqueueJob(ctx, job, 0, 0) != 0 {
			t.Fatal("enqueue failed")
		}
		want := int32(1)
		if fail {
			want = -1
		}
		if status, gotCtx := m.XJS_ExecutePendingJobMV(rt); status != want || gotCtx != ctx {
			t.Fatalf("job result = (%d, %d), want (%d, %d)", status, gotCtx, want, ctx)
		}
		if fail {
			m.XJS_FreeValue(ctx, m.XJS_GetException(ctx))
		}
	}
	if status, gotCtx := m.XJS_ExecutePendingJobMV(rt); status != 0 || gotCtx != 0 {
		t.Fatalf("empty job result = (%d, %d)", status, gotCtx)
	}
}

func TestMultiValueReallocAndOpaque(t *testing.T) {
	m, rt, ctx := context(t)
	p, slack := m.Xjs_realloc2MV(ctx, 0, 32)
	if p == 0 || slack < 0 {
		t.Fatalf("realloc = (%d, %d)", p, slack)
	}
	if got := m.Xjs_malloc_usable_size(ctx, p); got != 32+slack {
		t.Fatalf("usable size = %d, want %d", got, 32+slack)
	}
	m.Xjs_free(ctx, p)
	id := m.XJS_NewClassID(alloc(t, m, ctx, 4))
	def := alloc(t, m, ctx, qjs.SizeofJSClassDef)
	write32(m, def+qjs.OffsetofJSClassDef_class_name, cstring(t, m, ctx, "OpaqueClass"))
	if m.XJS_NewClass(rt, id, def) != 0 {
		t.Fatal("class registration failed")
	}
	obj := requireValue(t, m, ctx, m.XJS_NewObjectClass(ctx, id))
	defer m.XJS_FreeValue(ctx, obj)
	m.XJS_SetOpaque(obj, 1234)
	if ptr, class := m.XJS_GetAnyOpaqueMV(obj); ptr != 1234 || class != m.XJS_GetClassID(obj) {
		t.Fatalf("opaque = (%d, %d)", ptr, class)
	}
	if ptr, class := m.XJS_GetAnyOpaqueMV(qjs.JS_NULL); ptr != 0 || class != 0 {
		t.Fatalf("invalid opaque = (%d, %d)", ptr, class)
	}
}
