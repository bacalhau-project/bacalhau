package repo

import (
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/fatih/structs"
	"github.com/mitchellh/mapstructure"
)

var supportedTypes = []string{
	models.StorageSourceRepoClone,
	models.StorageSourceRepoCloneLFS,
}

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
	// Check if the spec.Type is in the supportedTypes slice
	isSupported := false
	for _, t := range supportedTypes {
		if spec.IsType(t) {
			isSupported = true
			break
		}
	}

	if !isSupported {
		return Source{}, fmt.Errorf("invalid storage source type. expected one of %v, but received: %s",
			supportedTypes, spec.Type)
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
