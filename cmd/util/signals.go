//go:build !unix

package util

import "os"

var ShutdownSignals = []os.Signal{
	os.Interrupt,
}
