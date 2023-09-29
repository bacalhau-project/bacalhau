package jobstore

import "encoding/json"

// Envelope provides a wrapper around types that can be stored in a jobstore.
// It takes responsibility for ser/de for the wrapped type, ensuring that
type Envelope[T any] struct {
	Body      T
	marshal   MarshalFunc
	unmarshal UnmarshalFunc
}

type Option[T any] func(*Envelope[T])

func WithBody[T any](body T) Option[T] {
	return func(e *Envelope[T]) {
		e.Wrap(body)
	}
}

func WithMarshaller[T any](m MarshalFunc, u UnmarshalFunc) Option[T] {
	return func(e *Envelope[T]) {
		e.marshal = m
		e.unmarshal = u
	}
}

func NewEnvelope[T any](options ...Option[T]) *Envelope[T] {
	e := &Envelope[T]{
		marshal:   json.Marshal,
		unmarshal: json.Unmarshal,
	}

	for _, opt := range options {
		opt(e)
	}
	return e
}

func (e *Envelope[T]) Copy() *Envelope[T] {
	return &Envelope[T]{
		marshal:   e.marshal,
		unmarshal: e.unmarshal,
	}
}

func (e *Envelope[T]) Wrap(obj T) {
	e.Body = obj
}

func (e *Envelope[T]) Unwrap() T {
	return e.Body
}

func (e *Envelope[T]) Serialize() ([]byte, error) {
	return e.marshal(e)
}

func (e *Envelope[T]) Deserialize(data []byte) (*Envelope[T], error) {
	env := e.Copy()
	err := e.unmarshal(data, env)
	return env, err
}
