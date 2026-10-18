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
