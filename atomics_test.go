package quickjs_wasm_test

import (
	"testing"
	"testing/synctest"

	qjs "github.com/tosuke/go-quickjs-wasm"
)

func TestSharedArrayBufferAtomics(t *testing.T) {
	m, _, ctx := context(t)
	v := eval(t, m, ctx, `
        globalThis.shared = new SharedArrayBuffer(64);
        let results = [];
        for (const Type of [Int8Array, Uint8Array, Int16Array, Uint16Array, Int32Array, Uint32Array, BigInt64Array, BigUint64Array]) {
            const a = new Type(shared);
            const n = x => Type === BigInt64Array || Type === BigUint64Array ? BigInt(x) : x;
            Atomics.store(a, 0, n(7));
            results.push(Atomics.add(a, 0, n(3)) == n(7));
            results.push(Atomics.sub(a, 0, n(1)) == n(10));
            results.push(Atomics.and(a, 0, n(7)) == n(9));
            results.push(Atomics.or(a, 0, n(6)) == n(1));
            results.push(Atomics.xor(a, 0, n(3)) == n(7));
            results.push(Atomics.exchange(a, 0, n(12)) == n(4));
            results.push(Atomics.compareExchange(a, 0, n(0), n(99)) == n(12));
            results.push(Atomics.compareExchange(a, 0, n(12), n(42)) == n(12));
            results.push(Atomics.load(a, 0) == n(42));
        }
        results.every(Boolean)
    `)
	defer m.JS_FreeValue(ctx, v)
	if got := valueString(t, m, ctx, v); got != "true" {
		t.Fatalf("Atomics operations = %s", got)
	}
	// Retain a SharedArrayBuffer while the enclosing Wasm memory grows.
	v = eval(t, m, ctx, `new Uint8Array(8 * 1024 * 1024).fill(1); Atomics.add(new Int32Array(shared), 0, 1)`)
	defer m.JS_FreeValue(ctx, v)
	if got := valueString(t, m, ctx, v); got != "42" {
		t.Fatalf("SharedArrayBuffer after growth = %s", got)
	}
}

func TestAtomicsWait(t *testing.T) {
	m, rt, ctx := context(t)
	check := func(source, want string) {
		t.Helper()
		v := eval(t, m, ctx, source)
		defer m.JS_FreeValue(ctx, v)
		if got := valueString(t, m, ctx, v); got != want {
			t.Fatalf("%s = %s, want %s", source, got, want)
		}
	}
	check(`globalThis.a = new Int32Array(new SharedArrayBuffer(8)); try { Atomics.wait(a, 0, 0, 0) } catch (e) { e instanceof TypeError }`, "true")
	m.JS_SetCanBlock(rt, true)
	check(`Atomics.wait(a, 0, 1)`, "not-equal")
	check(`Atomics.wait(a, 0, 0, 0)`, "timed-out")
	check(`Atomics.wait(a, 0, 0, 1)`, "timed-out")
	check(`Atomics.wait(new BigInt64Array(a.buffer), 0, 1n)`, "not-equal")
	check(`Atomics.wait(new BigInt64Array(a.buffer), 0, 0n, 0)`, "timed-out")
	check(`Atomics.notify(a, 0)`, "0")
	check(`Atomics.notify(new Int32Array(1), 0)`, "0")
	m.JS_SetCanBlock(rt, false)
	check(`try { Atomics.wait(a, 0, 0, 0) } catch (e) { e instanceof TypeError }`, "true")
}

func TestAtomicsWaitNotify(t *testing.T) {
	for _, array := range []string{"Int32Array", "BigInt64Array"} {
		t.Run(array, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				m, rt, ctx := context(t)
				m.JS_SetCanBlock(rt, true)
				buf := eval(t, m, ctx, "globalThis.a = new "+array+"(new SharedArrayBuffer(8)); a.buffer")
				defer m.JS_FreeValue(ctx, buf)
				ptr, _ := m.JS_GetArrayBuffer(ctx, buf)
				// Use a separate leaf C adapter so notify does not reenter QuickJS.
				notifier := newTestModule(t, m)
				source := "Atomics.wait(a, 0, "
				if array == "BigInt64Array" {
					source += "0n)"
				} else {
					source += "0)"
				}
				code, file := cstring(t, m, ctx, source), cstring(t, m, ctx, "wait.js")
				done := make(chan qjs.JSValue, 1)
				go func() { done <- m.JS_Eval(ctx, code, int32(len(source)), file, 0) }()
				synctest.Wait()
				if got := notifier.Xtest_atomic_notify(ptr, 1); got != 1 {
					t.Fatalf("notify woke %d waiters, want 1", got)
				}
				v := <-done
				defer m.JS_FreeValue(ctx, v)
				if got := valueString(t, m, ctx, v); got != "ok" {
					t.Fatalf("Atomics.wait = %s, want ok", got)
				}
			})
		})
	}
}
