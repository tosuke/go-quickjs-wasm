package quickjs_wasm_test

import (
	"bytes"
	"math"
	"testing"

	qjs "github.com/tosuke/go-quickjs-wasm"
)

func TestHeaderNewBool(t *testing.T) {
	m := newTestModule(t, module(t))
	for _, tc := range []struct {
		value bool
		c     int32
	}{{false, 0}, {true, 1}} {
		if got, want := qjs.JS_NewBool(tc.value), qjs.JSValue(m.XJS_NewBool(0, tc.c)); got != want {
			t.Errorf("JS_NewBool(%t) = %#x, QuickJS = %#x", tc.value, uint64(got), uint64(want))
		}
	}
}

func TestHeaderValueGetters(t *testing.T) {
	m := newTestModule(t, module(t))
	for _, payload := range []int32{math.MinInt32, -1, 0, 1, math.MaxInt32} {
		v := qjs.JS_MKVAL(qjs.JS_TAG_INT, payload)
		if got, want := qjs.JS_VALUE_GET_INT(v), m.XJS_VALUE_GET_INT(int64(v)); got != want {
			t.Errorf("JS_VALUE_GET_INT(%#x) = %d, QuickJS = %d", uint64(v), got, want)
		}
		v = qjs.JS_MKVAL(qjs.JS_TAG_BOOL, payload)
		if got, want := qjs.JS_VALUE_GET_BOOL(v), m.XJS_VALUE_GET_BOOL(int64(v)) != 0; got != want {
			t.Errorf("JS_VALUE_GET_BOOL(%#x) = %t, QuickJS = %t", uint64(v), got, want)
		}
		v = qjs.JS_MKVAL(qjs.JS_TAG_SHORT_BIG_INT, payload)
		if got, want := qjs.JS_VALUE_GET_SHORT_BIG_INT(v), m.XJS_VALUE_GET_SHORT_BIG_INT(int64(v)); got != want {
			t.Errorf("JS_VALUE_GET_SHORT_BIG_INT(%#x) = %d, QuickJS = %d", uint64(v), got, want)
		}
	}
}

func TestHeaderPredicates(t *testing.T) {
	m := newTestModule(t, module(t))
	values := []qjs.JSValue{qjs.JS_NAN, qjs.JS_NewFloat64(1.5), qjs.JS_NewFloat64(-1.5)}
	for tag := int32(-10); tag <= 9; tag++ {
		values = append(values, qjs.JS_MKVAL(tag, 0), qjs.JS_MKVAL(tag, -1))
	}

	checks := []struct {
		name string
		goFn func(qjs.JSValue) bool
		cFn  func(int64) int32
	}{
		{"JS_IsNumber", qjs.JS_IsNumber, m.XJS_IsNumber},
		{"JS_IsBigInt", qjs.JS_IsBigInt, func(v int64) int32 { return m.XJS_IsBigInt(0, v) }},
		{"JS_IsBool", qjs.JS_IsBool, m.XJS_IsBool},
		{"JS_IsNull", qjs.JS_IsNull, m.XJS_IsNull},
		{"JS_IsUndefined", qjs.JS_IsUndefined, m.XJS_IsUndefined},
		{"JS_IsException", qjs.JS_IsException, m.XJS_IsException},
		{"JS_IsUninitialized", qjs.JS_IsUninitialized, m.XJS_IsUninitialized},
		{"JS_IsString", qjs.JS_IsString, m.XJS_IsString},
		{"JS_IsSymbol", qjs.JS_IsSymbol, m.XJS_IsSymbol},
		{"JS_IsObject", qjs.JS_IsObject, m.XJS_IsObject},
		{"JS_VALUE_IS_NAN", qjs.JS_VALUE_IS_NAN, m.XJS_VALUE_IS_NAN},
	}
	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			for _, v := range values {
				if got, want := check.goFn(v), check.cFn(int64(v)) != 0; got != want {
					t.Errorf("%s(%#x) = %t, QuickJS = %t", check.name, uint64(v), got, want)
				}
			}
		})
	}
	for _, a := range values {
		for _, b := range values {
			if got, want := qjs.JS_VALUE_IS_BOTH_INT(a, b), m.XJS_VALUE_IS_BOTH_INT(int64(a), int64(b)) != 0; got != want {
				t.Errorf("JS_VALUE_IS_BOTH_INT(%#x, %#x) = %t, QuickJS = %t", uint64(a), uint64(b), got, want)
			}
			if got, want := qjs.JS_VALUE_IS_BOTH_FLOAT(a, b), m.XJS_VALUE_IS_BOTH_FLOAT(int64(a), int64(b)) != 0; got != want {
				t.Errorf("JS_VALUE_IS_BOTH_FLOAT(%#x, %#x) = %t, QuickJS = %t", uint64(a), uint64(b), got, want)
			}
		}
	}
}

