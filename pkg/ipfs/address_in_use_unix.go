//go:build unix

package ipfs

import "syscall"

var addressInUseError = syscall.EADDRINUSE
