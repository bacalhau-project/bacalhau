package urldownload

import (
	"errors"
	"fmt"

	"github.com/fatih/structs"
	"github.com/mitchellh/mapstructure"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type Source struct {
	URL string
}

func (c Source) Validate() error {
	if c.URL == "" {
		return errors.New("invalid url storage params: url cannot be empty")
	}
	if _, err := IsURLSupported(c.URL); err != nil {
		return fmt.Errorf("invalid url storage params: %w", err)
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

func NewSpecConfig(url string) (*models.SpecConfig, error) {
	s := Source{URL: url}
	if err := s.Validate(); err != nil {
		return nil, fmt.Errorf("creating %s storage spec: %w", models.StorageSourceURL, err)
	}
	return &models.SpecConfig{
		// TODO(forrest) [refactor] the type definition ought to live in this package.
		Type:   models.StorageSourceURL,
		Params: s.ToMap(),
	}, nil
}
