package proxy

import "errors"

type Result[T any] struct {
	Response T
	Error    string
}

func newResult[T any](response *T, err error) Result[T] {
	if err != nil {
		return Result[T]{
			Error: err.Error(),
		}
	}

	return Result[T]{
		Response: *response,
	}
}

func (r *Result[T]) Rehydrate() (T, error) {
	var e error = nil

	if r.Error != "" {
		e = errors.New(r.Error)
	}

	return r.Response, e
}
