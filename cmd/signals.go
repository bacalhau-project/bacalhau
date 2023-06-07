//go:build !unix

package cmd

import "os"

var ShutdownSignals = []os.Signal{
	os.Interrupt,
}
