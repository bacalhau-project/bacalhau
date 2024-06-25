package publicapi

import (
	"github.com/labstack/echo/v4"
)

// normalizable is an interface that defines a Normalize method.
type normalizable interface {
	Normalize()
}

// CustomBinder is a custom binder that extends the default binder with normalization.
type CustomBinder struct {
	defaultBinder echo.Binder
}

// NewCustomBinder creates a new CustomBinder with the default binder.
func NewCustomBinder() *CustomBinder {
	return &CustomBinder{
		defaultBinder: &echo.DefaultBinder{},
	}
}

// Bind binds and validates the request body, then normalizes if it implements the normalizable interface.
func (cb *CustomBinder) Bind(i interface{}, c echo.Context) error {
	if err := cb.defaultBinder.Bind(i, c); err != nil {
		return err
	}
	if normalizer, ok := i.(normalizable); ok {
		normalizer.Normalize()
	}
	return nil
}
