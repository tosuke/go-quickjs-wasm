package quickjs_wasm

import "math"

// Represents the QuickJS C API types as int32 for wasm32 ABI compatibility.
// Convert to uint32 for unsigned arithmetic and comparisons.
// Ptr[byte] represents C char *; Ptr[[0]byte] represents C void *.
type (
	Ptr[T any]   = int32
	Size_t       = int32
	JSRuntimePtr = int32
	JSContextPtr = int32
	JSClassPtr   = int32
	JSClassID    = int32
	JSAtom       = int32
)

// all tags wth a reference count are negative
const (
	js_TAG_FIRST             = -9
	JS_TAG_BIG_INT           = -9
	JS_TAG_SYMBOL            = -8
	JS_TAG_STRING            = -7
	JS_TAG_STRING_ROPE       = -6
	JS_TAG_MODULE            = -3
	JS_TAG_FUNCTION_BYTECODE = -2
	JS_TAG_OBJECT            = -1
)

// all tags without a reference count are non-negative
const (
	JS_TAG_INT           = 0
	JS_TAG_BOOL          = 1
	JS_TAG_NULL          = 2
	JS_TAG_UNDEFINED     = 3
	JS_TAG_UNINITIALIZED = 4
	JS_TAG_CATCH_OFFSET  = 5
	JS_TAG_EXCEPTION     = 6
	JS_TAG_SHORT_BIG_INT = 7
	// JS_TAG_FLOAT64 is a normalized tag for fp values. Any larget tag is also a fp value.
	JS_TAG_FLOAT64 = 8
)

type JSValue = int64

const JS_FLOAT64_NAN_BITS = 0x7ff8_0000_0000_0000

func JS_VALUE_GET_TAG(v JSValue) int32 {
	return int32(uint64(v) >> 32)
}

func JS_VALUE_GET_INT(v JSValue) int32 {
	return int32(v)
}

func JS_VALUE_GET_BOOL(v JSValue) bool {
	return int32(v) != 0
}

func JS_VALUE_GET_SHORT_BIG_INT(v JSValue) int32 {
	return int32(v)
}

func JS_VALUE_GET_PTR(v JSValue) int32 {
	return int32(v)
}

func js_MKVAL(tag int32, val int32) JSValue {
	return JSValue(uint32(val)) | JSValue(uint32(tag))<<32
}

func JS_MKVAL(tag int32, val int32) JSValue {
	return js_MKVAL(tag, val)
}

func JS_MKPTR(tag int32, ptr int32) JSValue {
	return js_MKVAL(tag, ptr)
}

const js_FLOAT64_TAG_ADDEND = 0x7ff8_0000 - js_TAG_FIRST + 1

func JS_VALUE_GET_FLOAT64(v JSValue) float64 {
	return math.Float64frombits(uint64(v) + js_FLOAT64_TAG_ADDEND<<32)
}

const JS_NAN JSValue = JS_FLOAT64_NAN_BITS - js_FLOAT64_TAG_ADDEND<<32

func JS_TAG_IS_FLOAT64(tag int32) bool {
	return uint32(tag-js_TAG_FIRST) >= uint32(JS_TAG_FLOAT64-js_TAG_FIRST)
}

func __JS_NewFloat64(d float64) JSValue {
	if math.IsNaN(d) {
		return JS_NAN
	}
	return JSValue(math.Float64bits(d) - js_FLOAT64_TAG_ADDEND<<32)
}

func JS_VALUE_GET_NORM_TAG(v JSValue) int32 {
	tag := JS_VALUE_GET_TAG(v)
	if JS_TAG_IS_FLOAT64(tag) {
		return JS_TAG_FLOAT64
	}
	return tag
}

func JS_VALUE_IS_NAN(v JSValue) bool {
	return JS_VALUE_GET_TAG(v) == int32(JS_NAN>>32)
}

func JS_VALUE_IS_BOTH_INT(v1, v2 JSValue) bool {
	return JS_VALUE_GET_TAG(v1)|JS_VALUE_GET_TAG(v2) == JS_TAG_INT
}

func JS_VALUE_IS_BOTH_FLOAT(v1, v2 JSValue) bool {
	return JS_TAG_IS_FLOAT64(JS_VALUE_GET_TAG(v1)) && JS_TAG_IS_FLOAT64(JS_VALUE_GET_TAG(v2))
}

func JS_VALUE_HAS_REF_COUNT(v JSValue) bool {
	return uint64(v) >= uint64(1<<32+js_TAG_FIRST)<<32
}

// special values
const (
	JS_NULL          JSValue = JS_TAG_NULL << 32
	JS_UNDEFINED     JSValue = JS_TAG_UNDEFINED << 32
	JS_FALSE         JSValue = JS_TAG_BOOL << 32
	JS_TRUE          JSValue = JS_TAG_BOOL<<32 | 1
	JS_EXCEPTION     JSValue = JS_TAG_EXCEPTION << 32
	JS_UNINITIALIZED JSValue = JS_TAG_UNINITIALIZED << 32
)

// flags for object properties
const (
	JS_PROP_CONFIGURABLE = 1 << 0
	JS_PROP_WRITABLE     = 1 << 1
	JS_PROP_ENUMERABLE   = 1 << 2
	JS_PROP_C_W_E        = JS_PROP_CONFIGURABLE | JS_PROP_WRITABLE | JS_PROP_ENUMERABLE
	js_PROP_LENGTH       = 1 << 3
	// JS_PROP_TMASK is mask for the property type (normal, getter/setter, varref, autoinit)
	JS_PROP_TMASK    = 3 << 4
	JS_PROP_NORMAL   = 0 << 4
	JS_PROP_GETSET   = 1 << 4
	js_PROP_VARREF   = 2 << 4
	js_PROP_AUTOINIT = 3 << 4
)

// flags for JS_DefineProperty
const (
	JS_PROP_HAS_SHIFT        = 8
	JS_PROP_HAS_CONFIGURABLE = 1 << 8
	JS_PROP_HAS_WRITABLE     = 1 << 9
	JS_PROP_HAS_ENUMERABLE   = 1 << 10
	JS_PROP_HAS_GET          = 1 << 11
	JS_PROP_HAS_SET          = 1 << 12
	JS_PROP_HAS_VALUE        = 1 << 13
)

// JS_PROP_THROW indicates that the property throw an exception if false would be returned.
//
// This is used for JS_DefineProperty and JS_SetProperty.
const JS_PROP_THROW = 1 << 14

// JS_PROP_THROW_STRICT indicates that the property throw an exception if false would be returned in strict mode.
//
// This is used for JS_SetProperty.
const JS_PROP_THROW_STRICT = 1 << 15

// const js_PROP_NO_EXOTIC = 1 << 16 // internal use

const JS_DEFAULT_STACK_SIZE = 1024 * 1024

// JS_Eval() flags
const (
	// JS_EVAL_TYPE_GLOBAL indicates global code (default)
	JS_EVAL_TYPE_GLOBAL = 0 << 0
	// JS_EVAL_TYPE_MODULE indicates module code
	JS_EVAL_TYPE_MODULE   = 1 << 0
	js_EVAL_TYPE_DIRECT   = 2 << 0
	js_EVAL_TYPE_INDIRECT = 3 << 0
	JS_EVAL_TYPE_MASK     = 3 << 0

	// JS_EVAL_FLAG_STRICT forces 'strict' mode
	JS_EVAL_FLAG_STRICT = 1 << 3
	// JS_EVAL_FLAG_COMPILE_ONLY indicates that JS_Eval compiles but does not run the code.
	//
	// The result is an object with a JS_TAG_FUNCTION_BYTECODE or JS_TAG_MODULE tag.
	// It can be executed with JS_EvalFunction().
	JS_EVAL_FLAG_COMPILE_ONLY = 1 << 5
	// JS_EVAL_FLAG_BACKTRACE_BARRIER don't include the stack frames before this eval in the Error() backtraces.
	JS_EVAL_FLAG_BACKTRACE_BARRIER = 1 << 6
	// JS_EVAL_FLAG_ASYNC allows top-level awaint in normal script.
	//
	// JS_Eval() returns a promise. Only allowed with JS_EVAL_TYPE_GLOBAL.
	JS_EVAL_FLAG_ASYNC = 1 << 7
)

type (
	JSCFunction      = func(ctx JSContextPtr, this_val JSValue, argc int32, argv Ptr[JSValue]) JSValue
	JSCFunctionMagic = func(ctx JSContextPtr, this_val JSValue, argc int32, argv Ptr[JSValue], magic int32) JSValue
	JSCFunctionData  = func(ctx JSContextPtr, this_val JSValue, argc int32, argv Ptr[JSValue], magic int32, func_data Ptr[JSValue]) JSValue
)

