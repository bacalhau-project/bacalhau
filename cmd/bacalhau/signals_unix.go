//go:build unix

package bacalhau

import (
	"os"
	"syscall"
)

var ShutdownSignals = []os.Signal{
	os.Interrupt,
	syscall.SIGTERM,
}
