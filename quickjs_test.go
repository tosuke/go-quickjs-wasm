package quickjs_wasm_test

import (
	"testing"

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
			defer m.JS_FreeValue(ctx, v)
			if got := valueString(t, m, ctx, v); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestException(t *testing.T) {
	m, _, ctx := context(t)
	source := "throw new TypeError('boom')"
	v := m.JS_Eval(ctx, cstring(t, m, ctx, source), int32(len(source)), cstring(t, m, ctx, "throw.js"), 0)
	if !qjs.JS_IsException(v) {
		m.JS_FreeValue(ctx, v)
		t.Fatal("expected exception")
	}
	exc := m.JS_GetException(ctx)
	defer m.JS_FreeValue(ctx, exc)
	if got := valueString(t, m, ctx, exc); got != "TypeError: boom" {
		t.Fatalf("unexpected exception: %q", got)
	}
	// Retrieving an exception must leave the context usable.
	v = eval(t, m, ctx, "40 + 2")
	m.JS_FreeValue(ctx, v)
}
