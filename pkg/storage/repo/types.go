package repo

import (
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/fatih/structs"
	"github.com/mitchellh/mapstructure"
)

type Source struct {
	Repo string
}

func (c Source) Validate() error {
	if c.Repo == "" {
		return fmt.Errorf("invalid repo params. repo cannot be empty")
	}
	return nil
}

func (c Source) ToMap() map[string]interface{} {
	return structs.Map(c)
}

func DecodeSpec(spec *models.SpecConfig) (Source, error) {
	if !spec.IsType(models.StorageSourceRepoClone) {
		return Source{}, fmt.Errorf("invalid storage source type. expected %s, but received: %s",
			models.StorageSourceRepoClone, spec.Type)
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
