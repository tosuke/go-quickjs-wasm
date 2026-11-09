/* Raw wasm32 adapters for quickjs.h inline functions and macros. */
#include "quickjs.h"
#include <math.h>
#include <stdlib.h>
#include <stdatomic.h>

/* SAB payloads can outlive the runtime that created them. Only this refcount
   and the backing bytes are shared; JS objects remain runtime-local. */
typedef struct {
    _Atomic(uint32_t) refs;
    uint32_t padding[3];
} SharedBufferHeader;

static void *shared_buffer_alloc(void *opaque, size_t size)
{
    if (size > SIZE_MAX - sizeof(SharedBufferHeader)) return NULL;
    SharedBufferHeader *h = malloc(sizeof(*h) + size);
    if (!h) return NULL;
    atomic_init(&h->refs, 1);
    return h + 1;
}

static void shared_buffer_dup(void *opaque, void *ptr)
{
    SharedBufferHeader *h = (SharedBufferHeader *)ptr - 1;
    atomic_fetch_add_explicit(&h->refs, 1, memory_order_relaxed);
}

static void shared_buffer_free(void *opaque, void *ptr)
{
    SharedBufferHeader *h = (SharedBufferHeader *)ptr - 1;
    if (atomic_fetch_sub_explicit(&h->refs, 1, memory_order_acq_rel) == 1)
        free(h);
}

__attribute__((export_name("qjs_set_default_shared_array_buffer_functions")))
void qjs_set_default_shared_array_buffer_functions(JSRuntime *rt)
{
    const JSSharedArrayBufferFunctions functions = {
        shared_buffer_alloc, shared_buffer_free, shared_buffer_dup, NULL
    };
    JS_SetSharedArrayBufferFunctions(rt, &functions);
}

__attribute__((export_name("JS_FreeValue")))
void qjs_api_JS_FreeValue(JSContext * ctx, JSValue v) { JS_FreeValue(ctx, v); }
__attribute__((export_name("JS_FreeValueRT")))
void qjs_api_JS_FreeValueRT(JSRuntime * rt, JSValue v) { JS_FreeValueRT(rt, v); }
__attribute__((export_name("JS_DupValue")))
JSValue qjs_api_JS_DupValue(JSContext * ctx, JSValue v) { return JS_DupValue(ctx, v); }
__attribute__((export_name("JS_DupValueRT")))
JSValue qjs_api_JS_DupValueRT(JSRuntime * rt, JSValue v) { return JS_DupValueRT(rt, v); }

/* Multi-value adapters: original return value, then output arguments. */
__attribute__((export_name("JS_ComputeMemoryUsage")))
JSMemoryUsage qjs_mv_JS_ComputeMemoryUsage(JSRuntime *rt)
{
	JSMemoryUsage r = {0};
	JS_ComputeMemoryUsage(rt, &r);
	return r;
}

typedef struct { int status; int32_t value; } JS_ToInt32Result;

__attribute__((export_name("JS_ToInt32")))
JS_ToInt32Result qjs_mv_JS_ToInt32(JSContext *ctx, JSValueConst value)
{
    JS_ToInt32Result r = {0};
    r.status = JS_ToInt32(ctx, &r.value, value);
    return r;
}

typedef struct { int status; int64_t value; } JS_ToInt64Result;

__attribute__((export_name("JS_ToInt64")))
JS_ToInt64Result qjs_mv_JS_ToInt64(JSContext *ctx, JSValueConst value)
{
    JS_ToInt64Result r = {0};
    r.status = JS_ToInt64(ctx, &r.value, value);
    return r;
}

typedef struct { int status; uint64_t value; } JS_ToIndexResult;

__attribute__((export_name("JS_ToIndex")))
JS_ToIndexResult qjs_mv_JS_ToIndex(JSContext *ctx, JSValueConst value)
{
    JS_ToIndexResult r = {0};
    r.status = JS_ToIndex(ctx, &r.value, value);
    return r;
}

typedef struct { int status; double value; } JS_ToFloat64Result;

__attribute__((export_name("JS_ToFloat64")))
JS_ToFloat64Result qjs_mv_JS_ToFloat64(JSContext *ctx, JSValueConst value)
{
    JS_ToFloat64Result r = {0};
    r.status = JS_ToFloat64(ctx, &r.value, value);
    return r;
}

typedef struct { int status; int64_t value; } JS_ToBigInt64Result;

__attribute__((export_name("JS_ToBigInt64")))
JS_ToBigInt64Result qjs_mv_JS_ToBigInt64(JSContext *ctx, JSValueConst value)
{
    JS_ToBigInt64Result r = {0};
    r.status = JS_ToBigInt64(ctx, &r.value, value);
    return r;
}

typedef struct { int status; int64_t value; } JS_ToInt64ExtResult;

