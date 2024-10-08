package s3

import (
	"errors"

	"github.com/fatih/structs"
	"github.com/mitchellh/mapstructure"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/s3"
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

func DecodeSourceSpec(spec *models.SpecConfig) (SourceSpec, error) {
	if !spec.IsType(models.StorageSourceS3) {
		return SourceSpec{}, s3.NewS3InputSourceError(s3.BadRequestErrorCode, "invalid storage source type. expected "+models.StorageSourceS3+", but received: "+spec.Type)
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

func DecodePreSignedResultSpec(spec *models.SpecConfig) (PreSignedResultSpec, error) {
	if !spec.IsType(models.StorageSourceS3PreSigned) {
		return PreSignedResultSpec{}, s3.NewS3InputSourceError(s3.BadRequestErrorCode, "invalid storage source type. expected "+models.StorageSourceS3PreSigned+", but received: "+spec.Type)
	}

	inputParams := spec.Params
	if inputParams == nil {
		return PreSignedResultSpec{}, s3.NewS3InputSourceError(s3.BadRequestErrorCode, "invalid signed result params. cannot be nil")
	}

	var c PreSignedResultSpec
	if err := mapstructure.Decode(spec.Params, &c); err != nil {
		return c, err
	}

	return c, c.Validate()
}
