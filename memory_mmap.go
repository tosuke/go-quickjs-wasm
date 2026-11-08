//go:build linux || darwin

package quickjs_wasm

import (
	"fmt"
	"syscall"
)

type memoryRegion struct{ data []byte }

func reserveMemory(size int64) (*memoryRegion, error) {
	if size > int64(^uint(0)>>1) {
		return nil, fmt.Errorf("Wasm memory reservation requires a 64-bit address space")
	}
	b, err := syscall.Mmap(-1, 0, int(size), syscall.PROT_NONE, syscall.MAP_PRIVATE|syscall.MAP_ANON)
	if err != nil {
		return nil, fmt.Errorf("reserve Wasm memory: %w", err)
	}
	return &memoryRegion{data: b}, nil
}

func (r *memoryRegion) enable(start, end int) error {
	if start == end {
		return nil
	}
	return syscall.Mprotect(r.data[start:end], syscall.PROT_READ|syscall.PROT_WRITE)
}

func (r *memoryRegion) close() error {
	if r.data == nil {
		return nil
	}
	if err := syscall.Munmap(r.data); err != nil {
		return err
	}
	r.data = nil
	return nil
}
