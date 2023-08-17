package s3

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/fatih/structs"
	"github.com/mitchellh/mapstructure"
)

type SourceSpec struct {
	Bucket         string
	Key            string
	Region         string
	Endpoint       string
	VersionID      string
	ChecksumSHA256 string
}

func (c SourceSpec) Validate() error {
	if c.Bucket == "" {
		return errors.New("invalid s3 storage params: bucket cannot be empty")
	}
	if c.Key == "" {
		return errors.New("invalid s3 storage params: key cannot be empty")
	}
	return nil
}

func (c SourceSpec) ToMap() map[string]interface{} {
	return structs.Map(c)
}

type PublisherSpec struct {
	Bucket   string `json:"Bucket"`
	Key      string `json:"Key"`
	Endpoint string `json:"Endpoint"`
	Region   string `json:"Region"`
	Compress bool   `json:"Compress"`
}

func DecodeSourceSpec(spec *models.SpecConfig) (SourceSpec, error) {
	if spec.Type != models.StorageSourceS3 {
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

func DecodePublisherSpec(spec *models.SpecConfig) (PublisherSpec, error) {
	if spec.Type != models.PublisherS3 {
		return PublisherSpec{}, fmt.Errorf("invalid publisher type. expected %s, but received: %s",
			models.PublisherS3, spec.Type)
	}
	inputParams := spec.Params
	if inputParams == nil {
		return PublisherSpec{}, fmt.Errorf("invalid publisher params. cannot be nil")
	}

	// convert compress to bool
	if _, ok := inputParams["Compress"]; ok && reflect.TypeOf(inputParams["Compress"]).Kind() == reflect.String {
		inputParams["Compress"] = inputParams["Compress"] == "true"
	}
	if _, ok := inputParams["compress"]; ok && reflect.TypeOf(inputParams["compress"]).Kind() == reflect.String {
		inputParams["compress"] = inputParams["compress"] == "true"
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
