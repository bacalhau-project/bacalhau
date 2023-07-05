//go:build unix

package util

import (
	"os"
	"syscall"
)

var ShutdownSignals = []os.Signal{
	os.Interrupt,
	syscall.SIGTERM,
}
