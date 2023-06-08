//go:build !unix

package handler

import "os"

var ShutdownSignals = []os.Signal{
	os.Interrupt,
}
