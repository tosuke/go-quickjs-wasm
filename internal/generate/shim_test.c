/* Test-only adapters for the Go implementations in quickjs.go. */
#include "quickjs.h"

_Static_assert(sizeof(void *) == 4 && sizeof(JSValue) == 8, "expected wasm32 QuickJS ABI");

__attribute__((export_name("JS_NewBool")))
JSValue qjs_api_JS_NewBool(JSContext * a0, int a1) { return JS_NewBool(a0, a1); }
__attribute__((export_name("JS_NewInt32")))
JSValue qjs_api_JS_NewInt32(JSContext * a0, int32_t a1) { return JS_NewInt32(a0, a1); }
__attribute__((export_name("JS_NewInt64")))
JSValue qjs_api_JS_NewInt64(JSContext * a0, int64_t a1) { return JS_NewInt64(a0, a1); }
__attribute__((export_name("JS_NewUint32")))
JSValue qjs_api_JS_NewUint32(JSContext * a0, uint32_t a1) { return JS_NewUint32(a0, a1); }
__attribute__((export_name("JS_NewFloat64")))
JSValue qjs_api_JS_NewFloat64(JSContext * a0, double a1) { return JS_NewFloat64(a0, a1); }
__attribute__((export_name("JS_NewCatchOffset")))
JSValue qjs_api_JS_NewCatchOffset(JSContext * a0, int32_t a1) { return JS_NewCatchOffset(a0, a1); }
__attribute__((export_name("JS_VALUE_GET_TAG")))
int qjs_api_JS_VALUE_GET_TAG(JSValue v) { return JS_VALUE_GET_TAG(v); }
__attribute__((export_name("JS_VALUE_GET_PTR")))
void * qjs_api_JS_VALUE_GET_PTR(JSValue v) { return JS_VALUE_GET_PTR(v); }
__attribute__((export_name("JS_VALUE_GET_FLOAT64")))
double qjs_api_JS_VALUE_GET_FLOAT64(JSValue a0) { return JS_VALUE_GET_FLOAT64(a0); }
__attribute__((export_name("JS_VALUE_GET_NORM_TAG")))
int qjs_api_JS_VALUE_GET_NORM_TAG(JSValue a0) { return JS_VALUE_GET_NORM_TAG(a0); }
__attribute__((export_name("JS_TAG_IS_FLOAT64")))
int qjs_api_JS_TAG_IS_FLOAT64(int tag) { return JS_TAG_IS_FLOAT64(tag); }
__attribute__((export_name("JS_VALUE_HAS_REF_COUNT")))
int qjs_api_JS_VALUE_HAS_REF_COUNT(JSValue v) { return JS_VALUE_HAS_REF_COUNT(v); }
__attribute__((export_name("JS_MKVAL")))
JSValue qjs_api_JS_MKVAL(int tag, int val) { return JS_MKVAL(tag, val); }
__attribute__((export_name("JS_MKPTR")))
JSValue qjs_api_JS_MKPTR(int tag, void *ptr) { return JS_MKPTR(tag, ptr); }
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
__attribute__((export_name("JS_VALUE_IS_NAN")))
int qjs_api_JS_VALUE_IS_NAN(JSValue a0) { return JS_VALUE_IS_NAN(a0); }
__attribute__((export_name("JS_VALUE_IS_BOTH_INT")))
int qjs_api_JS_VALUE_IS_BOTH_INT(JSValue v1, JSValue v2) { return JS_VALUE_IS_BOTH_INT(v1, v2); }
__attribute__((export_name("JS_VALUE_IS_BOTH_FLOAT")))
int qjs_api_JS_VALUE_IS_BOTH_FLOAT(JSValue v1, JSValue v2) { return JS_VALUE_IS_BOTH_FLOAT(v1, v2); }
__attribute__((export_name("JS_VALUE_GET_INT")))
int qjs_api_JS_VALUE_GET_INT(JSValue v) { return JS_VALUE_GET_INT(v); }
__attribute__((export_name("JS_VALUE_GET_BOOL")))
int qjs_api_JS_VALUE_GET_BOOL(JSValue v) { return JS_VALUE_GET_BOOL(v); }
__attribute__((export_name("JS_VALUE_GET_SHORT_BIG_INT")))
int32_t qjs_api_JS_VALUE_GET_SHORT_BIG_INT(JSValue v) { return JS_VALUE_GET_SHORT_BIG_INT(v); }

