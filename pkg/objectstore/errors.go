package objectstore

import "fmt"

var ErrUnknownImplementation = fmt.Errorf("unknown objectstore implementation")
var ErrInvalidOption = fmt.Errorf("option type did not match implementation")
