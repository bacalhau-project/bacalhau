package main

import (
	"github.com/filecoin-project/bacalhau/cmd/bacalhau"
)

// Values for version are injected by the build.
var (
	VERSION = ""
)

func main() {
	bacalhau.Execute(VERSION)
}
