package s3

import (
	"errors"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/fatih/structs"
	"github.com/mitchellh/mapstructure"
)

type Source struct {
	Bucket         string
	Key            string
	Region         string
	Endpoint       string
	VersionID      string
	ChecksumSHA256 string
}

func (c Source) Validate() error {
	if c.Bucket == "" {
		return errors.New("invalid s3 storage params: bucket cannot be empty")
	}
	if c.Key == "" {
		return errors.New("invalid s3 storage params: key cannot be empty")
	}
	return nil
}

func (c Source) ToMap() map[string]interface{} {
	return structs.Map(c)
}

func DecodeSpec(spec *models.SpecConfig) (Source, error) {
	if spec.Type != models.StorageSourceS3 {
		return Source{}, errors.New("invalid storage source type. expected " + models.StorageSourceS3 + ", but received: " + spec.Type)
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
