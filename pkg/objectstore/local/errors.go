package local

import (
	"errors"
	"fmt"
)

var ErrInvalidPrefixName = func(s string) error { return fmt.Errorf("invalid prefix name: %s", s) }
var ErrNoSuchPrefix = func(s string) error { return fmt.Errorf("unknown prefix: %s", s) }
var ErrDatabaseClosed = errors.New("database is closed")
