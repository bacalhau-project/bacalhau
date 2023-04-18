package bprotocol

import "errors"

type Result[T any] struct {
	Response T
	Error    string
}

func (r *Result[T]) Rehydrate() (T, error) {
	var e error = nil

	if r.Error != "" {
		e = errors.New(r.Error)
	}

	return r.Response, e
}
