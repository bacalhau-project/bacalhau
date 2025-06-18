package s3managed

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	s3helper "github.com/bacalhau-project/bacalhau/pkg/s3"
	"github.com/rs/zerolog/log"
)

type PreSignedURLGeneratorParams struct {
	ClientProvider  *s3helper.ClientProvider
	PublisherConfig *types.S3ManagedPublisher
}

// PreSignedURLGenerator is responsible for generating pre-signed URLs for the managed S3 publisher.
type PreSignedURLGenerator struct {
	clientProvider  *s3helper.ClientProvider
	publisherConfig *types.S3ManagedPublisher
}

func NewPreSignedURLGenerator(params PreSignedURLGeneratorParams) (*PreSignedURLGenerator, error) {
	if err := params.PublisherConfig.Validate(); err != nil {
		return nil, err
	}
	return &PreSignedURLGenerator{
		clientProvider:  params.ClientProvider,
		publisherConfig: params.PublisherConfig,
	}, nil
}

// From the orchestrator's point of view the managed publisher is installed if the S3 client is available
// and mandatory configuration properties are set.
func (p *PreSignedURLGenerator) IsInstalled() bool {
	return p.clientProvider != nil &&
		p.publisherConfig != nil &&
		p.clientProvider.IsInstalled() &&
		p.publisherConfig.Validate() == nil
}

func (p *PreSignedURLGenerator) GeneratePreSignedPutURL(
	ctx context.Context,
	jobID string,
	executionID string,
) (string, error) {
	if jobID == "" || executionID == "" {
		return "", fmt.Errorf("jobID and executionID must be provided")
	}

	key := p.generateObjectKey(jobID, executionID)

	// TODO: Use default expiration if not configured
	expiration := p.publisherConfig.PreSignedURLExpiration.AsTimeDuration()

	client := p.clientProvider.GetClient(p.publisherConfig.Endpoint, p.publisherConfig.Region)

	// Create PUT request
	// Do not provide a body because we do not have the full execution result file.
	request := &s3.PutObjectInput{
		Bucket: &p.publisherConfig.Bucket,
		Key:    &key,
	}

	log.Ctx(ctx).Debug().
		Str("job_id", jobID).
		Str("execution_id", executionID).
		Str("bucket", p.publisherConfig.Bucket).
		Str("key", key).
		Msgf("Generating pre-signed PUT URL for S3 object")

	resp, err := client.PresignClient().PresignPutObject(ctx, request, s3.WithPresignExpires(expiration))
	if err != nil {
		return "", err
	}

	return resp.URL, nil
}

// GeneratePresignedGetURL creates a pre-signed URL for downloading an S3 object.
func (p *PreSignedURLGenerator) GeneratePresignedGetURL(
	ctx context.Context,
	jobID string,
	executionID string,
) (string, error) {
	if jobID == "" || executionID == "" {
		return "", fmt.Errorf("jobID and executionID must be provided")
	}

	key := p.generateObjectKey(jobID, executionID)

	// Use configured expiration time
	expiration := p.publisherConfig.PreSignedURLExpiration.AsTimeDuration()

	client := p.clientProvider.GetClient(p.publisherConfig.Endpoint, p.publisherConfig.Region)

	// Create GET request for downloading the result
	request := &s3.GetObjectInput{
		Bucket: &p.publisherConfig.Bucket,
		Key:    &key,
	}

	log.Ctx(ctx).Debug().
		Str("job_id", jobID).
		Str("execution_id", executionID).
		Str("bucket", p.publisherConfig.Bucket).
		Str("key", key).
		Msgf("Generating pre-signed GET URL for S3 object")

	// Generate pre-signed URL for GET operation
	resp, err := client.PresignClient().PresignGetObject(ctx, request, s3.WithPresignExpires(expiration))
	if err != nil {
		return "", err
	}

	return resp.URL, nil
}

// generateObjectKey constructs the S3 object key based on job and execution IDs.
func (p *PreSignedURLGenerator) generateObjectKey(jobID, executionID string) string {
	// Create key with prefix if available
	key := fmt.Sprintf("%s/%s.tar.gz", jobID, executionID)
	if p.publisherConfig.Key != "" {
		prefix := strings.TrimSuffix(p.publisherConfig.Key, "/")
		key = fmt.Sprintf("%s/%s", prefix, key)
	}

	return key
}
