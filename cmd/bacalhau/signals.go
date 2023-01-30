//go:build !unix

package bacalhau

import "os"

var ShutdownSignals = []os.Signal{
	os.Interrupt,
}
