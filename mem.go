package quickjs_wasm

import "math"

func (m *Module) LoadI8(ptr Ptr[int8]) int8 {
	return int8(m.memory[uint32(ptr)])
}

func (m *Module) StoreI8(ptr Ptr[int8], value int8) {
	m.memory[uint32(ptr)] = byte(value)
}

func (m *Module) LoadU8(ptr Ptr[uint8]) uint8 {
	return m.memory[uint32(ptr)]
}

func (m *Module) StoreU8(ptr Ptr[uint8], value uint8) {
	m.memory[uint32(ptr)] = value
}

func (m *Module) LoadI16(ptr Ptr[int16]) int16 {
	return int16(load16(m.memory, uint32(ptr)))
}

func (m *Module) StoreI16(ptr Ptr[int16], value int16) {
	store16(m.memory, uint32(ptr), uint16(value))
}

func (m *Module) LoadU16(ptr Ptr[uint16]) uint16 {
	return load16(m.memory, uint32(ptr))
}

func (m *Module) StoreU16(ptr Ptr[uint16], value uint16) {
	store16(m.memory, uint32(ptr), value)
}

func (m *Module) LoadI32(ptr Ptr[int32]) int32 {
	return int32(load32(m.memory, uint32(ptr)))
}

func (m *Module) StoreI32(ptr Ptr[int32], value int32) {
	store32(m.memory, uint32(ptr), uint32(value))
}

func (m *Module) LoadU32(ptr Ptr[uint32]) uint32 {
	return load32(m.memory, uint32(ptr))
}

func (m *Module) StoreU32(ptr Ptr[uint32], value uint32) {
	store32(m.memory, uint32(ptr), value)
}

func (m *Module) LoadJSAtom(ptr Ptr[JSAtom]) JSAtom {
	return JSAtom(load32(m.memory, uint32(ptr)))
}

func (m *Module) StoreJSAtom(ptr Ptr[JSAtom], value JSAtom) {
	store32(m.memory, uint32(ptr), uint32(value))
}

func LoadPtr[T any](m *Module, ptr Ptr[Ptr[T]]) Ptr[T] {
	return Ptr[T](load32(m.memory, uint32(ptr)))
}

func StorePtr[T any](m *Module, ptr Ptr[Ptr[T]], value Ptr[T]) {
	store32(m.memory, uint32(ptr), uint32(value))
}

func (m *Module) LoadI64(ptr Ptr[int64]) int64 {
	return int64(load64(m.memory, uint32(ptr)))
}

func (m *Module) StoreI64(ptr Ptr[int64], value int64) {
	store64(m.memory, uint32(ptr), uint64(value))
}

func (m *Module) LoadJSValue(ptr Ptr[JSValue]) JSValue {
	return JSValue(load64(m.memory, uint32(ptr)))
}

func (m *Module) StoreJSValue(ptr Ptr[JSValue], value JSValue) {
	store64(m.memory, uint32(ptr), uint64(value))
}

func (m *Module) LoadF64(ptr Ptr[float64]) float64 {
	return math.Float64frombits(load64(m.memory, uint32(ptr)))
}

func (m *Module) StoreF64(ptr Ptr[float64], value float64) {
	store64(m.memory, uint32(ptr), math.Float64bits(value))
}

func (m *Module) LoadBytes(dst []byte, ptr Ptr[byte], n int) {
	copy(dst, m.memory[uint32(ptr):uint32(ptr)+uint32(n)])
}

func (m *Module) StoreBytes(ptr Ptr[byte], src []byte) {
	copy(m.memory[uint32(ptr):uint32(ptr)+uint32(len(src))], src)
}

func (m *Module) StoreString(ptr Ptr[byte], s string) {
	copy(m.memory[uint32(ptr):uint32(ptr)+uint32(len(s))], s)
	m.memory[uint32(ptr)+uint32(len(s))] = 0
}
