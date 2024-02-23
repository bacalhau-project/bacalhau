package s3

import (
	"errors"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/fatih/structs"
	"github.com/mitchellh/mapstructure"
)

type SourceSpec struct {
	Bucket         string
	Key            string
	Filter         string
	Region         string
	Endpoint       string
	VersionID      string
	ChecksumSHA256 string
}

func (c SourceSpec) Validate() error {
	if c.Bucket == "" {
		return errors.New("invalid s3 storage params: bucket cannot be empty")
	}
	return nil
}

func (c SourceSpec) ToMap() map[string]interface{} {
	return structs.Map(c)
}

type PreSignedResultSpec struct {
	SourceSpec
	PreSignedURL string
}

func (c PreSignedResultSpec) Validate() error {
	if c.PreSignedURL == "" {
		return errors.New("invalid s3 signed storage params: signed url cannot be empty")
	}
	return c.SourceSpec.Validate()
}

func (c PreSignedResultSpec) ToMap() map[string]interface{} {
	return structs.Map(c)
}

type PublisherSpec struct {
	Bucket   string `json:"Bucket"`
	Key      string `json:"Key"`
	Endpoint string `json:"Endpoint"`
	Region   string `json:"Region"`
}

func DecodeSourceSpec(spec *models.SpecConfig) (SourceSpec, error) {
	if !spec.IsType(models.StorageSourceS3) {
		return SourceSpec{}, errors.New("invalid storage source type. expected " + models.StorageSourceS3 + ", but received: " + spec.Type)
	}
	inputParams := spec.Params
	if inputParams == nil {
		return SourceSpec{}, errors.New("invalid storage source params. cannot be nil")
	}

	var c SourceSpec
	if err := mapstructure.Decode(spec.Params, &c); err != nil {
		return c, err
	}

	return c, c.Validate()
}

func DecodePreSignedResultSpec(spec *models.SpecConfig) (PreSignedResultSpec, error) {
	if !spec.IsType(models.StorageSourceS3PreSigned) {
		return PreSignedResultSpec{}, errors.New(
			"invalid storage source type. expected " + models.StorageSourceS3PreSigned + ", but received: " + spec.Type)
	}

	inputParams := spec.Params
	if inputParams == nil {
		return PreSignedResultSpec{}, errors.New("invalid signed result params. cannot be nil")
	}

	var c PreSignedResultSpec
	if err := mapstructure.Decode(spec.Params, &c); err != nil {
		return c, err
	}

	return c, c.Validate()
}

func DecodePublisherSpec(spec *models.SpecConfig) (PublisherSpec, error) {
	if !spec.IsType(models.PublisherS3) {
		return PublisherSpec{}, fmt.Errorf("invalid publisher type. expected %s, but received: %s",
			models.PublisherS3, spec.Type)
	}
	inputParams := spec.Params
	if inputParams == nil {
		return PublisherSpec{}, fmt.Errorf("invalid publisher params. cannot be nil")
	}

	var c PublisherSpec
	if err := mapstructure.Decode(spec.Params, &c); err != nil {
		return c, err
	}

	return c, c.Validate()
}

func (c PublisherSpec) Validate() error {
	if c.Bucket == "" {
		return fmt.Errorf("invalid s3 params. bucket cannot be empty")
	}
	if c.Key == "" {
		return fmt.Errorf("invalid s3 params. key cannot be empty")
	}
	return nil
}

func (c PublisherSpec) ToMap() map[string]interface{} {
	return structs.Map(c)
}