type jsMallocState struct {
	malloc_count uint32
	malloc_size  uint32
	malloc_limit uint32
	opaque       Ptr[[0]byte]
}

type jsMallocFunctions struct {
	js_malloc             Ptr[JSMallocFunctionsMalloc]
	js_free               Ptr[JSMallocFunctionsFree]
	js_realloc            Ptr[JSMallocFunctionsRealloc]
	js_malloc_usable_size Ptr[JSMallocFunctionsMallocUsableSize]
}

type (
	JSMallocFunctionsMalloc           = func(s Ptr[JSMallocState], size Size_t) Ptr[[0]byte]
	JSMallocFunctionsFree             = func(s Ptr[JSMallocState], ptr Ptr[[0]byte])
	JSMallocFunctionsRealloc          = func(s Ptr[JSMallocState], ptr Ptr[[0]byte], size Size_t) Ptr[[0]byte]
	JSMallocFunctionsMallocUsableSize = func(ptr Ptr[[0]byte]) Size_t
)

type JSGCObjectHeaderPtr = int32

func (m *Module) JS_NewRuntime() JSRuntimePtr {
	return JSRuntimePtr(m.XJS_NewRuntime())
}
func (m *Module) JS_SetMemoryLimit(rt JSRuntimePtr, limit uint32) {
	m.XJS_SetMemoryLimit(int32(rt), int32(limit))
}
func (m *Module) JS_SetGCThreshold(rt JSRuntimePtr, gc_threshold uint32) {
	m.XJS_SetGCThreshold(int32(rt), int32(gc_threshold))
}

// JS_SetMaxStackSize sets the maximum stack size for the runtime.
//
// use 0 to disable maximum stack size checking.
func (m *Module) JS_SetMaxStackSize(rt JSRuntimePtr, stack_size uint32) {
	m.XJS_SetMaxStackSize(int32(rt), int32(stack_size))
}
func (m *Module) JS_NewRuntime2(mf Ptr[JSMallocFunctions], opaque Ptr[[0]byte]) JSRuntimePtr {
	return JSRuntimePtr(m.XJS_NewRuntime2(int32(mf), int32(opaque)))
}
func (m *Module) JS_FreeRuntime(rt JSRuntimePtr) {
	m.XJS_FreeRuntime(int32(rt))
}

type JS_MarkFunc = func(rt JSRuntimePtr, gp JSGCObjectHeaderPtr)

func (m *Module) JS_MarkValue(rt JSRuntimePtr, val JSValue, mark_func Ptr[JS_MarkFunc]) {
	m.XJS_MarkValue(int32(rt), int64(val), int32(mark_func))
}

func (m *Module) JS_RunGC(rt JSRuntimePtr) {
	m.XJS_RunGC(int32(rt))
}
func (m *Module) JS_IsLiveObject(rt JSRuntimePtr, obj JSValue) bool {
	return m.XJS_IsLiveObject(int32(rt), int64(obj)) != 0
}

func (m *Module) JS_NewContext(rt JSRuntimePtr) JSContextPtr {
	return JSContextPtr(m.XJS_NewContext(int32(rt)))
}
func (m *Module) JS_FreeContext(ctx JSContextPtr) {
	m.XJS_FreeContext(int32(ctx))
}
func (m *Module) JS_DupContext(ctx JSContextPtr) JSContextPtr {
	return JSContextPtr(m.XJS_DupContext(int32(ctx)))
}
func (m *Module) JS_GetRuntime(ctx JSContextPtr) JSRuntimePtr {
	return JSRuntimePtr(m.XJS_GetRuntime(int32(ctx)))
}
func (m *Module) JS_SetClassProto(ctx JSContextPtr, class_id JSClassID, obj JSValue) {
	m.XJS_SetClassProto(int32(ctx), int32(class_id), int64(obj))
}
func (m *Module) JS_GetClassProto(ctx JSContextPtr, class_id JSClassID) JSValue {
	return JSValue(m.XJS_GetClassProto(int32(ctx), int32(class_id)))
}

func (m *Module) JS_malloc_rt(rt JSRuntimePtr, size uint32) Ptr[[0]byte] {
	return Ptr[[0]byte](m.Xjs_malloc_rt(int32(rt), int32(size)))
}
func (m *Module) JS_free_rt(rt JSRuntimePtr, ptr Ptr[[0]byte]) {
	m.Xjs_free_rt(int32(rt), int32(ptr))
}
func (m *Module) JS_realloc_rt(rt JSRuntimePtr, ptr Ptr[[0]byte], size uint32) Ptr[[0]byte] {
	return Ptr[[0]byte](m.Xjs_realloc_rt(int32(rt), int32(ptr), int32(size)))
}
func (m *Module) JS_malloc_usable_size_rt(rt JSRuntimePtr, ptr Ptr[[0]byte]) uint32 {
	return uint32(m.Xjs_malloc_usable_size_rt(int32(rt), int32(ptr)))
}
func (m *Module) JS_mallocz_rt(rt JSRuntimePtr, size uint32) Ptr[[0]byte] {
	return Ptr[[0]byte](m.Xjs_mallocz_rt(int32(rt), int32(size)))
}
func (m *Module) JS_malloc(ctx JSContextPtr, size uint32) Ptr[[0]byte] {
	return Ptr[[0]byte](m.Xjs_malloc(int32(ctx), int32(size)))
}
func (m *Module) JS_free(ctx JSContextPtr, ptr Ptr[[0]byte]) {
	m.Xjs_free(int32(ctx), int32(ptr))
}
func (m *Module) JS_realloc(ctx JSContextPtr, ptr Ptr[[0]byte], size uint32) Ptr[[0]byte] {
	return Ptr[[0]byte](m.Xjs_realloc(int32(ctx), int32(ptr), int32(size)))
}
func (m *Module) JS_malloc_usable_size(ctx JSContextPtr, ptr Ptr[[0]byte]) uint32 {
	return uint32(m.Xjs_malloc_usable_size(int32(ctx), int32(ptr)))
}
func (m *Module) JS_realloc2(ctx JSContextPtr, ptr Ptr[[0]byte], size uint32) (newptr Ptr[[0]byte], slack uint32) {
	p, s := m.Xjs_realloc2(int32(ctx), int32(ptr), int32(size))
	return Ptr[[0]byte](p), uint32(s)
}
func (m *Module) JS_mallocz(ctx JSContextPtr, size uint32) Ptr[[0]byte] {
	return Ptr[[0]byte](m.Xjs_mallocz(int32(ctx), int32(size)))
}

type JSMemoryUsage struct {
	Malloc_size, Malloc_limit, Memory_used_size, Malloc_count, Memory_used_count,
	Atom_count, Atom_size, Str_count, Str_size, Obj_count, Obj_size, Prop_count, Prop_size,
	Shape_count, Shape_size, Js_func_count, Js_func_size, Js_func_code_size,
	Js_func_pc2line_count, Js_func_pc2line_size, C_func_count, Array_count,
	Fast_array_count, Fast_array_elements, Binary_object_count, Binary_object_size int64
}

func (m *Module) JS_ComputeMemoryUsage(rt JSRuntimePtr) JSMemoryUsage {
	var s JSMemoryUsage
	s.Malloc_size,
		s.Malloc_limit, s.Memory_used_size, s.Malloc_count, s.Memory_used_count,
		s.Atom_count, s.Atom_size, s.Str_count, s.Str_size,
		s.Obj_count, s.Obj_size, s.Prop_count, s.Prop_size, s.Shape_count,
		s.Shape_size, s.Js_func_count, s.Js_func_size, s.Js_func_code_size, s.Js_func_pc2line_count,
		s.Js_func_pc2line_size, s.C_func_count, s.Array_count, s.Fast_array_count,
		s.Fast_array_elements, s.Binary_object_count, s.Binary_object_size =
		m.XJS_ComputeMemoryUsage(int32(rt))
	return s
}

const JS_ATOM_NULL = 0