__attribute__((export_name("JS_ToInt64Ext")))
JS_ToInt64ExtResult qjs_mv_JS_ToInt64Ext(JSContext *ctx, JSValueConst value)
{
    JS_ToInt64ExtResult r = {0};
    r.status = JS_ToInt64Ext(ctx, &r.value, value);
    return r;
}

typedef struct { const char *ptr; size_t len; } CStringResult;

__attribute__((export_name("JS_AtomToCStringLen")))
CStringResult qjs_mv_JS_AtomToCStringLen(JSContext *ctx, JSAtom atom)
{
    CStringResult r = {0};
    r.ptr = JS_AtomToCStringLen(ctx, &r.len, atom);
    return r;
}

__attribute__((export_name("JS_ToCStringLen2")))
CStringResult qjs_mv_JS_ToCStringLen2(JSContext *ctx, JSValueConst value, JS_BOOL cesu8)
{
    CStringResult r = {0};
    r.ptr = JS_ToCStringLen2(ctx, &r.len, value, cesu8);
    return r;
}

typedef struct { void *ptr; size_t size; } BufferResult;

__attribute__((export_name("JS_GetArrayBuffer")))
BufferResult qjs_mv_JS_GetArrayBuffer(JSContext *ctx, JSValueConst obj)
{
    BufferResult r = {0};
    r.ptr = JS_GetArrayBuffer(ctx, &r.size, obj);
    return r;
}

typedef struct { JSValue buffer; size_t byte_offset, byte_length, bytes_per_element; } TypedArrayResult;

__attribute__((export_name("JS_GetTypedArrayBuffer")))
TypedArrayResult qjs_mv_JS_GetTypedArrayBuffer(JSContext *ctx, JSValueConst obj)
{
    TypedArrayResult r = {0};
    r.buffer = JS_GetTypedArrayBuffer(ctx, obj, &r.byte_offset, &r.byte_length, &r.bytes_per_element);
    return r;
}

typedef struct { int status; JSPropertyEnum *tab; uint32_t len; } PropertyNamesResult;

__attribute__((export_name("JS_GetOwnPropertyNames")))
PropertyNamesResult qjs_mv_JS_GetOwnPropertyNames(JSContext *ctx, JSValueConst obj, int flags)
{
    PropertyNamesResult r = {0};
    r.status = JS_GetOwnPropertyNames(ctx, &r.tab, &r.len, obj, flags);
    return r;
}

typedef struct { int status, flags; JSValue value, getter, setter; } PropertyResult;

__attribute__((export_name("JS_GetOwnProperty")))
PropertyResult qjs_mv_JS_GetOwnProperty(JSContext *ctx, JSValueConst obj, JSAtom prop)
{
    JSPropertyDescriptor desc = {0, JS_UNDEFINED, JS_UNDEFINED, JS_UNDEFINED};
    int status = JS_GetOwnProperty(ctx, &desc, obj, prop);
    return (PropertyResult){status, desc.flags, desc.value, desc.getter, desc.setter};
}

typedef struct { JSValue promise, resolve, reject; } PromiseResult;

__attribute__((export_name("JS_NewPromiseCapability")))
PromiseResult qjs_mv_JS_NewPromiseCapability(JSContext *ctx)
{
    JSValue funcs[2] = {JS_UNDEFINED, JS_UNDEFINED};
    JSValue promise = JS_NewPromiseCapability(ctx, funcs);
    return (PromiseResult){promise, funcs[0], funcs[1]};
}

typedef struct { int status; JSContext *ctx; } JobResult;

__attribute__((export_name("JS_ExecutePendingJob")))
JobResult qjs_mv_JS_ExecutePendingJob(JSRuntime *rt)
{
    JobResult r = {0};
    r.status = JS_ExecutePendingJob(rt, &r.ctx);
    return r;
}

typedef struct { uint8_t *ptr; size_t size; uint8_t **sab_tab; size_t sab_tab_len; } WriteObjectResult;

__attribute__((export_name("JS_WriteObject2")))
WriteObjectResult qjs_mv_JS_WriteObject2(JSContext *ctx, JSValueConst obj, int flags)
{
    WriteObjectResult r = {0};
    r.ptr = JS_WriteObject2(ctx, &r.size, obj, flags, &r.sab_tab, &r.sab_tab_len);
    return r;
}

__attribute__((export_name("js_realloc2")))
BufferResult qjs_mv_js_realloc2(JSContext *ctx, void *ptr, size_t size)
{
    BufferResult r = {0};
    r.ptr = js_realloc2(ctx, ptr, size, &r.size);
    return r;
}

typedef struct { void *ptr; JSClassID class_id; } OpaqueResult;

__attribute__((export_name("JS_GetAnyOpaque")))
OpaqueResult qjs_mv_JS_GetAnyOpaque(JSValueConst obj)
{
    OpaqueResult r = {0};
    r.ptr = JS_GetAnyOpaque(obj, &r.class_id);
    return r;
}
