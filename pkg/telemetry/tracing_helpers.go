package telemetry

import (
	"io"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// RecordErrorOnSpan records the error returned by the function in the given span.
// Should be used like `return RecordErrorOnSpan(span)(c.client.NetworkRemove(ctx, networkID))`
func RecordErrorOnSpan(span trace.Span) func(error) error {
	return func(err error) error {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		return err
	}
}

// RecordErrorOnSpanTwo is similar to RecordErrorOnSpan but the function being called takes an additional parameter.
func RecordErrorOnSpanTwo[T any](span trace.Span) func(T, error) (T, error) {
	return func(t T, err error) (T, error) {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		return t, err
	}
}

// RecordErrorOnSpanThree is similar to RecordError but the function being called takes an additional parameter.
func RecordErrorOnSpanThree[T any, S any](span trace.Span) func(T, S, error) (T, S, error) {
	return func(t T, s S, err error) (T, S, error) {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		return t, s, err
	}
}

// RecordErrorOnSpanReadCloserAndClose is similar to RecordError but takes an additional parameter that is an io.ReadCloser.
// The function will end the given span when the io.ReadCloser is closed. The caller is expected to _not_ call end on
// the span.
func RecordErrorOnSpanReadCloserAndClose(span trace.Span) func(io.ReadCloser, error) (io.ReadCloser, error) {
	return func(closer io.ReadCloser, err error) (io.ReadCloser, error) {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		return spanClosingReader{
			delegate: closer,
			span:     span,
		}, err
	}
}

// RecordErrorOnSpanReadCloserTwoAndClose is similar to RecordErrorOnSpanReadCloserAndClose but takes an additional parameter.
func RecordErrorOnSpanReadCloserTwoAndClose[T any](span trace.Span) func(io.ReadCloser, T, error) (io.ReadCloser, T, error) {
	return func(closer io.ReadCloser, t T, err error) (io.ReadCloser, T, error) {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		return spanClosingReader{
			delegate: closer,
			span:     span,
		}, t, err
	}
}

// RecordErrorOnSpanOneChannel is similar to RecordErrorOnSpanTwo but one parameter is a channel.
func RecordErrorOnSpanOneChannel[T any](span trace.Span) func(<-chan T, error) (<-chan T, error) {
	return func(ts <-chan T, err error) (<-chan T, error) {
		if err != nil {
			// Assume the standard pattern of returning nil for the first parameter when also returning an error
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			span.End()
			return nil, err
		}

		newTS := make(chan T)
		go func() {
			defer close(newTS)
			defer span.End()
			newTS <- <-ts
		}()
		return newTS, nil
	}
}

// RecordErrorOnSpanTwoChannels is similar to RecordErrorOnSpanTwo but both parameters are channels.
func RecordErrorOnSpanTwoChannels[T any](span trace.Span) func(<-chan T, <-chan error) (<-chan T, <-chan error) {
	return func(ts <-chan T, errors <-chan error) (<-chan T, <-chan error) {
		newErrors := make(chan error, 1)
		newTS := make(chan T)
		go func() {
			defer close(newErrors)
			defer close(newTS)
			defer span.End()
			select {
			case err := <-errors:
				newErrors <- err
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
			case t := <-ts:
				// Need to listen for this as well to finish this select statement, otherwise it wouldn't finish if
				// no error was sent.
				newTS <- t
			}
		}()
		return newTS, newErrors
	}
}

var _ io.ReadCloser = spanClosingReader{}

type spanClosingReader struct {
	delegate io.ReadCloser
	span     trace.Span
}

func (s spanClosingReader) Read(p []byte) (n int, err error) {
	return s.delegate.Read(p)
}

func (s spanClosingReader) Close() error {
	defer s.span.End()
	return s.delegate.Close()
}
