package urldownload

import (
	"errors"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/fatih/structs"
	"github.com/mitchellh/mapstructure"
)

type Source struct {
	URL string
}

func (c Source) Validate() error {
	if c.URL == "" {
		return errors.New("invalid url storage params: url cannot be empty")
	}
	return nil
}

func (c Source) ToMap() map[string]interface{} {
	return structs.Map(c)
}

func DecodeSpec(spec *models.SpecConfig) (Source, error) {
	if !spec.IsType(models.StorageSourceURL) {
		return Source{}, errors.New("invalid storage source type. expected " + models.StorageSourceURL + ", but received: " + spec.Type)
	}
	inputParams := spec.Params
	if inputParams == nil {
		return Source{}, errors.New("invalid storage source params. cannot be nil")
	}

	var c Source
	if err := mapstructure.Decode(spec.Params, &c); err != nil {
		return c, err
	}

	return c, c.Validate()
}