func FuzzHeaderIntegerConstructors(f *testing.F) {
	for _, v := range []int64{
		math.MinInt64, math.MinInt32 - 1, math.MinInt32, -1, 0, 1,
		math.MaxInt32, math.MaxInt32 + 1, math.MaxUint32,
		1<<53 - 1, 1 << 53, 1<<53 + 1, math.MaxInt64,
	} {
		f.Add(v)
	}

	// These header functions do not use the context argument.
	m := newTestModule(f, module(f))
	f.Fuzz(func(t *testing.T, v int64) {
		if got, want := qjs.JS_NewInt64(v), qjs.JSValue(m.XJS_NewInt64(0, v)); got != want {
			t.Errorf("JS_NewInt64(%d) = %#x, QuickJS = %#x", v, uint64(got), uint64(want))
		}
		i := int32(v)
		if got, want := qjs.JS_NewInt32(i), qjs.JSValue(m.XJS_NewInt32(0, i)); got != want {
			t.Errorf("JS_NewInt32(%d) = %#x, QuickJS = %#x", i, uint64(got), uint64(want))
		}
		if got, want := qjs.JS_NewCatchOffset(i), qjs.JSValue(m.XJS_NewCatchOffset(0, i)); got != want {
			t.Errorf("JS_NewCatchOffset(%d) = %#x, QuickJS = %#x", i, uint64(got), uint64(want))
		}
		u := uint32(v)
		if got, want := qjs.JS_NewUint32(u), qjs.JSValue(m.XJS_NewUint32(0, int32(u))); got != want {
			t.Errorf("JS_NewUint32(%d) = %#x, QuickJS = %#x", u, uint64(got), uint64(want))
		}
	})
}

func FuzzHeaderFloat64(f *testing.F) {
	for _, bits := range []uint64{
		0, 1 << 63, 1, 0x8000000000000001,
		math.Float64bits(math.MinInt32), math.Float64bits(math.MaxInt32),
		math.Float64bits(math.MinInt32 - 1), math.Float64bits(math.MaxInt32 + 1),
		math.Float64bits(math.Nextafter(math.MinInt32, math.Inf(-1))),
		math.Float64bits(math.Nextafter(math.MaxInt32, math.Inf(1))),
		math.Float64bits(1.5), math.Float64bits(math.Inf(1)), math.Float64bits(math.Inf(-1)),
		0x7ff0000000000001, 0x7ff8000000000000, 0xfff8000000000001,
	} {
		f.Add(bits)
	}

	m := newTestModule(f, module(f))
	f.Fuzz(func(t *testing.T, bits uint64) {
		d := math.Float64frombits(bits)
		got, want := qjs.JS_NewFloat64(d), qjs.JSValue(m.XJS_NewFloat64(0, d))
		if got != want {
			t.Fatalf("JS_NewFloat64(%016x) = %#x, QuickJS = %#x", bits, uint64(got), uint64(want))
		}
		if qjs.JS_VALUE_GET_TAG(got) == qjs.JS_TAG_INT {
			if decoded := float64(qjs.JS_VALUE_GET_INT(got)); math.Float64bits(decoded) != bits {
				t.Errorf("integer round trip = %016x, want %016x", math.Float64bits(decoded), bits)
			}
		} else if decoded := qjs.JS_VALUE_GET_FLOAT64(got); math.IsNaN(d) {
			if !math.IsNaN(decoded) || got != qjs.JS_NAN {
				t.Errorf("NaN round trip = %016x, value = %#x", math.Float64bits(decoded), uint64(got))
			}
		} else if math.Float64bits(decoded) != bits {
			t.Errorf("float round trip = %016x, want %016x", math.Float64bits(decoded), bits)
		}
	})
}

