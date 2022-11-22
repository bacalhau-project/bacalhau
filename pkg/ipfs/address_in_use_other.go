//go:build !unix

package ipfs

import (
	"errors"
)

var addressInUseError = errors.New("unsure what error other OSes give for this so have an error which won't match anything")
