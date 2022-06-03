package main

import (
	"github.com/filecoin-project/bacalhau/cmd/bacalhau"

	// Please don't remove the below.
	// It is an import for initialization - https://go.dev/ref/spec#Import_declarations
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
)

// Values for version are injected by the build.
var (
	VERSION = ""
)

func main() {
	bacalhau.Execute(VERSION)
}
