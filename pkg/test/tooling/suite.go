package tooling

import (
	"github.com/filecoin-project/bacalhau/pkg/system"
)

type TestSuite struct {
	Cm *system.CleanupManager
}

// return noop executors for all engines
func NewTestSuite() *TestSuite {
	return &TestSuite{
		Cm: system.NewCleanupManager(),
	}
}
