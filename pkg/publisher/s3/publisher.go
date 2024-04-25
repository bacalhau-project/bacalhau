package s3

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publisher"
	"github.com/bacalhau-project/bacalhau/pkg/repo"
	s3helper "github.com/bacalhau-project/bacalhau/pkg/s3"
)

type PublisherParams struct {
	Storage        repo.ComputeStorage
	ClientProvider *s3helper.ClientProvider
}

// Compile-time check that publisher implements the correct interface:
var _ publisher.Publisher = (*Publisher)(nil)

type Publisher struct {
	storage        repo.ComputeStorage
	clientProvider *s3helper.ClientProvider
}

func NewPublisher(params PublisherParams) *Publisher {
	return &Publisher{
		storage:        params.Storage,
		clientProvider: params.ClientProvider,
	}
}

// IsInstalled returns true if the S3 client is installed.
func (publisher *Publisher) IsInstalled(_ context.Context) (bool, error) {
	return publisher.clientProvider.IsInstalled(), nil
}

// ValidateJob validates the job spec and returns an error if the job is invalid.
func (publisher *Publisher) ValidateJob(_ context.Context, j models.Job) error {
	_, err := s3helper.DecodePublisherSpec(j.Task().Publisher)
	return err
}

func (publisher *Publisher) PublishResult(
	ctx context.Context,
	execution *models.Execution,
	resultPath string,
) (models.SpecConfig, error) {
	spec, err := s3helper.DecodePublisherSpec(execution.Job.Task().Publisher)
	if err != nil {
		return models.SpecConfig{}, err
	}

	client := publisher.clientProvider.GetClient(spec.Endpoint, spec.Region)
	key := ParsePublishedKey(spec.Key, execution, true)

	// Create a new GZIP writer that writes to the file.
	targetFile, err := publisher.storage.Create(fmt.Sprintf("bacalhau-archive-%d.tar.gz", time.Now().UTC().Unix()))
	if err != nil {
		return models.SpecConfig{}, err
	}
	defer targetFile.Close()
	defer os.Remove(targetFile.Name())

	err = archiveDirectory(resultPath, targetFile)
	if err != nil {
		return models.SpecConfig{}, err
	}

	// reset the archived file to read and upload it
	_, err = targetFile.Seek(0, io.SeekStart)
	if err != nil {
		return models.SpecConfig{}, err
	}

	putObjectInput := &s3.PutObjectInput{
		Bucket: aws.String(spec.Bucket),
		Key:    aws.String(key),
		Body:   targetFile,
	}

	// Only use SHA256 checksums if the endpoint is AWS, as it is
	// not supported by other S3-compatible providers, such as GCP buckets
	if client.IsAWSEndpoint() {
		putObjectInput.ChecksumAlgorithm = types.ChecksumAlgorithmSha256
	}

	// Upload the GZIP archive to S3.
	res, err := client.Uploader.Upload(ctx, putObjectInput)
	if err != nil {
		return models.SpecConfig{}, err
	}
	log.Debug().Msgf("Uploaded s3://%s/%s", spec.Bucket, aws.ToString(res.Key))

	return models.SpecConfig{
		Type: models.StorageSourceS3,
		Params: s3helper.SourceSpec{
			Bucket:         spec.Bucket,
			Key:            key,
			Endpoint:       spec.Endpoint,
			Region:         spec.Region,
			ChecksumSHA256: aws.ToString(res.ChecksumSHA256),
			VersionID:      aws.ToString(res.VersionID),
		}.ToMap(),
	}, nil
}