func FuzzHeaderValueEncoding(f *testing.F) {
	for _, tag := range []int32{math.MinInt32, -10, -9, -1, 0, 7, 8, math.MaxInt32} {
		for _, payload := range []int32{math.MinInt32, -1, 0, 1, math.MaxInt32} {
			f.Add(tag, payload)
		}
	}

	m := newTestModule(f, module(f))
	f.Fuzz(func(t *testing.T, tag, payload int32) {
		v := qjs.JS_MKVAL(tag, payload)
		if want := qjs.JSValue(m.XJS_MKVAL(tag, payload)); v != want {
			t.Errorf("JS_MKVAL(%d, %d) = %#x, QuickJS = %#x", tag, payload, uint64(v), uint64(want))
		}
		if got, want := qjs.JS_MKPTR(tag, payload), qjs.JSValue(m.XJS_MKPTR(tag, payload)); got != want {
			t.Errorf("JS_MKPTR(%d, %#x) = %#x, QuickJS = %#x", tag, uint32(payload), uint64(got), uint64(want))
		}
		if got, c := qjs.JS_VALUE_GET_TAG(v), m.XJS_VALUE_GET_TAG(int64(v)); got != c || got != tag {
			t.Errorf("JS_VALUE_GET_TAG(%#x) = %d, QuickJS = %d, want %d", uint64(v), got, c, tag)
		}
		if got, c := qjs.JS_VALUE_GET_PTR(v), m.XJS_VALUE_GET_PTR(int64(v)); got != c || got != payload {
			t.Errorf("JS_VALUE_GET_PTR(%#x) = %#x, QuickJS = %#x, want %#x", uint64(v), uint32(got), uint32(c), uint32(payload))
		}
		if got, want := math.Float64bits(qjs.JS_VALUE_GET_FLOAT64(v)), math.Float64bits(m.XJS_VALUE_GET_FLOAT64(int64(v))); got != want {
			t.Errorf("JS_VALUE_GET_FLOAT64(%#x) = %016x, QuickJS = %016x", uint64(v), got, want)
		}
		if got, want := qjs.JS_VALUE_GET_NORM_TAG(v), m.XJS_VALUE_GET_NORM_TAG(int64(v)); got != want {
			t.Errorf("JS_VALUE_GET_NORM_TAG(%#x) = %d, QuickJS = %d", uint64(v), got, want)
		}
		if got, want := qjs.JS_TAG_IS_FLOAT64(tag), m.XJS_TAG_IS_FLOAT64(tag) != 0; got != want {
			t.Errorf("JS_TAG_IS_FLOAT64(%d) = %t, QuickJS = %t", tag, got, want)
		}
		if got, want := qjs.JS_VALUE_HAS_REF_COUNT(v), m.XJS_VALUE_HAS_REF_COUNT(int64(v)) != 0; got != want {
			t.Errorf("JS_VALUE_HAS_REF_COUNT(%#x) = %t, QuickJS = %t", uint64(v), got, want)
		}
	})
}

func FuzzPrintValueOptionsBitfields(f *testing.F) {
	for _, hidden := range []int8{-128, -127, -1, 0, 1, 126, 127} {
		for _, raw := range []int8{-128, -127, -1, 0, 1, 126, 127} {
			f.Add([]byte{0x00, 0xff, 0x55, 0xaa, 0x80, 0x7f, 0x01, 0xfe}, hidden, raw)
		}
	}
	f.Add([]byte{}, int8(0), int8(0))
	f.Add([]byte{0xff}, int8(-1), int8(-128))

	m := module(f)
	c := newTestModule(f, m)
	const guard = 8
	const blockSize = guard + qjs.JSPrintValueOptionsSize + guard
	allocate := func() int32 {
		p := m.Xmalloc(blockSize)
		if p == 0 {
			f.Fatal("malloc returned NULL")
		}
		f.Cleanup(func() { m.Xfree(p) })
		return p
	}
	goBase, cBase := allocate(), allocate()
	goPtr, cPtr := goBase+guard, cBase+guard

	f.Fuzz(func(t *testing.T, initial []byte, hidden, raw int8) {
		var initialBlock, goBlock, cBlock [blockSize]byte
		for i := range initialBlock {
			if len(initial) != 0 {
				initialBlock[i] = initial[i%len(initial)]
			}
		}
		fields := []struct {
			name    string
			address func(qjs.Ptr[qjs.JSPrintValueOptions]) qjs.Ptr[int8]
			read    func(int32) int32
			write   func(int32, int32)
			value   int8
		}{
			{"show_hidden", qjs.JSPrintValueOptions_show_hidden, c.XJSPrintValueOptions_show_hidden_read, c.XJSPrintValueOptions_show_hidden_write, hidden},
			{"raw_dump", qjs.JSPrintValueOptions_raw_dump, c.XJSPrintValueOptions_raw_dump_read, c.XJSPrintValueOptions_raw_dump_write, raw},
		}
		for _, field := range fields {
			// Randomize the other bitfield, padding, ordinary members, and guards.
			// Reset for each field so an individual store must preserve its neighbors.
			m.StoreBytes(goBase, initialBlock[:])
			m.StoreBytes(cBase, initialBlock[:])
			address := field.address(goPtr)
			if got, want := int32(m.LoadI8(address)), field.read(int32(goPtr)); got != want {
				t.Fatalf("%s initial Go read = %d, C read = %d", field.name, got, want)
			}

			field.write(int32(cPtr), int32(field.value))
			if got := m.LoadI8(field.address(cPtr)); got != field.value {
				t.Fatalf("%s Go read after C write = %d, want %d", field.name, got, field.value)
			}
			m.StoreI8(address, field.value)
			if got := field.read(int32(goPtr)); got != int32(field.value) {
				t.Fatalf("%s C read after Go write = %d, want %d", field.name, got, field.value)
			}
			m.LoadBytes(goBlock[:], goBase, len(goBlock))
			m.LoadBytes(cBlock[:], cBase, len(cBlock))
			if !bytes.Equal(goBlock[:], cBlock[:]) {
				t.Fatalf("%s store differs (including neighbors and guards):\nGo %x\nC  %x", field.name, goBlock, cBlock)
			}
		}
	})
}