func (m *Module) JS_NewAtomLen(ctx JSContextPtr, str Ptr[byte], len uint32) JSAtom {
	return JSAtom(m.XJS_NewAtomLen(int32(ctx), int32(str), int32(len)))
}
func (m *Module) JS_NewAtom(ctx JSContextPtr, str Ptr[byte]) JSAtom {
	return JSAtom(m.XJS_NewAtom(int32(ctx), int32(str)))
}
func (m *Module) JS_NewAtomUInt32(ctx JSContextPtr, n uint32) JSAtom {
	return JSAtom(m.XJS_NewAtomUInt32(int32(ctx), int32(n)))
}
func (m *Module) JS_DupAtom(ctx JSContextPtr, v JSAtom) JSAtom {
	return JSAtom(m.XJS_DupAtom(int32(ctx), int32(v)))
}
func (m *Module) JS_FreeAtom(ctx JSContextPtr, v JSAtom) {
	m.XJS_FreeAtom(int32(ctx), int32(v))
}
func (m *Module) JS_FreeAtomRT(rt JSRuntimePtr, v JSAtom) {
	m.XJS_FreeAtomRT(int32(rt), int32(v))
}
func (m *Module) JS_AtomToValue(ctx JSContextPtr, atom JSAtom) JSValue {
	return JSValue(m.XJS_AtomToValue(int32(ctx), int32(atom)))
}
func (m *Module) JS_AtomToString(ctx JSContextPtr, atom JSAtom) JSValue {
	return JSValue(m.XJS_AtomToString(int32(ctx), int32(atom)))
}
func (m *Module) JS_AtomToCStringLen(ctx JSContextPtr, atom JSAtom) (str Ptr[byte], len uint32) {
	p, l := m.XJS_AtomToCStringLen(int32(ctx), int32(atom))
	return Ptr[byte](p), uint32(l)
}
func (m *Module) JS_ValueToAtom(ctx JSContextPtr, val JSValue) JSAtom {
	return JSAtom(m.XJS_ValueToAtom(int32(ctx), int64(val)))
}

// Object class

type jsPropertyEnum struct {
	is_enumerable int32
	atom          JSAtom
}

type jsPropertyDescriptor struct {
	flags                 int32
	value, getter, setter JSValue
}

// JSClassExoticMethods defines the exotic methods for a class.
//
// QuickJS emulates the [[HasProperty]], [[Get]], [[Set]], [[GetPrototypeOf]],
// [[SetPrototypeOf]], [[IsExtensible]], and [[PreventExtensions]] with
// [[GetOwnProperty]], [[DefineOwnProperty]], [[Delete]], and [[OwnPropertyKeys]].
// So these exotic methods are usulally not needed.
//
// To get a consistent object behavior when get_prototype != NULL,
// get_property, set_propert and set_prototype must be != NULL
// and the object must be created with a JS_NULL prototype.
type jsClassExoticMethods struct {
	get_own_property       Ptr[JSClassExoticMethodGetOwnProperty]
	get_own_property_names Ptr[JSClassExoticMethodGetOwnPropertyNames]
	delete_property        Ptr[JSClassExoticMethodDeleteProperty]
	define_own_property    Ptr[JSClassExoticMethodDefineOwnProperty]
	has_property           Ptr[JSClassExoticMethodHasProperty]
	get_property           Ptr[JSClassExoticMethodGetProperty]
	set_property           Ptr[JSClassExoticMethodSetProperty]
	get_prototype          Ptr[JSClassExoticMethodGetPrototype]
	set_prototype          Ptr[JSClassExoticMethodSetPrototype]
	is_extensible          Ptr[JSClassExoticMethodIsExtensible]
	prevent_extensions     Ptr[JSClassExoticMethodPreventExtensions]
}

type (
	// JSClassExoticMethodGetOwnProperty is [[[GetOwnProperty]]](https://tc39.es/ecma262/multipage/ecmascript-data-types-and-values.html#internal-method-getownproperty).
	//
	// Returns -1 if exception (can only happen in case of Proxy object),
	// FALSE if the property does not exists, TRUE if it exist.
	// If 1(TRUE) is returned, the property descriptor 'desc' is filled if not NULL.
	JSClassExoticMethodGetOwnProperty = func(
		ctx JSContextPtr, desc Ptr[JSPropertyDescriptor],
		obj JSValue, prop JSAtom) (status int32)
	// JSClassExoticMethodGetOwnPropertyNames is [[[OwnPropertyKeys]]](https://tc39.es/ecma262/multipage/ecmascript-data-types-and-values.html#internal-method-ownpropertykeys).
	//
	// '*ptab' should hold the '*plen' property keys.
	// Return 0 if OK, -1 if exception. The 'is_enumerable' field is ignored.
	JSClassExoticMethodGetOwnPropertyNames = func(
		ctx JSContextPtr, ptab Ptr[Ptr[JSPropertyEnum]],
		plen Ptr[int32],
		obj JSValue) (status int32)
	// JSClassExoticMethodDeleteProperty is [[[Delete]]](https://tc39.es/ecma262/multipage/ecmascript-data-types-and-values.html#internal-method-delete).
	//
	// Returns < 0 if exception, or TRUE/FALSE.
	JSClassExoticMethodDeleteProperty = func(ctx JSContextPtr, obj JSValue, prop JSAtom) (status int32)
	// JSClassExoticMethodDefineOwnProperty is [[[DefineOwnProperty]]](https://tc39.es/ecma262/multipage/ecmascript-data-types-and-values.html#internal-method-defineownproperty).
	//
	// Returns < 0 if exception, or TRUE/FALSE.
	JSClassExoticMethodDefineOwnProperty = func(
		ctx JSContextPtr, this_obj JSValue,
		prop JSAtom, val, getter, setter JSValue, flags int32) (status int32)
	// JSClassExoticMethodHasProperty is [[[HasProperty]]](https://tc39.es/ecma262/multipage/ecmascript-data-types-and-values.html#internal-method-hasproperty).
	//
	// Returns < 0 if exception, or TRUE/FALSE.
	JSClassExoticMethodHasProperty = func(ctx JSContextPtr, obj JSValue, prop JSAtom) (status int32)
	// JSClassExoticMethodGetProperty is [[[Get]]](https://tc39.es/ecma262/multipage/ecmascript-data-types-and-values.html#internal-method-get).
	JSClassExoticMethodGetProperty = func(
		ctx JSContextPtr, obj JSValue, atom JSAtom, receiver JSValue) JSValue
	// JSClassExoticMethodSetProperty is [[[Set]]](https://tc39.es/ecma262/multipage/ecmascript-data-types-and-values.html#internal-method-set).
	//
	// Returns < 0 if exception, or TRUE/FALSE.
	JSClassExoticMethodSetProperty = func(
		ctx JSContextPtr, obj JSValue, atom JSAtom,
		value, receiver JSValue, flags int32) (status int32)
	// JSClassExoticMethodGetPrototype is [[[GetPrototypeOf]]](https://tc39.es/ecma262/multipage/ecmascript-data-types-and-values.html#internal-method-getprototypeof).
	JSClassExoticMethodGetPrototype = func(ctx JSContextPtr, obj JSValue) JSValue
	// JSClassExoticMethodSetPrototype is [[[SetPrototypeOf]]](https://tc39.es/ecma262/multipage/ecmascript-data-types-and-values.html#internal-method-setprototypeof).
	//
	// Returns < 0 if exception, or TRUE/FALSE.
	JSClassExoticMethodSetPrototype = func(ctx JSContextPtr, obj JSValue, proto_val JSValue) (status int32)
	// JSClassExoticMethodIsExtensible is [[[IsExtensible]]](https://tc39.es/ecma262/multipage/ecmascript-data-types-and-values.html#internal-method-isextensible).
	//
	// Returns < 0 if exception, or TRUE/FALSE.
	JSClassExoticMethodIsExtensible = func(ctx JSContextPtr, obj JSValue) (status int32)
	// JSClassExoticMethodPreventExtensions is [[[PreventExtensions]]](https://tc39.es/ecma262/multipage/ecmascript-data-types-and-values.html#internal-method-preventextensions).
	//
	// Returns < 0 if exception, or TRUE/FALSE.
	JSClassExoticMethodPreventExtensions = func(ctx JSContextPtr, obj JSValue) (status int32)
)

type JSClassFinalizer = func(rt JSRuntimePtr, val JSValue)
type JSClassGCMark = func(rt JSRuntimePtr, val JSValue, mark_func Ptr[JS_MarkFunc])

const JS_CALL_FLAG_CONSTRUCTOR = 1 << 0

type JSClassCall = func(
	ctx JSContextPtr, func_obj, this_val JSValue,
	argc int32, argv Ptr[JSValue], flags int32) JSValue

// JSClassDef defines a class.
//
// If call != NULL, the object is a function.
// If flags & JS_CALL_FLAG_CONSTRUCTOR != 0, the function is called as a constructor.
// In this case, 'this_val' is new.target. A constructor call only happens
// if the object constructor bit is set (see JS_SetConstructorBit).
type jsClassDef struct {
	class_name Ptr[byte]
	finalizer  Ptr[JSClassFinalizer]
	gc_mark    Ptr[JSClassGCMark]
	call       Ptr[JSClassCall]
	exotic     Ptr[JSClassExoticMethods]
}

