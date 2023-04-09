package s3

import (
	"context"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/bacalhau-project/bacalhau/pkg/job"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/publisher"
	s3helper "github.com/bacalhau-project/bacalhau/pkg/s3"
	"github.com/rs/zerolog/log"
)

type PublisherParams struct {
	LocalDir       string
	ClientProvider *s3helper.ClientProvider
}

// Compile-time check that Verifier implements the correct interface:
var _ publisher.Publisher = (*Publisher)(nil)

type Publisher struct {
	localDir       string
	clientProvider *s3helper.ClientProvider
}

func NewPublisher(params PublisherParams) *Publisher {
	return &Publisher{
		localDir:       params.LocalDir,
		clientProvider: params.ClientProvider,
	}
}

// IsInstalled returns true if the S3 client is installed.
func (publisher *Publisher) IsInstalled(_ context.Context) (bool, error) {
	return publisher.clientProvider.IsInstalled(), nil
}

// ValidateJob validates the job spec and returns an error if the job is invalid.
func (publisher *Publisher) ValidateJob(_ context.Context, j model.Job) error {
	_, err := DecodeConfig(j.Spec.PublisherSpec)
	return err
}

func (publisher *Publisher) PublishResult(
	ctx context.Context,
	executionID string,
	j model.Job,
	resultPath string,
) (model.StorageSpec, error) {
	spec, err := DecodeConfig(j.Spec.PublisherSpec)
	if err != nil {
		return model.StorageSpec{}, err
	}

	if spec.Archive {
		return publisher.publishArchive(ctx, spec, executionID, j, resultPath)
	} else {
		return publisher.publishDirectory(ctx, spec, executionID, j, resultPath)
	}
}

func (publisher *Publisher) publishArchive(
	ctx context.Context,
	spec PublisherConfig,
	executionID string,
	j model.Job,
	resultPath string,
) (model.StorageSpec, error) {
	client := publisher.clientProvider.GetClient(spec.Endpoint, spec.Region)
	key := ParsePublishedKey(spec.Key, executionID, j, true)

	// Create a new GZIP writer that writes to the file.
	targetFile, err := os.CreateTemp(publisher.localDir, "bacalhau-archive-*.tar.gz")
	if err != nil {
		return model.StorageSpec{}, err
	}
	defer targetFile.Close()
	defer os.Remove(targetFile.Name())

	err = archiveDirectory(resultPath, targetFile)
	if err != nil {
		return model.StorageSpec{}, err
	}

	// reopen the archived file to upload it
	toUploadFile, err := os.Open(targetFile.Name())
	if err != nil {
		return model.StorageSpec{}, err
	}
	defer toUploadFile.Close()

	// Upload the GZIP archive to S3.
	res, err := client.Uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket:            aws.String(spec.Bucket),
		Key:               aws.String(key),
		Body:              toUploadFile,
		ChecksumAlgorithm: types.ChecksumAlgorithmSha256,
	})
	if err != nil {
		return model.StorageSpec{}, err
	}
	log.Debug().Msgf("Uploaded s3://%s/%s", spec.Bucket, aws.ToString(res.Key))

	return model.StorageSpec{
		StorageSource: model.StorageSourceS3,
		Name:          job.GetPublishedStorageName(executionID, j),
		S3: &model.S3StorageSpec{
			Bucket:         spec.Bucket,
			Key:            key,
			Endpoint:       spec.Endpoint,
			Region:         spec.Region,
			ChecksumSHA256: aws.ToString(res.ChecksumSHA256),
			VersionID:      aws.ToString(res.VersionID),
		},
	}, nil
}

func (publisher *Publisher) publishDirectory(
	ctx context.Context,
	spec PublisherConfig,
	executionID string,
	j model.Job,
	resultPath string,
) (model.StorageSpec, error) {
	client := publisher.clientProvider.GetClient(spec.Endpoint, spec.Region)
	key := ParsePublishedKey(spec.Key, executionID, j, false)

	// Walk the directory tree and upload each file to S3.
	err := filepath.Walk(resultPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil // skip directories
		}
		// Read the file contents.
		data, err := os.Open(path)
		if err != nil {
			return err
		}
		defer data.Close()

		relativePath, err := filepath.Rel(resultPath, path)
		if err != nil {
			return err
		}
		// Upload the file to S3.
		res, err := client.Uploader.Upload(ctx, &s3.PutObjectInput{
			Bucket:            aws.String(spec.Bucket),
			Key:               aws.String(filepath.Join(key, relativePath)),
			Body:              data,
			ChecksumAlgorithm: types.ChecksumAlgorithmSha256,
		})
		if err != nil {
			return err
		}
		log.Debug().Msgf("Uploaded s3://%s/%s", spec.Bucket, aws.ToString(res.Key))
		return nil
	})

	if err != nil {
		return model.StorageSpec{}, err
	}

	return model.StorageSpec{
		StorageSource: model.StorageSourceS3,
		Name:          job.GetPublishedStorageName(executionID, j),
		S3: &model.S3StorageSpec{
			Bucket:   spec.Bucket,
			Key:      key,
			Endpoint: spec.Endpoint,
			Region:   spec.Region,
		},
	}, nil
}
