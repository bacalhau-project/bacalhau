package s3managed

import (
	"context"
	"fmt"
	"io"
	"os"
	"reflect"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/lib/envelope"
	"github.com/bacalhau-project/bacalhau/pkg/lib/gzip"
	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages"
	"github.com/bacalhau-project/bacalhau/pkg/publisher"
	s3helper "github.com/bacalhau-project/bacalhau/pkg/s3"
	"github.com/rs/zerolog/log"
)

const (
	ResultCompressionErrorMessage = "failed to compress execution result"
)

type PublisherParams struct {
	NCLPublisherProvider ncl.PublisherProvider
	NodeInfoProvider     models.BaseNodeInfoProvider
	LocalDir             string
	URLUploader          URLUploader
}

type Publisher struct {
	localDir             string
	uploader             URLUploader
	nclPublisherProvider ncl.PublisherProvider
}

func NewPublisher(params PublisherParams) *Publisher {
	return &Publisher{
		localDir:             params.LocalDir,
		uploader:             params.URLUploader,
		nclPublisherProvider: params.NCLPublisherProvider,
	}
}

// This publisher is considered always installed from the compute node's perspective,
// as it's the orchestrator that is reponsible for providing pre-signed URLs for this publisher.
// If the orchestrator does not have the managed S3 publisher installed,
// it will reject jobs that use it before sending them to the compute nodes.
func (p Publisher) IsInstalled(ctx context.Context) (bool, error) {
	return true, nil
}

func (p Publisher) ValidateJob(ctx context.Context, j models.Job) error {
	spec := j.Task().Publisher
	if !spec.IsType(models.PublisherS3Managed) {
		return s3helper.NewS3PublisherError(s3helper.BadRequestErrorCode,
			fmt.Sprintf("invalid publisher type. expected %s, but received: %s",
				models.PublisherS3Managed, spec.Type))
	}
	return nil
}

func (p Publisher) PublishResult(ctx context.Context, execution *models.Execution, resultPath string) (models.SpecConfig, error) {
	log.Ctx(ctx).Debug().
		Str("execution_id", execution.ID).
		Str("job_id", execution.Job.ID).
		Msg("Publishing results to managed S3 bucket using pre-signed URL")

	// Get the pre-signed URL for uploading the result file
	preSignedURL, err := p.getUploadURL(ctx, execution)
	if err != nil {
		return models.SpecConfig{}, bacerrors.New("failed to get pre-signed URL for managed S3 publisher").
			WithHint("Ensure that the node is connected to an orchestrator and the orchestrator is configured to support managed S3 publisher")
	}

	// Archive and compress the results
	targetFile, err := os.CreateTemp(p.localDir, "bacalhau-execution-result-*.tar.gz")
	if err != nil {
		return models.SpecConfig{}, err
	}
	defer func() { _ = targetFile.Close() }()
	defer os.Remove(targetFile.Name())

	err = gzip.Compress(resultPath, targetFile)
	if err != nil {
		return models.SpecConfig{}, bacerrors.Wrap(err, ResultCompressionErrorMessage)
	}

	// Flush gzip buffer to ensure all data is written
	if err := targetFile.Sync(); err != nil {
		return models.SpecConfig{}, bacerrors.Wrap(err, ResultCompressionErrorMessage)
	}

	// Reset file to read from beginning
	_, err = targetFile.Seek(0, io.SeekStart)
	if err != nil {
		return models.SpecConfig{}, bacerrors.Wrap(err, ResultCompressionErrorMessage)
	}

	// Upload the result file using the pre-signed URL
	if err := p.uploader.Upload(ctx, preSignedURL, targetFile.Name()); err != nil {
		log.Ctx(ctx).Error().
			Err(err).
			Str("execution_id", execution.ID).
			Str("job_id", execution.Job.ID).
			Str("result_path", resultPath).
			Msg("Failed to upload result file to managed S3 bucket")

		return models.SpecConfig{}, bacerrors.New("failed to upload result file to managed S3 bucket").
			WithHint("Verify orchestrator configuration and ensure that the pre-signed URL expiration period is not too short")
	}

	log.Ctx(ctx).Debug().
		Str("execution_id", execution.ID).
		Str("job_id", execution.Job.ID).
		Str("result_path", resultPath).
		Msg("published result to managed S3 bucket")

	return models.SpecConfig{
		Type: models.StorageSourceS3Managed,
		Params: SourceSpec{
			JobID:       execution.Job.ID,
			ExecutionID: execution.ID,
		}.ToMap(),
	}, nil
}

// getUploadURL retrieves a pre-signed URL for uploading the result file.
// The URL is provided by the orchestrator via an NCL Publisher
func (p Publisher) getUploadURL(ctx context.Context, execution *models.Execution) (string, error) {
	// Use the NCL publisher provider to get the upload URL
	nclMessagePublisher, err := p.nclPublisherProvider.GetPublisher()
	if err != nil {
		return "", fmt.Errorf("failed to get NCL publisher: %w", err)
	}
	if nclMessagePublisher == nil {
		return "", fmt.Errorf("node is disconnected from the orchestrator, or the managed S3 publisher is not supported by the orchestrator")
	}

	message := envelope.NewMessage(messages.ManagedPublisherPreSignURLRequest{
		JobID:       execution.Job.ID,
		ExecutionID: execution.ID,
	}).
		WithMetadataValue(envelope.KeyMessageType, messages.ManagedPublisherPreSignURLRequestType)

	response, err := nclMessagePublisher.Request(ctx, ncl.NewPublishRequest(message))
	if err != nil {
		return "", fmt.Errorf("failed to request pre-signed URL: %w", err)
	}

	responseMessage, ok := response.Payload.(*messages.ManagedPublisherPreSignURLResponse)
	if !ok {
		return "", envelope.NewErrUnexpectedPayloadType("ManagedPublisherPreSignURLResponse", reflect.TypeOf(response.Payload).String())
	}

	log.Ctx(ctx).Debug().
		Str("execution_id", responseMessage.ExecutionID).
		Str("job_id", responseMessage.JobID).
		Str("url", responseMessage.PreSignedURL).
		Msg("Received pre-signed URL for managed S3 publisher")

	return responseMessage.PreSignedURL, nil
}

// Compile-time check that publisher implements the correct interface:
var _ publisher.Publisher = (*Publisher)(nil)
