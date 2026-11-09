package quickjs_wasm_test

import (
	"math"
	"sync"
	"testing"
	"unsafe"
)

func TestImportedMemoryGrowth(t *testing.T) {
	shared := sharedMemory(t)
	mem := shared
	view := mem.Bytes()
	base := unsafe.SliceData(view)
	reserved := *mem.Slice()
	if uint64(len(reserved)) != uint64(65536)*65536 {
		t.Fatal("mmap header does not span the maximum reservation")
	}
	pages := int64(len(view) >> 16)
	view[len(view)-1] = 123
	for _, delta := range []int64{-1, math.MinInt64, math.MaxInt64, 65536} {
		if got := mem.Grow(delta, 65536); got != -1 {
			t.Fatalf("Grow(%d) = %d, want -1", delta, got)
		}
		if got := mem.Grow(0, 65536); got != pages {
			t.Fatalf("failed growth changed size to %d pages", got)
		}
	}
	if got := mem.Grow(2, pages+1); got != -1 {
		t.Fatalf("growth exceeded caller limit: %d", got)
	}
	if got := mem.Grow(2, pages+2); got != pages {
		t.Fatalf("Grow(2) = %d, want %d", got, pages)
	}
	grown := mem.Bytes()
	if len(grown) != len(view)+2*65536 || unsafe.SliceData(grown) != base {
		t.Fatal("growth did not preserve base and extend length by two pages")
	}
	if len(*mem.Slice()) != len(reserved) || unsafe.SliceData(*mem.Slice()) != base {
		t.Fatal("Grow changed the mmap slice header")
	}
	for _, b := range grown[len(view):] {
		if b != 0 {
			t.Fatal("new pages are not zero initialized")
		}
	}
	if grown[len(view)-1] != 123 {
		t.Fatal("growth lost old data")
	}
	grown[len(view)-1] = 42
	if view[len(view)-1] != 42 {
		t.Fatal("host view no longer aliases Wasm memory")
	}
	// Exercise the final wasm32 page without touching all intervening pages.
	if got := mem.Grow(65536-pages-2, math.MaxInt64); got != pages+2 {
		t.Fatalf("growth to the wasm32 limit = %d", got)
	}
	last := (*mem.Slice())[uint64(1)<<32-1:]
	if last[0] != 0 {
		t.Fatal("last page was not zero initialized")
	}
	last[0] = 99
	if got := mem.Grow(1, math.MaxInt64); got != -1 {
		t.Fatalf("growth exceeded wasm32 limit: %d", got)
	}
	if err := shared.Close(); err != nil {
		t.Fatal(err)
	}
	if len(*mem.Slice()) != 0 || mem.Grow(0, 65536) != -1 {
		t.Fatal("closed memory is still usable")
	}
}

func TestConcurrentMemoryGrowth(t *testing.T) {
	m := module(t)
	defer m.Close()
	mem := m.Xmemory()
	start := mem.Grow(0, 65536)
	const workers = 16
	claimed := make(chan int64, workers)
	var wg sync.WaitGroup
	for range workers {
		wg.Go(func() { claimed <- mem.Grow(1, 65536) })
	}
	wg.Wait()
	close(claimed)
	seen := make(map[int64]bool)
	for old := range claimed {
		if old < start || old >= start+workers || seen[old] {
			t.Fatalf("invalid or duplicate page claim: %d", old)
		}
		seen[old] = true
	}
	if got := mem.Grow(0, 65536); got != start+workers {
		t.Fatalf("memory size = %d, want %d", got, start+workers)
	}
}
