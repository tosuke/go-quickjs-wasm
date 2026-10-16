module github.com/tosuke/go-quickjs-wasm/internal

go 1.26.0

require (
	github.com/ncruces/wasm2go v0.4.18 // indirect
	golang.org/x/tools v0.52.0 // indirect
)

tool (
	github.com/ncruces/wasm2go
	github.com/ncruces/wasm2go/libc-gen
)
