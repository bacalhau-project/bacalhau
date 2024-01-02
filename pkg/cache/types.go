package cache

import (
	"errors"
)

var ErrCacheWrongType error = errors.New("requested cache exists with that name, but a different type")
var ErrCacheTooCostly error = errors.New("item too costly for cache")
var ErrCacheFull error = errors.New("cache is full")
