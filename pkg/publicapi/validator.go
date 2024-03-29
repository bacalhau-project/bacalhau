package publicapi

import (
	"github.com/bacalhau-project/bacalhau/pkg/lib/bad"
	"github.com/go-playground/validator/v10"
)

type validatable interface {
	Validate() error
}

// CustomValidator is a custom validator for echo framework
// that does the following:
// - Uses go-playground/validator for validation if validator tags are present
// - Uses Validate() method if the struct implements validatable interface
type CustomValidator struct {
	validator *validator.Validate
}

func (cv *CustomValidator) Validate(i interface{}) error {
	if err := cv.validator.Struct(i); err != nil {
		return bad.Input(err)
	}
	if v, ok := i.(validatable); ok {
		err := v.Validate()
		if err != nil {
			return bad.Input(err)
		}
	}
	return nil
}

func NewCustomValidator() *CustomValidator {
	return &CustomValidator{
		validator: validator.New(),
	}
}
