package jobstore

import "github.com/bacalhau-project/bacalhau/pkg/lib/marshaller"

// Envelope provides a wrapper around types that can be stored in a jobstore.
// It takes responsibility for the wrapped type, ensuring that
type Envelope[T any] struct {
	Body       T
	marshaller marshaller.Marshaller
}

type Option[T any] func(*Envelope[T])

func WithBody[T any](body T) Option[T] {
	return func(e *Envelope[T]) {
		e.Wrap(body)
	}
}

func WithMarshaller[T any](marshaller marshaller.Marshaller) Option[T] {
	return func(e *Envelope[T]) {
		e.marshaller = marshaller
	}
}

func NewEnvelope[T any](options ...Option[T]) *Envelope[T] {
	e := &Envelope[T]{
		marshaller: marshaller.NewJSONMarshaller(),
	}

	for _, opt := range options {
		opt(e)
	}
	return e
}

func (e *Envelope[T]) Copy() *Envelope[T] {
	return &Envelope[T]{
		marshaller: e.marshaller,
	}
}

func (e *Envelope[T]) Wrap(obj T) {
	e.Body = obj
}

func (e *Envelope[T]) Unwrap() T {
	return e.Body
}

func (e *Envelope[T]) Serialize() ([]byte, error) {
	return e.marshaller.Marshal(e)
}

func (e *Envelope[T]) Deserialize(data []byte) (*Envelope[T], error) {
	env := e.Copy()
	err := e.marshaller.Unmarshal(data, env)
	return env, err
}
