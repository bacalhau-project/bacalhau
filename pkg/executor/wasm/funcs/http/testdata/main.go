package testdata

import (
	"embed"
	"io/fs"
)

//go:embed main.wasm
var file embed.FS

// Program returns the WASM program
func Program() (b []byte) {
	b, err := fs.ReadFile(file, "main.wasm")
	if err != nil {
		panic(err)
	}
	return
}
