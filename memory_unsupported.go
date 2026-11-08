//go:build !linux && !darwin

package quickjs_wasm

import "fmt"

type memoryRegion struct{ data []byte }

func reserveMemory(size int64) (*memoryRegion, error) {
	return nil, fmt.Errorf("mmap-backed Wasm memory is unavailable on this platform")
}

func (r *memoryRegion) enable(start, end int) error { return nil }
func (r *memoryRegion) close() error                { return nil }
