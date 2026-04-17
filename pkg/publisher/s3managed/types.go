package s3managed

import (
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/s3"
	"github.com/fatih/structs"
	"github.com/mitchellh/mapstructure"
)

type SourceSpec struct {
	JobID       string `json:"job_id"`
	ExecutionID string `json:"execution_id"`
}

func (c SourceSpec) ToMap() map[string]any {
	return structs.Map(c)
}

func (c SourceSpec) Validate() error {
	if c.JobID == "" {
		return s3.NewS3InputSourceError(s3.BadRequestErrorCode, "invalid storage params: job id cannot be empty")
	}
	if c.ExecutionID == "" {
		return s3.NewS3InputSourceError(s3.BadRequestErrorCode, "invalid storage params: execution id cannot be empty")
	}

	return nil
}

func DecodeSourceSpec(spec *models.SpecConfig) (SourceSpec, error) {
	if !spec.IsType(models.StorageSourceS3Managed) {
		return SourceSpec{}, s3.NewS3InputSourceError(
			s3.BadRequestErrorCode,
			fmt.Sprintf("invalid storage source type. expected %s but received: %s", models.StorageSourceS3Managed, spec.Type))
	}

	inputParams := spec.Params
	if inputParams == nil {
		return SourceSpec{}, s3.NewS3InputSourceError(s3.BadRequestErrorCode, "invalid storage source params. cannot be nil")
	}

	var c SourceSpec
	if err := mapstructure.Decode(spec.Params, &c); err != nil {
		return c, err
	}

	return c, c.Validate()
}
