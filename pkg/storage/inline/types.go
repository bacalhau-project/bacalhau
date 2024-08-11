package inline

import (
	"fmt"

	"github.com/fatih/structs"
	"github.com/mitchellh/mapstructure"

	"github.com/bacalhau-project/bacalhau/pkg/lib/validate"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type Source struct {
	URL string `json:"URL"`
}

func (c Source) Validate() error {
	return validate.NotBlank(c.URL, "invalid inline params: url cannot be empty")
}

func (c Source) ToMap() map[string]interface{} {
	return structs.Map(c)
}

func DecodeSpec(spec *models.SpecConfig) (Source, error) {
	if !spec.IsType(models.StorageSourceInline) {
		return Source{}, fmt.Errorf("invalid storage source type. expected %s, but received: %s",
			models.StorageSourceInline, spec.Type)
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

func NewSpecConfig(url string) *models.SpecConfig {
	s := Source{URL: url}

	return &models.SpecConfig{
		Type:   models.StorageSourceInline,
		Params: s.ToMap(),
	}
}