const JS_INVALID_CLASS_ID = 0

func (m *Module) JS_NewClassID(pclass_id Ptr[JSClassID]) JSClassID {
	return JSClassID(m.XJS_NewClassID(int32(pclass_id)))
}

// JS_GetClassID returns the class ID of `v` if `v` is an object, otherwise returns JS_INVALID_CLASS_ID.
func (m *Module) JS_GetClassID(v JSValue) JSClassID {
	return JSClassID(m.XJS_GetClassID(int64(v)))
}
func (m *Module) JS_NewClass(rt JSRuntimePtr, class_id JSClassID, class_def Ptr[JSClassDef]) (status int32) {
	return m.XJS_NewClass(int32(rt), int32(class_id), int32(class_def))
}
func (m *Module) JS_IsRegisteredClass(rt JSRuntimePtr, class_id JSClassID) bool {
	return m.XJS_IsRegisteredClass(int32(rt), int32(class_id)) != 0
}

func JS_NewBool(val bool) (v JSValue) {
	v = JS_FALSE
	if val {
		v = JS_TRUE
	}
	return
}

func JS_NewInt32(val int32) JSValue {
	return /* JS_TAG_INT<<32 | */ JSValue(uint32(val))
}

func JS_NewCatchOffset(val int32) JSValue {
	return JS_TAG_CATCH_OFFSET<<32 | JSValue(uint32(val))
}

func JS_NewInt64(val int64) JSValue {
	if val == int64(int32(val)) {
		return JS_NewInt32(int32(val))
	}
	return __JS_NewFloat64(float64(val))
}

func JS_NewUint32(val uint32) JSValue {
	if val <= math.MaxInt32 {
		return JS_NewInt32(int32(val))
	}
	return __JS_NewFloat64(float64(val))
}

func (m *Module) JS_NewBigInt64(ctx JSContextPtr, v int64) JSValue {
	return JSValue(m.XJS_NewBigInt64(int32(ctx), v))
}
func (m *Module) JS_NewBigUint64(ctx JSContextPtr, v uint64) JSValue {
	return JSValue(m.XJS_NewBigUint64(int32(ctx), int64(v)))
}

func JS_NewFloat64(d float64) JSValue {
	if d >= math.MinInt32 && d <= math.MaxInt32 {
		i := int32(d)
		if math.Float64bits(d) == math.Float64bits(float64(i)) {
			return JS_NewInt32(i)
		}
	}
	return __JS_NewFloat64(d)
}

func JS_IsNumber(v JSValue) bool {
	tag := JS_VALUE_GET_TAG(v)
	return tag == JS_TAG_INT || JS_TAG_IS_FLOAT64(tag)
}

func JS_IsBigInt(v JSValue) bool {
	tag := JS_VALUE_GET_TAG(v)
	return tag == JS_TAG_BIG_INT || tag == JS_TAG_SHORT_BIG_INT
}

func JS_IsBool(v JSValue) bool {
	return JS_VALUE_GET_TAG(v) == JS_TAG_BOOL
}

func JS_IsNull(v JSValue) bool {
	return JS_VALUE_GET_TAG(v) == JS_TAG_NULL
}

func JS_IsUndefined(v JSValue) bool {
	return JS_VALUE_GET_TAG(v) == JS_TAG_UNDEFINED
}

func JS_IsException(v JSValue) bool {
	return JS_VALUE_GET_TAG(v) == JS_TAG_EXCEPTION
}

func JS_IsUninitialized(v JSValue) bool {
	return JS_VALUE_GET_TAG(v) == JS_TAG_UNINITIALIZED
}

func JS_IsString(v JSValue) bool {
	tag := JS_VALUE_GET_TAG(v)
	return tag == JS_TAG_STRING || tag == JS_TAG_STRING_ROPE
}

func JS_IsSymbol(v JSValue) bool {
	return JS_VALUE_GET_TAG(v) == JS_TAG_SYMBOL
}

func JS_IsObject(v JSValue) bool {
	return JS_VALUE_GET_TAG(v) == JS_TAG_OBJECT
}

func (m *Module) JS_Throw(ctx JSContextPtr, obj JSValue) JSValue {
	return JSValue(m.XJS_Throw(int32(ctx), int64(obj)))
}
func (m *Module) JS_SetUncatchableException(ctx JSContextPtr, flag bool) {
	m.XJS_SetUncatchableException(int32(ctx), btoi(flag))
}
func (m *Module) JS_GetException(ctx JSContextPtr) JSValue {
	return JSValue(m.XJS_GetException(int32(ctx)))
}
func (m *Module) JS_HasException(ctx JSContextPtr) bool {
	return m.XJS_HasException(int32(ctx)) != 0
}
func (m *Module) JS_IsError(ctx JSContextPtr, val JSValue) bool {
	return m.XJS_IsError(int32(ctx), int64(val)) != 0
}
func (m *Module) JS_NewError(ctx JSContextPtr) JSValue {
	return JSValue(m.XJS_NewError(int32(ctx)))
}
func (m *Module) JS_ThrowSyntaxError(ctx JSContextPtr, fmt Ptr[byte], pvargs Ptr[[0]byte]) JSValue {
	return JSValue(m.XJS_ThrowSyntaxError(int32(ctx), int32(fmt), int32(pvargs)))
}
func (m *Module) JS_ThrowTypeError(ctx JSContextPtr, fmt Ptr[byte], pvargs Ptr[[0]byte]) JSValue {
	return JSValue(m.XJS_ThrowTypeError(int32(ctx), int32(fmt), int32(pvargs)))
}
func (m *Module) JS_ThrowReferenceError(ctx JSContextPtr, fmt Ptr[byte], pvargs Ptr[[0]byte]) JSValue {
	return JSValue(m.XJS_ThrowReferenceError(int32(ctx), int32(fmt), int32(pvargs)))
}
func (m *Module) JS_ThrowRangeError(ctx JSContextPtr, fmt Ptr[byte], pvargs Ptr[[0]byte]) JSValue {
	return JSValue(m.XJS_ThrowRangeError(int32(ctx), int32(fmt), int32(pvargs)))
}
func (m *Module) JS_ThrowInternalError(ctx JSContextPtr, fmt Ptr[byte], pvargs Ptr[[0]byte]) JSValue {
	return JSValue(m.XJS_ThrowInternalError(int32(ctx), int32(fmt), int32(pvargs)))
}
func (m *Module) JS_ThrowOutOfMemory(ctx JSContextPtr) JSValue {
	return JSValue(m.XJS_ThrowOutOfMemory(int32(ctx)))
}

func (m *Module) JS_FreeValue(ctx JSContextPtr, v JSValue) {
	m.XJS_FreeValue(int32(ctx), int64(v))
}
func (m *Module) JS_FreeValueRT(rt JSRuntimePtr, v JSValue) {
	m.XJS_FreeValueRT(int32(rt), int64(v))
}

func (m *Module) JS_DupValue(ctx JSContextPtr, v JSValue) JSValue {
	return JSValue(m.XJS_DupValue(int32(ctx), int64(v)))
}
func (m *Module) JS_DupValueRT(rt JSRuntimePtr, v JSValue) JSValue {
	return JSValue(m.XJS_DupValueRT(int32(rt), int64(v)))
}

func (m *Module) JS_StrictEq(ctx JSContextPtr, op1, op2 JSValue) bool {
	return m.XJS_StrictEq(int32(ctx), int64(op1), int64(op2)) != 0
}
func (m *Module) JS_SameValue(ctx JSContextPtr, op1, op2 JSValue) bool {
	return m.XJS_SameValue(int32(ctx), int64(op1), int64(op2)) != 0
}
func (m *Module) JS_SameValueZero(ctx JSContextPtr, op1, op2 JSValue) bool {
	return m.XJS_SameValueZero(int32(ctx), int64(op1), int64(op2)) != 0
}

