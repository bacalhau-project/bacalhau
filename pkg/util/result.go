package util

// An envelope that contains a value and an error.
// Can be convenient for using in channels or other concurrency patterns
// where you want to return a value and an error together instead of
// creating two separate channels for success and error.
// Example
// 	result := <-getResultAsync() // block until the result is available
// 	if result.Error != nil {
// 		// handle error
// 	} else {
//		// use result.Value
// 	}

type Result[T any] struct {
	Value T
	Error error
}

func NewResult[T any](value T, err error) Result[T] {
	return Result[T]{
		Value: value,
		Error: err,
	}
}
