package s3

import (
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/fatih/structs"
	"github.com/mitchellh/mapstructure"
)

type PublisherConfig struct {
	Bucket   string `json:"Bucket"`
	Key      string `json:"Key"`
	Endpoint string `json:"Endpoint"`
	Region   string `json:"Region"`
	Archive  bool   `json:"Compress"`
}

func DecodeConfig(spec model.PublisherSpec) (PublisherConfig, error) {
	if spec.Type != model.PublisherS3 {
		return PublisherConfig{}, fmt.Errorf("invalid publisher type. expected %s, but received: %s",
			model.PublisherS3, spec.Type)
	}
	var c PublisherConfig
	if err := mapstructure.Decode(spec.Config, &c); err != nil {
		return c, err
	}

	return c, c.Validate()
}

func (c PublisherConfig) Validate() error {
	if c.Bucket == "" {
		return fmt.Errorf("invalid publisher config. bucket cannot be empty")
	}
	if c.Key == "" {
		return fmt.Errorf("invalid publisher config. key cannot be empty")
	}
	return nil
}

func (c PublisherConfig) ToMap() map[string]interface{} {
	return structs.Map(c)
}