// JS_ToBool converts a JSValue to a boolean value.
//
// 0 is returned for false, 1 for true, -1 for JS_EXCEPTION.
func (m *Module) JS_ToBool(ctx JSContextPtr, val JSValue) int32 {
	return m.XJS_ToBool(int32(ctx), int64(val))
}
func (m *Module) JS_ToInt32(ctx JSContextPtr, val JSValue) (res int32, status int32) {
	status, res = m.XJS_ToInt32(int32(ctx), int64(val))
	return
}
func (m *Module) JS_ToUint32(ctx JSContextPtr, val JSValue) (res uint32, status int32) {
	i, status := m.JS_ToInt32(ctx, val)
	return uint32(i), status
}
func (m *Module) JS_ToInt64(ctx JSContextPtr, val JSValue) (res int64, status int32) {
	status, res = m.XJS_ToInt64(int32(ctx), int64(val))
	return
}
func (m *Module) JS_ToIndex(ctx JSContextPtr, val JSValue) (res uint64, status int32) {
	status, i := m.XJS_ToIndex(int32(ctx), int64(val))
	return uint64(i), status
}
func (m *Module) JS_ToFloat64(ctx JSContextPtr, val JSValue) (res float64, status int32) {
	status, res = m.XJS_ToFloat64(int32(ctx), int64(val))
	return
}

// JS_ToBigUint64 converts a JSValue to a uint64 value.
//
// Returns an exception if 'val' is a Number.
func (m *Module) JS_ToBigInt64(ctx JSContextPtr, val JSValue) (res int64, status int32) {
	status, res = m.XJS_ToBigInt64(int32(ctx), int64(val))
	return
}

// JS_ToInt64Ext is same as JS_ToInt64 but allows BigInt.
func (m *Module) JS_ToInt64Ext(ctx JSContextPtr, val JSValue) (res int64, status int32) {
	status, res = m.XJS_ToInt64Ext(int32(ctx), int64(val))
	return
}

func (m *Module) JS_NewStringLen(ctx JSContextPtr, str Ptr[byte], len uint32) JSValue {
	return JSValue(m.XJS_NewStringLen(int32(ctx), int32(str), int32(len)))
}
func (m *Module) JS_NewAtomString(ctx JSContextPtr, str Ptr[byte]) JSValue {
	return JSValue(m.XJS_NewAtomString(int32(ctx), int32(str)))
}
func (m *Module) JS_ToString(ctx JSContextPtr, val JSValue) JSValue {
	return JSValue(m.XJS_ToString(int32(ctx), int64(val)))
}
func (m *Module) JS_ToPropertyKey(ctx JSContextPtr, val JSValue) JSValue {
	return JSValue(m.XJS_ToPropertyKey(int32(ctx), int64(val)))
}
func (m *Module) JS_ToCStringLen2(ctx JSContextPtr, val JSValue, cesu8 bool) (str Ptr[byte], len uint32) {
	p, l := m.XJS_ToCStringLen2(int32(ctx), int64(val), btoi(cesu8))
	return Ptr[byte](p), uint32(l)
}
func (m *Module) JS_ToCStringLen(ctx JSContextPtr, val JSValue) (str Ptr[byte], len uint32) {
	return m.JS_ToCStringLen2(ctx, val, false)
}
func (m *Module) JS_FreeCString(ctx JSContextPtr, ptr Ptr[byte]) {
	m.XJS_FreeCString(int32(ctx), int32(ptr))
}

func (m *Module) JS_NewObjectProtoClass(ctx JSContextPtr, proto JSValue, class_id JSClassID) JSValue {
	return JSValue(m.XJS_NewObjectProtoClass(int32(ctx), int64(proto), int32(class_id)))
}
func (m *Module) JS_NewObjectClass(ctx JSContextPtr, class_id JSClassID) JSValue {
	return JSValue(m.XJS_NewObjectClass(int32(ctx), int32(class_id)))
}
func (m *Module) JS_NewObjectProto(ctx JSContextPtr, proto JSValue) JSValue {
	return JSValue(m.XJS_NewObjectProto(int32(ctx), int64(proto)))
}
func (m *Module) JS_NewObject(ctx JSContextPtr) JSValue {
	return JSValue(m.XJS_NewObject(int32(ctx)))
}

func (m *Module) JS_IsFunction(ctx JSContextPtr, val JSValue) bool {
	return m.XJS_IsFunction(int32(ctx), int64(val)) != 0
}
func (m *Module) JS_IsConstructor(ctx JSContextPtr, val JSValue) bool {
	return m.XJS_IsConstructor(int32(ctx), int64(val)) != 0
}
func (m *Module) JS_SetConstructorBit(ctx JSContextPtr, func_obj JSValue, val bool) bool {
	return m.XJS_SetConstructorBit(int32(ctx), int64(func_obj), btoi(val)) != 0
}

func (m *Module) JS_NewArray(ctx JSContextPtr) JSValue {
	return JSValue(m.XJS_NewArray(int32(ctx)))
}
func (m *Module) JS_IsArray(ctx JSContextPtr, val JSValue) int32 {
	return m.XJS_IsArray(int32(ctx), int64(val))
}

func (m *Module) JS_NewDate(ctx JSContextPtr, epoch_ms float64) JSValue {
	return JSValue(m.XJS_NewDate(int32(ctx), epoch_ms))
}

func (m *Module) JS_GetPropertyInternal(
	ctx JSContextPtr, obj JSValue, prop JSAtom, receiver JSValue, throw_ref_error bool,
) JSValue {
	return JSValue(m.XJS_GetPropertyInternal(
		int32(ctx), int64(obj), int32(prop), int64(receiver), btoi(throw_ref_error)))
}
func (m *Module) JS_GetProperty(ctx JSContextPtr, this_obj JSValue, prop JSAtom) JSValue {
	return m.JS_GetPropertyInternal(ctx, this_obj, prop, this_obj, false)
}
func (m *Module) JS_GetPropertyStr(ctx JSContextPtr, this_obj JSValue, prop Ptr[byte]) JSValue {
	return JSValue(m.XJS_GetPropertyStr(int32(ctx), int64(this_obj), int32(prop)))
}
func (m *Module) JS_GetPropertyUint32(ctx JSContextPtr, this_obj JSValue, idx uint32) JSValue {
	return JSValue(m.XJS_GetPropertyUint32(int32(ctx), int64(this_obj), int32(idx)))
}

func (m *Module) JS_SetPropertyInternal(
	ctx JSContextPtr, obj JSValue, prop JSAtom, val JSValue, this_obj JSValue, flags int32,
) (status int32) {
	return m.XJS_SetPropertyInternal(int32(ctx), int64(obj), int32(prop), int64(val), int64(this_obj), flags)
}
func (m *Module) JS_SetProperty(ctx JSContextPtr, this_obj JSValue, prop JSAtom, val JSValue) (status int32) {
	return m.JS_SetPropertyInternal(ctx, this_obj, prop, val, this_obj, JS_PROP_THROW)
}
func (m *Module) JS_SetPropertyUint32(ctx JSContextPtr, this_obj JSValue, idx uint32, val JSValue) (status int32) {
	return m.XJS_SetPropertyUint32(int32(ctx), int64(this_obj), int32(idx), int64(val))
}
func (m *Module) JS_SetPropertyInt64(ctx JSContextPtr, this_obj JSValue, idx int64, val JSValue) (status int32) {
	return m.XJS_SetPropertyInt64(int32(ctx), int64(this_obj), idx, int64(val))
}
func (m *Module) JS_SetPropertyStr(ctx JSContextPtr, this_obj JSValue, prop Ptr[byte], val JSValue) (status int32) {
	return m.XJS_SetPropertyStr(int32(ctx), int64(this_obj), int32(prop), int64(val))
}
func (m *Module) JS_IsExtensible(ctx JSContextPtr, obj JSValue) int32 {
	return m.XJS_IsExtensible(int32(ctx), int64(obj))
}
func (m *Module) JS_PreventExtensions(ctx JSContextPtr, obj JSValue) (status int32) {
	return m.XJS_PreventExtensions(int32(ctx), int64(obj))
}
func (m *Module) JS_DeleteProperty(ctx JSContextPtr, obj JSValue, prop JSAtom, flags int32) (status int32) {
	return m.XJS_DeleteProperty(int32(ctx), int64(obj), int32(prop), flags)
}
func (m *Module) JS_SetPrototype(ctx JSContextPtr, obj JSValue, proto_val JSValue) (status int32) {
	return m.XJS_SetPrototype(int32(ctx), int64(obj), int64(proto_val))
}
func (m *Module) JS_GetPrototype(ctx JSContextPtr, obj JSValue) JSValue {
	return JSValue(m.XJS_GetPrototype(int32(ctx), int64(obj)))
}

// JS_GetOwnPropertyNames() flags
const (
	JS_GPN_STRING_MASK  = 1 << 0
	JS_GPN_SYMBOL_MASK  = 1 << 1
	JS_GPN_PRIVATE_MASK = 1 << 2
	JS_GPN_ENUM_ONLY    = 1 << 4
	JS_GPN_SET_ENUM     = 1 << 5
)

