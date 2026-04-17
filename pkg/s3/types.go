package s3

import (
	"fmt"

	"github.com/fatih/structs"
	"github.com/mitchellh/mapstructure"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type ObjectSummary struct {
	Key       *string
	ETag      *string
	VersionID *string
	Size      int64
	IsDir     bool
}

type SourceSpec struct {
	Bucket         string
	Key            string
	Filter         string
	Region         string
	Endpoint       string
	VersionID      string
	ChecksumSHA256 string
	Partition      PartitionConfig
}

func (c SourceSpec) Validate() error {
	if c.Bucket == "" {
		return NewS3InputSourceError(BadRequestErrorCode, "invalid s3 storage params: bucket cannot be empty")
	}
	return c.Partition.Validate()
}

func (c SourceSpec) ToMap() map[string]interface{} {
	return structs.Map(c)
}

type PreSignedResultSpec struct {
	SourceSpec
	PreSignedURL string
	Managed      bool
}

func (c PreSignedResultSpec) Validate() error {
	if c.PreSignedURL == "" {
		return NewS3DownloaderError(BadRequestErrorCode, "invalid s3 signed storage params: signed url cannot be empty")
	}
	if !c.Managed {
		return c.SourceSpec.Validate()
	}
	return nil
}

func (c PreSignedResultSpec) ToMap() map[string]interface{} {
	return structs.Map(c)
}

func DecodeSourceSpec(spec *models.SpecConfig) (SourceSpec, error) {
	if !spec.IsType(models.StorageSourceS3) {
		return SourceSpec{}, NewS3InputSourceError(
			BadRequestErrorCode,
			fmt.Sprintf("invalid storage source type. expected %s but received: %s", models.StorageSourceS3, spec.Type))
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
			fmt.Sprintf("invalid storage source type. expected %s but received: %s",
				models.StorageSourceS3PreSigned, spec.Type))
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

type Encoding string

const (
	EncodingGzip  Encoding = "gzip"
	EncodingPlain Encoding = "plain"
)

func (e Encoding) IsValid() bool {
	return e == EncodingGzip || e == EncodingPlain
}

type PublisherSpec struct {
	Bucket   string   `json:"Bucket"`
	Key      string   `json:"Key"`
	Endpoint string   `json:"Endpoint"`
	Region   string   `json:"Region"`
	Encoding Encoding `json:"Encoding"`
}

func (c PublisherSpec) Validate() error {
	if c.Bucket == "" {
		return NewS3PublisherError(BadRequestErrorCode, "invalid s3 params. bucket cannot be empty")
	}
	if c.Key == "" {
		return NewS3PublisherError(BadRequestErrorCode, "invalid s3 params. key cannot be empty")
	}
	if c.Encoding != "" && !c.Encoding.IsValid() {
		return NewS3PublisherError(BadRequestErrorCode, "invalid s3 params. encoding must be either 'plain' or 'gzip'")
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
