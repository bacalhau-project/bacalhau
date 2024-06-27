package publicapi

import (
	"github.com/labstack/echo/v4"
)

// normalizable is an interface that defines a Normalize method.
type normalizable interface {
	Normalize()
}

// NormalizeBinder is a custom binder that extends the default binder with normalization.
type NormalizeBinder struct {
	defaultBinder echo.Binder
}

// NewNormalizeBinder creates a new NormalizeBinder with the default binder.
func NewNormalizeBinder() *NormalizeBinder {
	return &NormalizeBinder{
		defaultBinder: &echo.DefaultBinder{},
	}
}

// Bind binds and validates the request body, then normalizes if it implements the normalizable interface.
func (cb *NormalizeBinder) Bind(i interface{}, c echo.Context) error {
	if err := cb.defaultBinder.Bind(i, c); err != nil {
		return err
	}
	if normalizer, ok := i.(normalizable); ok {
		normalizer.Normalize()
	}
	return nil
}
