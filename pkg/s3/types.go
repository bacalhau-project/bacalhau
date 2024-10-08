package s3

import (
	"fmt"

	"github.com/fatih/structs"
	"github.com/mitchellh/mapstructure"

	"github.com/bacalhau-project/bacalhau/pkg/models"
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
		return NewS3InputSourceError(BadRequestErrorCode, "invalid s3 storage params: bucket cannot be empty")
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
		return NewS3DownloaderError(BadRequestErrorCode, "invalid s3 signed storage params: signed url cannot be empty")
	}
	return c.SourceSpec.Validate()
}

func (c PreSignedResultSpec) ToMap() map[string]interface{} {
	return structs.Map(c)
}

func DecodeSourceSpec(spec *models.SpecConfig) (SourceSpec, error) {
	if !spec.IsType(models.StorageSourceS3) {
		return SourceSpec{}, NewS3InputSourceError(BadRequestErrorCode, "invalid storage source type. expected "+models.StorageSourceS3+" but received: "+spec.Type)
	}
	inputParams := spec.Params
	if inputParams == nil {
		return SourceSpec{}, NewS3InputSourceError(BadRequestErrorCode, "invalid storage source params. cannot be nil")
	}

	var c SourceSpec
	if err := mapstructure.Decode(spec.Params, &c); err != nil {
		return c, err
	}

	return c, c.Validate()
}

func DecodePreSignedResultSpec(spec *models.SpecConfig) (PreSignedResultSpec, error) {
	if !spec.IsType(models.StorageSourceS3PreSigned) {
		return PreSignedResultSpec{}, NewS3InputSourceError(BadRequestErrorCode,
			"invalid storage source type. expected "+models.StorageSourceS3PreSigned+" but received: "+spec.Type)
	}

	inputParams := spec.Params
	if inputParams == nil {
		return PreSignedResultSpec{}, NewS3InputSourceError(BadRequestErrorCode, "invalid signed result params. cannot be nil")
	}

	var c PreSignedResultSpec
	if err := mapstructure.Decode(spec.Params, &c); err != nil {
		return c, err
	}

	return c, c.Validate()
}

type PublisherSpec struct {
	Bucket   string `json:"Bucket"`
	Key      string `json:"Key"`
	Endpoint string `json:"Endpoint"`
	Region   string `json:"Region"`
}

func (c PublisherSpec) Validate() error {
	if c.Bucket == "" {
		return NewS3PublisherError(BadRequestErrorCode, "invalid s3 params. bucket cannot be empty")
	}
	if c.Key == "" {
		return NewS3PublisherError(BadRequestErrorCode, "invalid s3 params. key cannot be empty")
	}
	return nil
}

func (c PublisherSpec) ToMap() map[string]interface{} {
	return structs.Map(c)
}

func DecodePublisherSpec(spec *models.SpecConfig) (PublisherSpec, error) {
	if !spec.IsType(models.PublisherS3) {
		return PublisherSpec{}, NewS3PublisherError(BadRequestErrorCode,
			fmt.Sprintf("invalid publisher type. expected %s, but received: %s",
				models.PublisherS3, spec.Type))
	}
	inputParams := spec.Params
	if inputParams == nil {
		return PublisherSpec{}, NewS3PublisherError(BadRequestErrorCode, "invalid publisher params. cannot be nil")
	}

	var c PublisherSpec
	if err := mapstructure.Decode(spec.Params, &c); err != nil {
		return c, err
	}

	return c, c.Validate()
}

type PublisherOption func(p *PublisherSpec)

func WithPublisherEndpoint(e string) PublisherOption {
	return func(p *PublisherSpec) {
		p.Endpoint = e
	}
}

func WithPublisherRegion(r string) PublisherOption {
	return func(p *PublisherSpec) {
		p.Region = r
	}
}

func NewPublisherSpec(bucket string, key string, opts ...PublisherOption) (*models.SpecConfig, error) {
	spec := &PublisherSpec{
		Bucket: bucket,
		Key:    key,
	}

	for _, opt := range opts {
		opt(spec)
	}

	if err := spec.Validate(); err != nil {
		return nil, err
	}

	return &models.SpecConfig{
		Type:   models.PublisherS3,
		Params: spec.ToMap(),
	}, nil
}
