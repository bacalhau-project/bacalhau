package main

import (
	"github.com/filecoin-project/bacalhau/testground/testcases"
	"github.com/testground/sdk-go/run"
)

func main() {
	run.InvokeMap(testcasesMap)
}

var testcasesMap = map[string]interface{}{
	"catFileToStdout": run.InitializedTestCaseFn(testcases.CatFileToStdout),
	"catFileToVolume": run.InitializedTestCaseFn(testcases.CatFileToVolume),
}
