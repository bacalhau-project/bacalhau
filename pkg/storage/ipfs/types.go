package ipfs

import (
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/fatih/structs"
	"github.com/mitchellh/mapstructure"
)

type Source struct {
	CID string `json:"CID"`
}

func (c Source) Validate() error {
	if c.CID == "" {
		return fmt.Errorf("invalid ipfs params. cid cannot be empty")
	}
	return nil
}

func (c Source) ToMap() map[string]interface{} {
	return structs.Map(c)
}

func DecodeSpec(spec *models.SpecConfig) (Source, error) {
	if !spec.IsType(models.StorageSourceIPFS) {
		return Source{}, fmt.Errorf("invalid storage source type. expected %s, but received: %s",
			models.StorageSourceIPFS, spec.Type)
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