func (m *Module) JS_GetOwnPropertyNames(
	ctx JSContextPtr, obj JSValue, flags int32,
) (tab Ptr[JSPropertyEnum], len uint32, status int32) {
	status, p, l := m.XJS_GetOwnPropertyNames(int32(ctx), int64(obj), flags)
	return Ptr[JSPropertyEnum](p), uint32(l), status
}
func (m *Module) JS_FreePropertyEnum(ctx JSContextPtr, tab Ptr[JSPropertyEnum], len uint32) {
	m.XJS_FreePropertyEnum(int32(ctx), int32(tab), int32(len))
}

func (m *Module) JS_GetOwnProperty(ctx JSContextPtr, obj JSValue, prop JSAtom) (value, getter, setter JSValue, flags, status int32) {
	var vvalue, vgetter, vsetter int64
	status, flags, vvalue, vgetter, vsetter = m.XJS_GetOwnProperty(int32(ctx), int64(obj), int32(prop))
	return JSValue(vvalue), JSValue(vgetter), JSValue(vsetter), flags, status
}

func (m *Module) JS_Call(ctx JSContextPtr, func_obj, this_obj JSValue, argc int32, argv Ptr[JSValue]) JSValue {
	return JSValue(m.XJS_Call(int32(ctx), int64(func_obj), int64(this_obj), argc, int32(argv)))
}
func (m *Module) JS_Invoke(ctx JSContextPtr, this_obj JSValue, prop JSAtom, argc int32, argv Ptr[JSValue]) JSValue {
	return JSValue(m.XJS_Invoke(int32(ctx), int64(this_obj), int32(prop), argc, int32(argv)))
}
func (m *Module) JS_CallConstructor(ctx JSContextPtr, func_obj JSValue, argc int32, argv Ptr[JSValue]) JSValue {
	return JSValue(m.XJS_CallConstructor(int32(ctx), int64(func_obj), argc, int32(argv)))
}
func (m *Module) JS_CallConstructor2(ctx JSContextPtr, func_obj, new_target JSValue, argc int32, argv Ptr[JSValue]) JSValue {
	return JSValue(m.XJS_CallConstructor2(int32(ctx), int64(func_obj), int64(new_target), argc, int32(argv)))
}
func (m *Module) JS_DetectModule(input Ptr[byte], input_len Size_t) bool {
	return m.XJS_DetectModule(int32(input), input_len) != 0
}

// JS_Eval evaluates JavaScript input code.
//
// 'input' must be zero terminated i.e. input[input_len] = '\0'.
func (m *Module) JS_Eval(ctx JSContextPtr, input Ptr[byte], input_len Size_t, filename Ptr[byte], eval_flags int32) JSValue {
	return JSValue(m.XJS_Eval(int32(ctx), int32(input), input_len, int32(filename), eval_flags))
}

// JS_EvalThis is same as JS_Eval but with an explicit 'this_obj'.
func (m *Module) JS_EvalThis(
	ctx JSContextPtr, this_obj JSValue, input Ptr[byte], input_len Size_t, filename Ptr[byte], eval_flags int32) JSValue {
	return JSValue(m.XJS_EvalThis(int32(ctx), int64(this_obj), int32(input), input_len, int32(filename), eval_flags))
}
func (m *Module) JS_GetGlobalObject(ctx JSContextPtr) JSValue {
	return JSValue(m.XJS_GetGlobalObject(int32(ctx)))
}
func (m *Module) JS_IsInstanceOf(ctx JSContextPtr, val, obj JSValue) int32 {
	return m.XJS_IsInstanceOf(int32(ctx), int64(val), int64(obj))
}
func (m *Module) JS_DefineProperty(
	ctx JSContextPtr, this_obj JSValue, prop JSAtom, val, getter, setter JSValue, flags int32) (status int32) {
	return m.XJS_DefineProperty(int32(ctx), int64(this_obj), int32(prop), int64(val), int64(getter), int64(setter), flags)
}
func (m *Module) JS_DefinePropertyValue(
	ctx JSContextPtr, this_obj JSValue, prop JSAtom, val JSValue, flags int32) (status int32) {
	return m.XJS_DefinePropertyValue(int32(ctx), int64(this_obj), int32(prop), int64(val), flags)
}
func (m *Module) JS_DefinePropertyValueUint32(
	ctx JSContextPtr, this_obj JSValue, idx uint32, val JSValue, flags int32) (status int32) {
	return m.XJS_DefinePropertyValueUint32(int32(ctx), int64(this_obj), int32(idx), int64(val), flags)
}
func (m *Module) JS_DefinePropertyValueStr(
	ctx JSContextPtr, this_obj JSValue, prop Ptr[byte], val JSValue, flags int32) (status int32) {
	return m.XJS_DefinePropertyValueStr(int32(ctx), int64(this_obj), int32(prop), int64(val), flags)
}
func (m *Module) JS_DefinePropertyGetSet(
	ctx JSContextPtr, this_obj JSValue, prop JSAtom, getter, setter JSValue, flags int32) (status int32) {
	return m.XJS_DefinePropertyGetSet(int32(ctx), int64(this_obj), int32(prop), int64(getter), int64(setter), flags)
}

// JS_ParseJSON parses a JSON string 'buf'.
//
// 'buf' must be zero terminated i.e. buf[buf_len] = '\0'.
func (m *Module) JS_ParseJSON(ctx JSContextPtr, buf Ptr[byte], buf_len Size_t, filename Ptr[byte]) JSValue {
	return JSValue(m.XJS_ParseJSON(int32(ctx), int32(buf), buf_len, int32(filename)))
}

// JS_PARSE_JSON_EXT allows extended JSON.
const JS_PARSE_JSON_EXT = 1 << 0

func (m *Module) JS_ParseJSON2(ctx JSContextPtr, buf Ptr[byte], buf_len Size_t, filename Ptr[byte], flags int32) JSValue {
	return JSValue(m.XJS_ParseJSON2(int32(ctx), int32(buf), buf_len, int32(filename), flags))
}
func (m *Module) JS_JSONStringify(ctx JSContextPtr, val, replacer, space0 JSValue) JSValue {
	return JSValue(m.XJS_JSONStringify(int32(ctx), int64(val), int64(replacer), int64(space0)))
}

type JSFreeArrayBufferDataFunc = func(rt JSRuntimePtr, opaque, ptr Ptr[[0]byte])

func (m *Module) JS_NewArrayBuffer(
	ctx JSContextPtr, buf Ptr[byte], len uint32, free_func Ptr[JSFreeArrayBufferDataFunc], opaque Ptr[[0]byte], is_shared bool,
) JSValue {
	return JSValue(m.XJS_NewArrayBuffer(int32(ctx), int32(buf), int32(len), int32(free_func), int32(opaque), btoi(is_shared)))
}

func (m *Module) JS_NewArrayBufferCopy(ctx JSContextPtr, buf Ptr[byte], len uint32) JSValue {
	return JSValue(m.XJS_NewArrayBufferCopy(int32(ctx), int32(buf), int32(len)))
}
func (m *Module) JS_DetachArrayBuffer(ctx JSContextPtr, obj JSValue) {
	m.XJS_DetachArrayBuffer(int32(ctx), int64(obj))
}
func (m *Module) JS_GetArrayBuffer(ctx JSContextPtr, obj JSValue) (ptr Ptr[byte], len uint32) {
	p, l := m.XJS_GetArrayBuffer(int32(ctx), int64(obj))
	return Ptr[byte](p), uint32(l)
}

type JSTypedArrayEnum = int32

const (
	JS_TYPED_ARRAY_UINT8C JSTypedArrayEnum = iota
	JS_TYPED_ARRAY_INT8
	JS_TYPED_ARRAY_UINT8
	JS_TYPED_ARRAY_INT16
	JS_TYPED_ARRAY_UINT16
	JS_TYPED_ARRAY_INT32
	JS_TYPED_ARRAY_UINT32
	JS_TYPED_ARRAY_BIG_INT64
	JS_TYPED_ARRAY_BIG_UINT64
	JS_TYPED_ARRAY_FLOAT16
	JS_TYPED_ARRAY_FLOAT32
	JS_TYPED_ARRAY_FLOAT64
)

func (m *Module) JS_NewTypedArray(ctx JSContextPtr, argc int32, argv Ptr[JSValue], array_type JSTypedArrayEnum) JSValue {
	return JSValue(m.XJS_NewTypedArray(int32(ctx), argc, int32(argv), int32(array_type)))
}
func (m *Module) JS_GetTypedArrayBuffer(
	ctx JSContextPtr, obj JSValue,
) (buffer JSValue, byte_offset, byte_length, bytes_per_element Size_t) {
	buf, ofs, len, bpe := m.XJS_GetTypedArrayBuffer(int32(ctx), int64(obj))
	return JSValue(buf), ofs, len, bpe
}

