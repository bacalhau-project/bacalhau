package objectstore

import "fmt"

// ErrNotFound is used when a key is not found in a store
type ErrNotFound struct {
	key string
}

func NewErrNotFound(key string) ErrNotFound {
	return ErrNotFound{key: key}
}

func (e ErrNotFound) Error() string {
	return fmt.Sprintf("unable to find object with key: %s", e.key)
}

// ErrInvalidPrefix is used when a prefix is provided that does not
// match the required format, or in some cases where it does not exist
// where we expect it to.
type ErrInvalidPrefix struct {
	prefix string
}

func NewErrInvalidPrefix(prefix string) ErrInvalidPrefix {
	return ErrInvalidPrefix{prefix: prefix}
}

func (e ErrInvalidPrefix) Error() string {
	return fmt.Sprintf("unable to use invalid prefix: %s", e.prefix)
}
