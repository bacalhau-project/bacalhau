// Generated by Makefile - DO NOT EDIT.
package easter

import "embed"
import "io/fs"

//go:embed main.wasm
var file embed.FS

func Program() (b []byte) {
	b, err := fs.ReadFile(file, "main.wasm")
	if err != nil {
		panic(err)
	}
	return
}