/* Access the actual C bitfields, without assuming their storage offsets. */
__attribute__((export_name("JSPrintValueOptions_show_hidden_read")))
int qjs_test_JSPrintValueOptions_show_hidden_read(const JSPrintValueOptions *p) { return p->show_hidden; }
__attribute__((export_name("JSPrintValueOptions_show_hidden_write")))
void qjs_test_JSPrintValueOptions_show_hidden_write(JSPrintValueOptions *p, int value) { p->show_hidden = value; }
__attribute__((export_name("JSPrintValueOptions_raw_dump_read")))
int qjs_test_JSPrintValueOptions_raw_dump_read(const JSPrintValueOptions *p) { return p->raw_dump; }
__attribute__((export_name("JSPrintValueOptions_raw_dump_write")))
void qjs_test_JSPrintValueOptions_raw_dump_write(JSPrintValueOptions *p, int value) { p->raw_dump = value; }

/* Call through the header's callback types so wasm2go checks the Go function
   values against the actual wasm32 indirect-call signatures. */
#define QJS_TEST_CALLBACK(name, type, ...) \
    __attribute__((export_name("test_call_" #name))) \
    void qjs_test_call_##name(type callback) { (void)callback(__VA_ARGS__); }
QJS_TEST_CALLBACK(JSCFunction, JSCFunction *, 0, 0, 0, 0)
QJS_TEST_CALLBACK(JSCFunctionMagic, JSCFunctionMagic *, 0, 0, 0, 0, 0)
QJS_TEST_CALLBACK(JSCFunctionData, JSCFunctionData *, 0, 0, 0, 0, 0, 0)
QJS_TEST_CALLBACK(JS_MarkFunc, JS_MarkFunc *, 0, 0)
QJS_TEST_CALLBACK(JSClassFinalizer, JSClassFinalizer *, 0, 0)
QJS_TEST_CALLBACK(JSClassGCMark, JSClassGCMark *, 0, 0, 0)
QJS_TEST_CALLBACK(JSClassCall, JSClassCall *, 0, 0, 0, 0, 0, 0)
QJS_TEST_CALLBACK(JSFreeArrayBufferDataFunc, JSFreeArrayBufferDataFunc *, 0, 0, 0)
QJS_TEST_CALLBACK(JSHostPromiseRejectionTracker, JSHostPromiseRejectionTracker *, 0, 0, 0, 0, 0)
QJS_TEST_CALLBACK(JSInterruptHandler, JSInterruptHandler *, 0, 0)
QJS_TEST_CALLBACK(JSModuleNormalizeFunc, JSModuleNormalizeFunc *, 0, 0, 0, 0)
QJS_TEST_CALLBACK(JSModuleLoaderFunc, JSModuleLoaderFunc *, 0, 0, 0)
QJS_TEST_CALLBACK(JSModuleLoaderFunc2, JSModuleLoaderFunc2 *, 0, 0, 0, 0)
QJS_TEST_CALLBACK(JSModuleCheckSupportedImportAttributes, JSModuleCheckSupportedImportAttributes *, 0, 0, 0)
QJS_TEST_CALLBACK(JSJobFunc, JSJobFunc *, 0, 0, 0)
QJS_TEST_CALLBACK(JSModuleInitFunc, JSModuleInitFunc *, 0, 0)
QJS_TEST_CALLBACK(JSPrintValueWrite, JSPrintValueWrite *, 0, 0, 0)
QJS_TEST_CALLBACK(JSMallocFunctionsMalloc, __typeof__(((JSMallocFunctions *)0)->js_malloc), 0, 0)
QJS_TEST_CALLBACK(JSMallocFunctionsFree, __typeof__(((JSMallocFunctions *)0)->js_free), 0, 0)
QJS_TEST_CALLBACK(JSMallocFunctionsRealloc, __typeof__(((JSMallocFunctions *)0)->js_realloc), 0, 0, 0)
QJS_TEST_CALLBACK(JSMallocFunctionsMallocUsableSize, __typeof__(((JSMallocFunctions *)0)->js_malloc_usable_size), 0)
QJS_TEST_CALLBACK(JSClassExoticMethodGetOwnProperty, __typeof__(((JSClassExoticMethods *)0)->get_own_property), 0, 0, 0, 0)
QJS_TEST_CALLBACK(JSClassExoticMethodGetOwnPropertyNames, __typeof__(((JSClassExoticMethods *)0)->get_own_property_names), 0, 0, 0, 0)
QJS_TEST_CALLBACK(JSClassExoticMethodDeleteProperty, __typeof__(((JSClassExoticMethods *)0)->delete_property), 0, 0, 0)
QJS_TEST_CALLBACK(JSClassExoticMethodDefineOwnProperty, __typeof__(((JSClassExoticMethods *)0)->define_own_property), 0, 0, 0, 0, 0, 0, 0)
QJS_TEST_CALLBACK(JSClassExoticMethodHasProperty, __typeof__(((JSClassExoticMethods *)0)->has_property), 0, 0, 0)
QJS_TEST_CALLBACK(JSClassExoticMethodGetProperty, __typeof__(((JSClassExoticMethods *)0)->get_property), 0, 0, 0, 0)
QJS_TEST_CALLBACK(JSClassExoticMethodSetProperty, __typeof__(((JSClassExoticMethods *)0)->set_property), 0, 0, 0, 0, 0, 0)
QJS_TEST_CALLBACK(JSClassExoticMethodGetPrototype, __typeof__(((JSClassExoticMethods *)0)->get_prototype), 0, 0)
QJS_TEST_CALLBACK(JSClassExoticMethodSetPrototype, __typeof__(((JSClassExoticMethods *)0)->set_prototype), 0, 0, 0)
QJS_TEST_CALLBACK(JSClassExoticMethodIsExtensible, __typeof__(((JSClassExoticMethods *)0)->is_extensible), 0, 0)
QJS_TEST_CALLBACK(JSClassExoticMethodPreventExtensions, __typeof__(((JSClassExoticMethods *)0)->prevent_extensions), 0, 0)
QJS_TEST_CALLBACK(JSCFunctionGenericMagic, __typeof__(((JSCFunctionType *)0)->generic_magic), 0, 0, 0, 0, 0)
QJS_TEST_CALLBACK(JSCFunctionConstructorMagic, __typeof__(((JSCFunctionType *)0)->constructor_magic), 0, 0, 0, 0, 0)
QJS_TEST_CALLBACK(JSCFunctionFF, __typeof__(((JSCFunctionType *)0)->f_f), 0)
QJS_TEST_CALLBACK(JSCFunctionFFF, __typeof__(((JSCFunctionType *)0)->f_f_f), 0, 0)
QJS_TEST_CALLBACK(JSCFunctionGetter, __typeof__(((JSCFunctionType *)0)->getter), 0, 0)
QJS_TEST_CALLBACK(JSCFunctionSetter, __typeof__(((JSCFunctionType *)0)->setter), 0, 0, 0)
QJS_TEST_CALLBACK(JSCFunctionGetterMagic, __typeof__(((JSCFunctionType *)0)->getter_magic), 0, 0, 0)
QJS_TEST_CALLBACK(JSCFunctionSetterMagic, __typeof__(((JSCFunctionType *)0)->setter_magic), 0, 0, 0, 0)
QJS_TEST_CALLBACK(JSCFunctionIteratorNext, __typeof__(((JSCFunctionType *)0)->iterator_next), 0, 0, 0, 0, 0, 0)

#undef QJS_TEST_CALLBACK
