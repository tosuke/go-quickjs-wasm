/* Raw wasm32 adapters for quickjs.h inline functions and macros. */
#include "quickjs.h"
#include <math.h>

/* abi.go describes this pinned wasm32 ABI. */
_Static_assert(sizeof(void *) == 4 && sizeof(JSValue) == 8, "expected wasm32 QuickJS ABI");
_Static_assert(sizeof(JSPropertyDescriptor) == 32, "update abi.go for this QuickJS version");
_Static_assert(sizeof(JSCFunctionListEntry) == 16, "update abi.go for this QuickJS version");
_Static_assert(sizeof(JSClassDef) == 20, "update abi.go for this QuickJS version");
_Static_assert(sizeof(JSMemoryUsage) == 208, "update abi.go for this QuickJS version");
_Static_assert(sizeof(JSPrintValueOptions) == 16, "update abi.go for this QuickJS version");

__attribute__((export_name("JS_VALUE_GET_FLOAT64")))
double qjs_api_JS_VALUE_GET_FLOAT64(JSValue a0) { return JS_VALUE_GET_FLOAT64(a0); }
__attribute__((export_name("JS_VALUE_GET_NORM_TAG")))
int qjs_api_JS_VALUE_GET_NORM_TAG(JSValue a0) { return JS_VALUE_GET_NORM_TAG(a0); }
__attribute__((export_name("JS_VALUE_IS_NAN")))
int qjs_api_JS_VALUE_IS_NAN(JSValue a0) { return JS_VALUE_IS_NAN(a0); }
__attribute__((export_name("JS_AtomToCString")))
const char * qjs_api_JS_AtomToCString(JSContext * a0, JSAtom a1) { return JS_AtomToCString(a0, a1); }
__attribute__((export_name("JS_NewBool")))
JSValue qjs_api_JS_NewBool(JSContext * a0, int a1) { return JS_NewBool(a0, a1); }
__attribute__((export_name("JS_NewInt32")))
JSValue qjs_api_JS_NewInt32(JSContext * a0, int32_t a1) { return JS_NewInt32(a0, a1); }
__attribute__((export_name("JS_NewCatchOffset")))
JSValue qjs_api_JS_NewCatchOffset(JSContext * a0, int32_t a1) { return JS_NewCatchOffset(a0, a1); }
__attribute__((export_name("JS_NewInt64")))
JSValue qjs_api_JS_NewInt64(JSContext * a0, int64_t a1) { return JS_NewInt64(a0, a1); }
__attribute__((export_name("JS_NewUint32")))
JSValue qjs_api_JS_NewUint32(JSContext * a0, uint32_t a1) { return JS_NewUint32(a0, a1); }
__attribute__((export_name("JS_NewFloat64")))
JSValue qjs_api_JS_NewFloat64(JSContext * a0, double a1) { return JS_NewFloat64(a0, a1); }
__attribute__((export_name("JS_IsNumber")))
int qjs_api_JS_IsNumber(JSValue a0) { return JS_IsNumber(a0); }
__attribute__((export_name("JS_IsBigInt")))
int qjs_api_JS_IsBigInt(JSContext * a0, JSValue a1) { return JS_IsBigInt(a0, a1); }
__attribute__((export_name("JS_IsBool")))
int qjs_api_JS_IsBool(JSValue a0) { return JS_IsBool(a0); }
__attribute__((export_name("JS_IsNull")))
int qjs_api_JS_IsNull(JSValue a0) { return JS_IsNull(a0); }
__attribute__((export_name("JS_IsUndefined")))
int qjs_api_JS_IsUndefined(JSValue a0) { return JS_IsUndefined(a0); }
__attribute__((export_name("JS_IsException")))
int qjs_api_JS_IsException(JSValue a0) { return JS_IsException(a0); }
__attribute__((export_name("JS_IsUninitialized")))
int qjs_api_JS_IsUninitialized(JSValue a0) { return JS_IsUninitialized(a0); }
__attribute__((export_name("JS_IsString")))
int qjs_api_JS_IsString(JSValue a0) { return JS_IsString(a0); }
__attribute__((export_name("JS_IsSymbol")))
int qjs_api_JS_IsSymbol(JSValue a0) { return JS_IsSymbol(a0); }
__attribute__((export_name("JS_IsObject")))
int qjs_api_JS_IsObject(JSValue a0) { return JS_IsObject(a0); }
__attribute__((export_name("JS_FreeValue")))
void qjs_api_JS_FreeValue(JSContext * a0, JSValue a1) { JS_FreeValue(a0, a1); }
__attribute__((export_name("JS_FreeValueRT")))
void qjs_api_JS_FreeValueRT(JSRuntime * a0, JSValue a1) { JS_FreeValueRT(a0, a1); }
__attribute__((export_name("JS_DupValue")))
JSValue qjs_api_JS_DupValue(JSContext * a0, JSValue a1) { return JS_DupValue(a0, a1); }
__attribute__((export_name("JS_DupValueRT")))
JSValue qjs_api_JS_DupValueRT(JSRuntime * a0, JSValue a1) { return JS_DupValueRT(a0, a1); }
__attribute__((export_name("JS_ToUint32")))
int qjs_api_JS_ToUint32(JSContext * a0, uint32_t * a1, JSValue a2) { return JS_ToUint32(a0, a1, a2); }
__attribute__((export_name("JS_NewString")))
JSValue qjs_api_JS_NewString(JSContext * a0, const char * a1) { return JS_NewString(a0, a1); }
__attribute__((export_name("JS_ToCStringLen")))
const char * qjs_api_JS_ToCStringLen(JSContext * a0, size_t * a1, JSValue a2) { return JS_ToCStringLen(a0, a1, a2); }
__attribute__((export_name("JS_ToCString")))
const char * qjs_api_JS_ToCString(JSContext * a0, JSValue a1) { return JS_ToCString(a0, a1); }
__attribute__((export_name("JS_GetProperty")))
JSValue qjs_api_JS_GetProperty(JSContext * a0, JSValue a1, JSAtom a2) { return JS_GetProperty(a0, a1, a2); }
__attribute__((export_name("JS_SetProperty")))
int qjs_api_JS_SetProperty(JSContext * a0, JSValue a1, JSAtom a2, JSValue a3) { return JS_SetProperty(a0, a1, a2, a3); }
__attribute__((export_name("JS_NewCFunction")))
JSValue qjs_api_JS_NewCFunction(JSContext * a0, JSCFunction * a1, const char * a2, int a3) { return JS_NewCFunction(a0, a1, a2, a3); }
__attribute__((export_name("JS_NewCFunctionMagic")))
JSValue qjs_api_JS_NewCFunctionMagic(JSContext * a0, JSCFunctionMagic * a1, const char * a2, int a3, JSCFunctionEnum a4, int a5) { return JS_NewCFunctionMagic(a0, a1, a2, a3, a4, a5); }
__attribute__((export_name("JSPrintValueOptions_get_show_hidden")))
int qjs_get_JSPrintValueOptions_show_hidden(JSPrintValueOptions *p) { return p->show_hidden; }
__attribute__((export_name("JSPrintValueOptions_set_show_hidden")))
void qjs_set_JSPrintValueOptions_show_hidden(JSPrintValueOptions *p, int v) { p->show_hidden = v; }
__attribute__((export_name("JSPrintValueOptions_get_raw_dump")))
int qjs_get_JSPrintValueOptions_raw_dump(JSPrintValueOptions *p) { return p->raw_dump; }
__attribute__((export_name("JSPrintValueOptions_set_raw_dump")))
void qjs_set_JSPrintValueOptions_raw_dump(JSPrintValueOptions *p, int v) { p->raw_dump = v; }
__attribute__((export_name("JS_VALUE_GET_TAG")))
int qjs_api_JS_VALUE_GET_TAG(JSValue v) { return JS_VALUE_GET_TAG(v); }
__attribute__((export_name("JS_VALUE_GET_INT")))
int qjs_api_JS_VALUE_GET_INT(JSValue v) { return JS_VALUE_GET_INT(v); }
__attribute__((export_name("JS_VALUE_GET_BOOL")))
int qjs_api_JS_VALUE_GET_BOOL(JSValue v) { return JS_VALUE_GET_BOOL(v); }
__attribute__((export_name("JS_VALUE_GET_SHORT_BIG_INT")))
int32_t qjs_api_JS_VALUE_GET_SHORT_BIG_INT(JSValue v) { return JS_VALUE_GET_SHORT_BIG_INT(v); }
__attribute__((export_name("JS_VALUE_GET_PTR")))
void * qjs_api_JS_VALUE_GET_PTR(JSValue v) { return JS_VALUE_GET_PTR(v); }
__attribute__((export_name("JS_MKVAL")))
JSValue qjs_api_JS_MKVAL(int tag, int val) { return JS_MKVAL(tag, val); }
__attribute__((export_name("JS_MKPTR")))
JSValue qjs_api_JS_MKPTR(int tag, void *ptr) { return JS_MKPTR(tag, ptr); }
__attribute__((export_name("JS_TAG_IS_FLOAT64")))
int qjs_api_JS_TAG_IS_FLOAT64(int tag) { return JS_TAG_IS_FLOAT64(tag); }
__attribute__((export_name("JS_VALUE_IS_BOTH_INT")))
int qjs_api_JS_VALUE_IS_BOTH_INT(JSValue v1, JSValue v2) { return JS_VALUE_IS_BOTH_INT(v1, v2); }
__attribute__((export_name("JS_VALUE_IS_BOTH_FLOAT")))
int qjs_api_JS_VALUE_IS_BOTH_FLOAT(JSValue v1, JSValue v2) { return JS_VALUE_IS_BOTH_FLOAT(v1, v2); }
__attribute__((export_name("JS_VALUE_HAS_REF_COUNT")))
int qjs_api_JS_VALUE_HAS_REF_COUNT(JSValue v) { return JS_VALUE_HAS_REF_COUNT(v); }
__attribute__((export_name("JS_FLOAT64_NAN")))
double qjs_api_JS_FLOAT64_NAN(void) { return JS_FLOAT64_NAN; }
__attribute__((export_name("JS_CFUNC_DEF")))
void qjs_api_JS_CFUNC_DEF(JSCFunctionListEntry *out, const char * a0, int a1, JSCFunction * a2) { *out = (JSCFunctionListEntry)JS_CFUNC_DEF(a0, a1, a2); }
__attribute__((export_name("JS_CFUNC_MAGIC_DEF")))
void qjs_api_JS_CFUNC_MAGIC_DEF(JSCFunctionListEntry *out, const char * a0, int a1, JSCFunctionMagic * a2, int a3) { *out = (JSCFunctionListEntry)JS_CFUNC_MAGIC_DEF(a0, a1, a2, a3); }
__attribute__((export_name("JS_CFUNC_SPECIAL_DEF")))
void qjs_api_JS_CFUNC_SPECIAL_DEF(JSCFunctionListEntry *out, const char * a0, int a1, JSCFunctionEnum a2, JSCFunction * a3) { *out = (JSCFunctionListEntry){ a0, JS_PROP_WRITABLE | JS_PROP_CONFIGURABLE, JS_DEF_CFUNC, 0, .u = { .func = { a1, a2, { .generic = a3 } } } }; }
__attribute__((export_name("JS_ITERATOR_NEXT_DEF")))
void qjs_api_JS_ITERATOR_NEXT_DEF(JSCFunctionListEntry *out, const char * a0, int a1, __typeof__(((JSCFunctionType *)0)->iterator_next) a2, int a3) { *out = (JSCFunctionListEntry)JS_ITERATOR_NEXT_DEF(a0, a1, a2, a3); }
__attribute__((export_name("JS_CGETSET_DEF")))
void qjs_api_JS_CGETSET_DEF(JSCFunctionListEntry *out, const char * a0, __typeof__(((JSCFunctionType *)0)->getter) a1, __typeof__(((JSCFunctionType *)0)->setter) a2) { *out = (JSCFunctionListEntry)JS_CGETSET_DEF(a0, a1, a2); }
__attribute__((export_name("JS_CGETSET_MAGIC_DEF")))
void qjs_api_JS_CGETSET_MAGIC_DEF(JSCFunctionListEntry *out, const char * a0, __typeof__(((JSCFunctionType *)0)->getter_magic) a1, __typeof__(((JSCFunctionType *)0)->setter_magic) a2, int a3) { *out = (JSCFunctionListEntry)JS_CGETSET_MAGIC_DEF(a0, a1, a2, a3); }
__attribute__((export_name("JS_PROP_STRING_DEF")))
void qjs_api_JS_PROP_STRING_DEF(JSCFunctionListEntry *out, const char * a0, const char * a1, int a2) { *out = (JSCFunctionListEntry)JS_PROP_STRING_DEF(a0, a1, a2); }
__attribute__((export_name("JS_PROP_INT32_DEF")))
void qjs_api_JS_PROP_INT32_DEF(JSCFunctionListEntry *out, const char * a0, int32_t a1, int a2) { *out = (JSCFunctionListEntry)JS_PROP_INT32_DEF(a0, a1, a2); }
__attribute__((export_name("JS_PROP_INT64_DEF")))
void qjs_api_JS_PROP_INT64_DEF(JSCFunctionListEntry *out, const char * a0, int64_t a1, int a2) { *out = (JSCFunctionListEntry)JS_PROP_INT64_DEF(a0, a1, a2); }
__attribute__((export_name("JS_PROP_DOUBLE_DEF")))
void qjs_api_JS_PROP_DOUBLE_DEF(JSCFunctionListEntry *out, const char * a0, double a1, int a2) { *out = (JSCFunctionListEntry)JS_PROP_DOUBLE_DEF(a0, a1, a2); }
__attribute__((export_name("JS_PROP_UNDEFINED_DEF")))
void qjs_api_JS_PROP_UNDEFINED_DEF(JSCFunctionListEntry *out, const char * a0, int a1) { *out = (JSCFunctionListEntry)JS_PROP_UNDEFINED_DEF(a0, a1); }
__attribute__((export_name("JS_PROP_ATOM_DEF")))
void qjs_api_JS_PROP_ATOM_DEF(JSCFunctionListEntry *out, const char * a0, JSAtom a1, int a2) { *out = (JSCFunctionListEntry)JS_PROP_ATOM_DEF(a0, a1, a2); }
__attribute__((export_name("JS_PROP_BOOL_DEF")))
void qjs_api_JS_PROP_BOOL_DEF(JSCFunctionListEntry *out, const char * a0, int a1, int a2) { *out = (JSCFunctionListEntry)JS_PROP_BOOL_DEF(a0, a1, a2); }
__attribute__((export_name("JS_OBJECT_DEF")))
void qjs_api_JS_OBJECT_DEF(JSCFunctionListEntry *out, const char * a0, const JSCFunctionListEntry * a1, int a2, int a3) { *out = (JSCFunctionListEntry)JS_OBJECT_DEF(a0, a1, a2, a3); }
__attribute__((export_name("JS_ALIAS_DEF")))
void qjs_api_JS_ALIAS_DEF(JSCFunctionListEntry *out, const char * a0, const char * a1) { *out = (JSCFunctionListEntry)JS_ALIAS_DEF(a0, a1); }
__attribute__((export_name("JS_ALIAS_BASE_DEF")))
void qjs_api_JS_ALIAS_BASE_DEF(JSCFunctionListEntry *out, const char * a0, const char * a1, int a2) { *out = (JSCFunctionListEntry)JS_ALIAS_BASE_DEF(a0, a1, a2); }
