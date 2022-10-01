package bacerrors

import (
	"github.com/serum-errors/go-serum"
)

type BacalhauErrorInterface interface {
	Error() string
	SetError(error)

	Code() string

	Message() string
	SetMessage(string)

	Details() map[string]interface{}
}

type BacalhauError struct {
	serumError serum.Error
	details    map[string]interface{}
	theCode    string //nolint:unused
	theMessage string
	theCause   error
}

func (e *BacalhauError) Error() string {
	return e.serumError.Error()
}
