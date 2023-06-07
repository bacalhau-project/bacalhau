//go:build !unix

package printer

import (
	"os"
	"syscall"
)

var ShutdownSignals = []os.Signal{
	os.Interrupt,
	syscall.SIGTERM,
}
