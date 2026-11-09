package quickjs_wasm

import (
	"math"
	"runtime"
)

func (m *Module) LoadI8(ptr Ptr[int8]) int8 {
	defer runtime.KeepAlive(m)
	return int8((*m.memory)[uint32(ptr)])
}

func (m *Module) StoreI8(ptr Ptr[int8], value int8) {
	defer runtime.KeepAlive(m)
	(*m.memory)[uint32(ptr)] = byte(value)
}

func (m *Module) LoadU8(ptr Ptr[uint8]) uint8 {
	defer runtime.KeepAlive(m)
	return (*m.memory)[uint32(ptr)]
}

func (m *Module) StoreU8(ptr Ptr[uint8], value uint8) {
	defer runtime.KeepAlive(m)
	(*m.memory)[uint32(ptr)] = value
}

func (m *Module) LoadI16(ptr Ptr[int16]) int16 {
	defer runtime.KeepAlive(m)
	return int16(load16((*m.memory), uint32(ptr)))
}

func (m *Module) StoreI16(ptr Ptr[int16], value int16) {
	defer runtime.KeepAlive(m)
	store16((*m.memory), uint32(ptr), uint16(value))
}

func (m *Module) LoadU16(ptr Ptr[uint16]) uint16 {
	defer runtime.KeepAlive(m)
	return load16((*m.memory), uint32(ptr))
}

func (m *Module) StoreU16(ptr Ptr[uint16], value uint16) {
	defer runtime.KeepAlive(m)
	store16((*m.memory), uint32(ptr), value)
}

func (m *Module) LoadI32(ptr Ptr[int32]) int32 {
	defer runtime.KeepAlive(m)
	return int32(load32((*m.memory), uint32(ptr)))
}

func (m *Module) StoreI32(ptr Ptr[int32], value int32) {
	defer runtime.KeepAlive(m)
	store32((*m.memory), uint32(ptr), uint32(value))
}

func (m *Module) LoadU32(ptr Ptr[uint32]) uint32 {
	defer runtime.KeepAlive(m)
	return load32((*m.memory), uint32(ptr))
}

func (m *Module) StoreU32(ptr Ptr[uint32], value uint32) {
	defer runtime.KeepAlive(m)
	store32((*m.memory), uint32(ptr), value)
}

func (m *Module) LoadJSAtom(ptr Ptr[JSAtom]) JSAtom {
	defer runtime.KeepAlive(m)
	return JSAtom(load32((*m.memory), uint32(ptr)))
}

func (m *Module) StoreJSAtom(ptr Ptr[JSAtom], value JSAtom) {
	defer runtime.KeepAlive(m)
	store32((*m.memory), uint32(ptr), uint32(value))
}

func LoadPtr[T any](m *Module, ptr Ptr[Ptr[T]]) Ptr[T] {
	defer runtime.KeepAlive(m)
	return Ptr[T](load32((*m.memory), uint32(ptr)))
}

func StorePtr[T any](m *Module, ptr Ptr[Ptr[T]], value Ptr[T]) {
	defer runtime.KeepAlive(m)
	store32((*m.memory), uint32(ptr), uint32(value))
}

func (m *Module) LoadI64(ptr Ptr[int64]) int64 {
	defer runtime.KeepAlive(m)
	return int64(load64((*m.memory), uint32(ptr)))
}

func (m *Module) StoreI64(ptr Ptr[int64], value int64) {
	defer runtime.KeepAlive(m)
	store64((*m.memory), uint32(ptr), uint64(value))
}

func (m *Module) LoadJSValue(ptr Ptr[JSValue]) JSValue {
	defer runtime.KeepAlive(m)
	return JSValue(load64((*m.memory), uint32(ptr)))
}

func (m *Module) StoreJSValue(ptr Ptr[JSValue], value JSValue) {
	defer runtime.KeepAlive(m)
	store64((*m.memory), uint32(ptr), uint64(value))
}

func (m *Module) LoadF64(ptr Ptr[float64]) float64 {
	defer runtime.KeepAlive(m)
	return math.Float64frombits(load64((*m.memory), uint32(ptr)))
}

func (m *Module) StoreF64(ptr Ptr[float64], value float64) {
	defer runtime.KeepAlive(m)
	store64((*m.memory), uint32(ptr), math.Float64bits(value))
}

func (m *Module) LoadBytes(dst []byte, ptr Ptr[byte], n int) {
	defer runtime.KeepAlive(m)
	copy(dst, (*m.memory)[uint32(ptr):uint32(ptr)+uint32(n)])
}

func (m *Module) StoreBytes(ptr Ptr[byte], src []byte) {
	defer runtime.KeepAlive(m)
	copy((*m.memory)[uint32(ptr):uint32(ptr)+uint32(len(src))], src)
}

func (m *Module) StoreString(ptr Ptr[byte], s string) {
	defer runtime.KeepAlive(m)
	copy((*m.memory)[uint32(ptr):uint32(ptr)+uint32(len(s))], s)
	(*m.memory)[uint32(ptr)+uint32(len(s))] = 0
}
