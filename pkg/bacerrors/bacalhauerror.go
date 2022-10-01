package bacerrors

import (
	"github.com/serum-errors/go-serum"
)

type BacalhauErrorInterface interface {
	Error() string
	Code() string
	Details() map[string]interface{}
}

type BacalhauError struct {
	serumError serum.Error
	details    map[string]interface{}
	cause      error
}

func (e *BacalhauError) Error() string {
	return e.serumError.Error()
}
