package system

import (
	"errors"
	"time"
)

var ErrorTimeout = errors.New("timed out")

type timeoutResult struct {
	Result interface{}
	Error  error
}

// wait for a file to appear that is owned by us
func Timeout(timeAllowed time.Duration, handler func() (interface{}, error)) (interface{}, error) {
	result := make(chan timeoutResult, 1)
	go func() {
		innerResult, err := handler()
		result <- timeoutResult{
			Result: innerResult,
			Error:  err,
		}
	}()
	select {
	case <-time.After(timeAllowed):
		return nil, ErrorTimeout
	case result := <-result:
		return result.Result, result.Error
	}
}
