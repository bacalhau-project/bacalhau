package s3

import (
	"fmt"
	"reflect"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/fatih/structs"
	"github.com/mitchellh/mapstructure"
)

type Params struct {
	Bucket   string `json:"Bucket"`
	Key      string `json:"Key"`
	Endpoint string `json:"Endpoint"`
	Region   string `json:"Region"`
	Compress bool   `json:"Compress"`
}

func DecodeSpec(spec *models.SpecConfig) (Params, error) {
	if spec.Type != models.PublisherS3 {
		return Params{}, fmt.Errorf("invalid publisher type. expected %s, but received: %s",
			models.PublisherS3, spec.Type)
	}
	inputParams := spec.Params
	if inputParams == nil {
		return Params{}, fmt.Errorf("invalid publisher params. cannot be nil")
	}

	// convert compress to bool
	if _, ok := inputParams["Compress"]; ok && reflect.TypeOf(inputParams["Compress"]).Kind() == reflect.String {
		inputParams["Compress"] = inputParams["Compress"] == "true"
	}
	if _, ok := inputParams["compress"]; ok && reflect.TypeOf(inputParams["compress"]).Kind() == reflect.String {
		inputParams["compress"] = inputParams["compress"] == "true"
	}

	var c Params
	if err := mapstructure.Decode(spec.Params, &c); err != nil {
		return c, err
	}

	return c, c.Validate()
}

func (c Params) Validate() error {
	if c.Bucket == "" {
		return fmt.Errorf("invalid s3 params. bucket cannot be empty")
	}
	if c.Key == "" {
		return fmt.Errorf("invalid s3 params. key cannot be empty")
	}
	return nil
}

func (c Params) ToMap() map[string]interface{} {
	return structs.Map(c)
}
