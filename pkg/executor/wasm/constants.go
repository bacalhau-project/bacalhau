package wasm

// WASM architecture and memory constants
// WebAssembly1:  linear memory objects have sizes measured in pages. Each page is 65536 (2^16) bytes.
// In WebAssembly version 1, a linear memory can have at most 65536 pages, for a total of 2^32 bytes (4 gibibytes).
const (
	// WasmArch represents the WebAssembly architecture (32-bit)
	WasmArch = 32

	// WasmPageSize represents the size of a WebAssembly memory page (64KB)
	WasmPageSize = 65536

	// WasmMaxPagesLimit represents the maximum number of memory pages allowed (4GB)
	WasmMaxPagesLimit = 1 << (WasmArch / 2)

	// BytesInGB represents the number of bytes in a gigabyte
	BytesInGB = 1 << 30
)
