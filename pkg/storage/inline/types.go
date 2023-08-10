package inline

import (
	"errors"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/lib/validate"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/fatih/structs"
	"github.com/mitchellh/mapstructure"
)

type Source struct {
	URL string `json:"URL"`
}

func (c Source) Validate() error {
	if validate.IsBlank(c.URL) {
		return errors.New("invalid inline params: url cannot be empty")
	}
	return nil
}

func (c Source) ToMap() map[string]interface{} {
	return structs.Map(c)
}

func DecodeSpec(spec *models.SpecConfig) (Source, error) {
	if spec.Type != models.StorageSourceInline {
		return Source{}, fmt.Errorf("invalid storage source type. expected %s, but received: %s",
			models.StorageSourceInline, spec.Type)
	}
	inputParams := spec.Params
	if inputParams == nil {
		return Source{}, fmt.Errorf("invalid storage source params. cannot be nil")
	}

	var c Source
	if err := mapstructure.Decode(spec.Params, &c); err != nil {
		return c, err
	}

	return c, c.Validate()
}
