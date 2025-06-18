package s3managed

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/s3"
)

// ResultTransformer transforms execution results from S3 managed publisher to S3 pre-signed URLs.
type ResultTransformer struct {
	urlGenerator *PreSignedURLGenerator
}

// NewResultTransformer creates a new ResultTransformer.
func NewResultTransformer(urlGenerator *PreSignedURLGenerator) *ResultTransformer {
	return &ResultTransformer{
		urlGenerator: urlGenerator,
	}
}

// Transform checks if the result is from a managed S3 publisher job
// and transforms it to use a pre-signed URL.
func (t *ResultTransformer) Transform(ctx context.Context, spec *models.SpecConfig) error {
	// Skip transformation if not a managed S3 publisher
	if spec.Type != models.StorageSourceS3Managed {
		return nil
	}

	// Skip if the URL generator is not installed
	if !t.urlGenerator.IsInstalled() {
		return nil
	}

	sourceSpec, err := DecodeSourceSpec(spec)
	if err != nil {
		return err
	}

	preSignedURL, err := t.urlGenerator.GeneratePresignedGetURL(ctx, sourceSpec.JobID, sourceSpec.ExecutionID)
	if err != nil {
		return err
	}

	spec.Type = models.StorageSourceS3PreSigned
	spec.Params = s3.PreSignedResultSpec{
		SourceSpec:   s3.SourceSpec{},
		PreSignedURL: preSignedURL,
		Managed:      true,
	}.ToMap()
	return nil
}