type JSPromiseStateEnum = int32

const (
	JS_PROMISE_PENDING JSPromiseStateEnum = iota
	JS_PROMISE_FULFILLED
	JS_PROMISE_REJECTED
)

func (m *Module) JS_NewPromiseCapability(ctx JSContextPtr) (promise, resolve_func, reject_func JSValue) {
	p, res, rej := m.XJS_NewPromiseCapability(int32(ctx))
	return JSValue(p), JSValue(res), JSValue(rej)
}
func (m *Module) JS_PromiseState(ctx JSContextPtr, promise JSValue) JSPromiseStateEnum {
	return JSPromiseStateEnum(m.XJS_PromiseState(int32(ctx), int64(promise)))
}
func (m *Module) JS_PromiseResult(ctx JSContextPtr, promise JSValue) JSValue {
	return JSValue(m.XJS_PromiseResult(int32(ctx), int64(promise)))
}

// JSHostPromiseRejectionTracker is [HostPromiseRejectionTracket](https://tc39.es/ecma262/2025/#sec-host-promise-rejection-tracker).
type JSHostPromiseRejectionTracker = func(
	ctx JSContextPtr, promise, reason JSValue, is_handled int32, opaque Ptr[[0]byte])

func (m *Module) JS_SetHostPromiseRejectionTracker(rt JSRuntimePtr, cb Ptr[JSHostPromiseRejectionTracker], opaque Ptr[[0]byte]) {
	m.XJS_SetHostPromiseRejectionTracker(int32(rt), int32(cb), int32(opaque))
}

// Returns != 0 if the JS code needs to be interrupted.
type JSInterruptHandler = func(rt JSRuntimePtr, opaque Ptr[[0]byte]) int32

func (m *Module) JS_SetInterruptHandler(rt JSRuntimePtr, cb Ptr[JSInterruptHandler], opaque Ptr[[0]byte]) {
	m.XJS_SetInterruptHandler(int32(rt), int32(cb), int32(opaque))
}

// JS_SetCanBlock enables blocking Atomics.wait calls for this runtime.
// Blocking is disabled by default.
func (m *Module) JS_SetCanBlock(rt JSRuntimePtr, can_block bool) {
	m.XJS_SetCanBlock(rt, btoi(can_block))
}

// JS_SetStripInfo() flags
const (
	JS_STRIP_SOURCE = 1 << 0
	JS_STRIP_DEBUG  = 1 << 1
)

func (m *Module) JS_SetStripInfo(rt JSRuntimePtr, flags int32) {
	m.XJS_SetStripInfo(int32(rt), flags)
}
func (m *Module) JS_GetStripInfo(rt JSRuntimePtr) int32 {
	return m.XJS_GetStripInfo(int32(rt))
}

// JS_SetIsHTMLDDA sets the [IsHTMLDDA] internal slot.
func (m *Module) JS_SetIsHTMLDDA(ctx JSContextPtr, obj JSValue) {
	m.XJS_SetIsHTMLDDA(int32(ctx), int64(obj))
}

type JSModuleDefPtr = int32

// JSModuleNormalizeFunc normalizes 'module_name' relative to 'module_base_name' and
// returns the module specifier (allocated with JS_malloc()) or NULL if exception.
type JSModuleNormalizeFunc = func(ctx JSContextPtr, module_base_name, module_name Ptr[byte], opaque Ptr[[0]byte]) (str Ptr[byte])

type JSModuleLoaderFunc = func(ctx JSContextPtr, module_name Ptr[byte], opaque Ptr[[0]byte]) JSModuleDefPtr
type JSModuleLoaderFunc2 = func(ctx JSContextPtr, module_name Ptr[byte], opaque Ptr[[0]byte], attributes JSValue) JSModuleDefPtr

// JSModuleCheckSupportedImportAttributes returns -1 if exception, 0 if OK.
type JSModuleCheckSupportedImportAttributes = func(ctx JSContextPtr, opaque Ptr[[0]byte], attributes JSValue) int32

// JS_SetModuleLoaderFunc sets the module loader functions.
//
// module_normalize == NULL is allowed and invokes the default module filename normalizer.
func (m *Module) JS_SetModuleLoaderFunc(rt JSRuntimePtr,
	module_normalize Ptr[JSModuleNormalizeFunc], module_loader Ptr[JSModuleLoaderFunc], opaque Ptr[[0]byte],
) {
	m.XJS_SetModuleLoaderFunc(int32(rt), int32(module_normalize), int32(module_loader), int32(opaque))
}

// JS_SetModuleLoaderFunc2 is same as JS_SetModuleLoaderFunc but with attributes.
//
// If module_check_attrs == NULL, no attribute checking is done.
func (m *Module) JS_SetModuleLoaderFunc2(rt JSRuntimePtr,
	module_normalize Ptr[JSModuleNormalizeFunc],
	module_loader Ptr[JSModuleLoaderFunc2],
	module_check_attrs Ptr[JSModuleCheckSupportedImportAttributes],
	opaque Ptr[[0]byte],
) {
	m.XJS_SetModuleLoaderFunc2(int32(rt),
		int32(module_normalize), int32(module_loader), int32(module_check_attrs), int32(opaque))
}

// JS_GetImportMeta returns the import.meta object of a module.
func (m *Module) JS_GetImportMeta(ctx JSContextPtr, md JSModuleDefPtr) JSValue {
	return JSValue(m.XJS_GetImportMeta(int32(ctx), int32(md)))
}
func (m *Module) JS_GetModuleName(ctx JSContextPtr, md JSModuleDefPtr) JSAtom {
	return JSAtom(m.XJS_GetModuleName(int32(ctx), int32(md)))
}
func (m *Module) JS_GetModuleNamespace(ctx JSContextPtr, md JSModuleDefPtr) JSValue {
	return JSValue(m.XJS_GetModuleNamespace(int32(ctx), int32(md)))
}

// JS Job support

type JSJobFunc = func(ctx JSContextPtr, argc int32, argv Ptr[JSValue]) JSValue

func (m *Module) JS_EnqueueJob(ctx JSContextPtr, job_func Ptr[JSJobFunc], argc int32, argv Ptr[JSValue]) (status int32) {
	return m.XJS_EnqueueJob(int32(ctx), int32(job_func), argc, int32(argv))
}

func (m *Module) JS_IsJobPending(rt JSRuntimePtr) bool {
	return m.XJS_IsJobPending(int32(rt)) != 0
}
func (m *Module) JS_ExecutePendingJob(rt JSRuntimePtr) (ctx JSContextPtr, status int32) {
	status, c := m.XJS_ExecutePendingJob(int32(rt))
	return JSContextPtr(c), status
}

// JS_WriteObject() flags
const (
	JS_WRITE_OBJ_BYTECODE  = 1 << 0 // allow function/module
	JS_WRITE_OBJ_BSWAP     = 1 << 1 // avoid duplicating 'buf' data
	JS_WRITE_OBJ_SAB       = 1 << 2 // allow SharedArrayBuffer
	JS_WRITE_OBJ_REFERENCE = 1 << 3 // allow object references
)

func (m *Module) JS_WriteObject(ctx JSContextPtr, obj JSValue, flags int32) (buf Ptr[byte], size uint32) {
	b, s, _, _ := m.JS_WriteObject2(ctx, obj, flags)
	return b, s
}
func (m *Module) JS_WriteObject2(
	ctx JSContextPtr, obj JSValue, flags int32,
) (buf Ptr[byte], size uint32, sab_tab Ptr[Ptr[byte]], sab_tab_len uint32) {
	p, s, sab, sab_len := m.XJS_WriteObject2(int32(ctx), int64(obj), flags)
	return Ptr[byte](p), uint32(s), Ptr[Ptr[byte]](sab), uint32(sab_len)
}

// JS_ReadObject() flags
const (
	JS_READ_OBJ_BYTECODE  = 1 << 0 // allow function/module
	JS_READ_OBJ_ROM_DATA  = 1 << 1 // avoid duplicating 'buf' data
	JS_READ_OBJ_SAB       = 1 << 2 // allow SharedArrayBuffer
	JS_READ_OBJ_REFERENCE = 1 << 3 // allow object references
)

func (m *Module) JS_ReadObject(ctx JSContextPtr, buf Ptr[byte], buf_len uint32, flags int32) JSValue {
	return JSValue(m.XJS_ReadObject(int32(ctx), int32(buf), int32(buf_len), flags))
}

