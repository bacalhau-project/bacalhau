package s3

import (
	"context"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/rs/zerolog/log"
)

type ResultSignerParams struct {
	ClientProvider *ClientProvider
	Expiration     time.Duration
}

type ResultSigner struct {
	clientProvider *ClientProvider
	expiration     time.Duration
}

func NewResultSigner(params ResultSignerParams) *ResultSigner {
	return &ResultSigner{
		clientProvider: params.ClientProvider,
		expiration:     params.Expiration,
	}
}

// Transform signs S3 SourceSpec with a pre-signed URL.
func (signer *ResultSigner) Transform(ctx context.Context, spec *models.SpecConfig) error {
	if spec.Type != models.StorageSourceS3 {
		return nil
	}

	if !signer.clientProvider.IsInstalled() {
		log.Ctx(ctx).Debug().Msg("AWS credentials not configured. Skipping signing.")
		return nil
	}

	sourceSpec, err := DecodeSourceSpec(spec)
	if err != nil {
		return err
	}

	if !strings.HasSuffix(sourceSpec.Key, ".tar.gz") {
		log.Ctx(ctx).Debug().Str("S3Key", sourceSpec.Key).Msg("Skipping signing because the result is not a tar.gz file.")
		return nil
	}
	client := signer.clientProvider.GetClient(sourceSpec.Endpoint, sourceSpec.Region)
	request := &s3.GetObjectInput{
		Bucket: &sourceSpec.Bucket,
		Key:    &sourceSpec.Key,
	}
	log.Ctx(ctx).Debug().Msgf("Signing URL for s3://%s/%s", sourceSpec.Bucket, sourceSpec.Key)

	resp, err := client.PresignClient().PresignGetObject(ctx, request, s3.WithPresignExpires(signer.expiration))
	if err != nil {
		return err
	}
	spec.Type = models.StorageSourceS3Signed
	spec.Params = SignedResultSpec{
		SourceSpec: sourceSpec,
		SignedURL:  resp.URL,
	}.ToMap()
	return nil
}