// JS_EvalFunction instantiates and evaluates a bytecode function.
// Only used when reading a script or module with JS_ReadObject().
func (m *Module) JS_EvalFunction(ctx JSContextPtr, func_obj JSValue) JSValue {
	return JSValue(m.XJS_EvalFunction(int32(ctx), int64(func_obj)))
}

// JS_ResolveModule loads the dependencies of the module 'obj'.
func (m *Module) JS_ResolveModule(ctx JSContextPtr, obj JSValue) (status int32) {
	return m.XJS_ResolveModule(int32(ctx), int64(obj))
}

type JSCFunctionEnum = int32

const (
	JS_CFUNC_generic JSCFunctionEnum = iota
	JS_CFUNC_generic_magic
	JS_CFUNC_constructor
	JS_CFUNC_constructor_magic
	JS_CFUNC_constructor_or_func
	JS_CFUNC_constructor_or_func_magic
	JS_CFUNC_f_f
	JS_CFUNC_f_f_f
	JS_CFUNC_getter
	JS_CFUNC_setter
	JS_CFUNC_getter_magic
	JS_CFUNC_setter_magic
	JS_CFUNC_iterator_next
)

type jsCFunctionType struct {
	generic             Ptr[JSCFunction]
	generic_magic       Ptr[JSCFunctionGenericMagic]
	constructor         Ptr[JSCFunction]
	constructor_magic   Ptr[JSCFunctionConstructorMagic]
	constructor_or_func Ptr[JSCFunction]
	f_f                 Ptr[JSCFunctionFF]
	f_f_f               Ptr[JSCFunctionFFF]
	getter              Ptr[JSCFunctionGetter]
	setter              Ptr[JSCFunctionSetter]
	getter_magic        Ptr[JSCFunctionGetterMagic]
	setter_magic        Ptr[JSCFunctionSetterMagic]
	iterator_next       Ptr[JSCFunctionIteratorNext]
}

type (
	JSCFunctionGenericMagic     = func(ctx JSContextPtr, this_val JSValue, argc int32, argv Ptr[JSValue], magic int32) JSValue
	JSCFunctionConstructorMagic = func(ctx JSContextPtr, new_target JSValue, argc int32, argv Ptr[JSValue], magic int32) JSValue
	JSCFunctionFF               = func(float64) float64
	JSCFunctionFFF              = func(float64, float64) float64
	JSCFunctionGetter           = func(ctx JSContextPtr, this_val JSValue) JSValue
	JSCFunctionSetter           = func(ctx JSContextPtr, this_val, val JSValue) JSValue
	JSCFunctionGetterMagic      = func(ctx JSContextPtr, this_val JSValue, magic int32) JSValue
	JSCFunctionSetterMagic      = func(ctx JSContextPtr, this_val, val JSValue, magic int32) JSValue
	JSCFunctionIteratorNext     = func(
		ctx JSContextPtr, this_val JSValue, argc int32, argv Ptr[JSValue], pdone Ptr[int32], magic int32) JSValue
)

func (m *Module) JS_NewCFunction2(
	ctx JSContextPtr, fun Ptr[JSCFunction], name Ptr[byte], length int32, cproto JSCFunctionEnum, magic int32) JSValue {
	return JSValue(m.XJS_NewCFunction2(int32(ctx), int32(fun), int32(name), length, int32(cproto), magic))
}

func (m *Module) JS_NewCFunctionData(
	ctx JSContextPtr, fun Ptr[JSCFunctionData], length int32, magic int32, data_len int32, data Ptr[JSValue]) JSValue {
	return JSValue(m.XJS_NewCFunctionData(int32(ctx), int32(fun), length, magic, data_len, int32(data)))
}

func (m *Module) JS_NewCFunction(ctx JSContextPtr, fun Ptr[JSCFunction], name Ptr[byte], length int32) JSValue {
	return m.JS_NewCFunction2(ctx, fun, name, length, JS_CFUNC_generic, 0)
}

func (m *Module) JS_NewCFunctionMagic(
	ctx JSContextPtr, fun Ptr[JSCFunctionGenericMagic], name Ptr[byte], length int32, cproto JSCFunctionEnum, magic int32,
) JSValue {
	return m.JS_NewCFunction2(ctx, Ptr[JSCFunction](fun), name, length, cproto, magic)
}

func (m *Module) JS_SetConstructor(ctx JSContextPtr, func_obj, proto JSValue) (status int32) {
	return m.XJS_SetConstructor(int32(ctx), int64(func_obj), int64(proto))
}

// property definition

type jsCFunctionListEntry struct {
	name       Ptr[byte]
	prop_flags uint8
	def_type   uint8
	magic      int16
	u          struct {
		func_ struct {
			length uint8
			cproto uint8
			cfunc  JSCFunctionType
		} `c:"func"`
		getset struct{ get, set JSCFunctionType }
		alias  struct {
			name Ptr[byte]
			base int32
		}
		prop_list struct {
			tab Ptr[JSCFunctionListEntry]
			len int32
		}
		str Ptr[byte]
		i32 int32
		i64 int64
		f64 float64
	}
}

const (
	JS_DEF_CFUNC          = 0
	JS_DEF_CGETSET        = 1
	JS_DEF_CGETSET_MAGIC  = 2
	JS_DEF_PROP_STRING    = 3
	JS_DEF_PROP_INT32     = 4
	JS_DEF_PROP_INT64     = 5
	JS_DEF_PROP_DOUBLE    = 6
	JS_DEF_PROP_UNDEFINED = 7
	JS_DEF_OBJECT         = 8
	JS_DEF_ALIAS          = 9
	JS_DEF_PROP_ATOM      = 10
	JS_DEF_PROP_BOOL      = 11
)

func (m *Module) JS_SetPropertyFunctionList(
	ctx JSContextPtr, obj JSValue, tab Ptr[JSCFunctionListEntry], len int32,
) (status int32) {
	return m.XJS_SetPropertyFunctionList(int32(ctx), int64(obj), int32(tab), len)
}

// module definition

type JSModuleInitFunc = func(ctx JSContextPtr, md JSModuleDefPtr) int32

func (m *Module) JS_NewCModule(ctx JSContextPtr, name_str Ptr[byte], fun Ptr[JSModuleInitFunc]) JSModuleDefPtr {
	return JSModuleDefPtr(m.XJS_NewCModule(int32(ctx), int32(name_str), int32(fun)))
}

func (m *Module) JS_AddModuleExport(ctx JSContextPtr, md JSModuleDefPtr, name_str Ptr[byte]) (status int32) {
	return m.XJS_AddModuleExport(int32(ctx), int32(md), int32(name_str))
}

func (m *Module) JS_AddModuleExportList(
	ctx JSContextPtr, md JSModuleDefPtr, tab Ptr[JSCFunctionListEntry], len int32,
) (status int32) {
	return m.XJS_AddModuleExportList(int32(ctx), int32(md), int32(tab), len)
}

func (m *Module) JS_SetModuleExport(ctx JSContextPtr, md JSModuleDefPtr, export_name Ptr[byte], val JSValue) (status int32) {
	return m.XJS_SetModuleExport(int32(ctx), int32(md), int32(export_name), int64(val))
}

func (m *Module) JS_SetModuleExportList(
	ctx JSContextPtr, md JSModuleDefPtr, tab Ptr[JSCFunctionListEntry], len int32,
) (status int32) {
	return m.XJS_SetModuleExportList(int32(ctx), int32(md), int32(tab), len)
}

func (m *Module) JS_SetModulePrivateValue(ctx JSContextPtr, md JSModuleDefPtr, val JSValue) (status int32) {
	return m.XJS_SetModulePrivateValue(int32(ctx), int32(md), int64(val))
}
func (m *Module) JS_GetModulePrivateValue(ctx JSContextPtr, md JSModuleDefPtr) JSValue {
	return JSValue(m.XJS_GetModulePrivateValue(int32(ctx), int32(md)))
}

// debug value output

type jsPrintValueOptions struct {
	// show_hidden    bool : 8
	// raw_dump       bool : 8
	max_depth         uint32
	max_string_length uint32
	max_item_count    uint32
}

func JSPrintValueOptions_show_hidden(ptr Ptr[JSPrintValueOptions]) Ptr[int8] { return ptr }
func JSPrintValueOptions_raw_dump(ptr Ptr[JSPrintValueOptions]) Ptr[int8]    { return ptr + 1 }

type JSPrintValueWrite = func(opaque Ptr[[0]byte], buf Ptr[byte], len Size_t)

func btoi(b bool) (i int32) {
	if b {
		i = 1
	}
	return
}
