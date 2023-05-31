package s3

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec"
	spec_s3 "github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/s3"
	"github.com/bacalhau-project/bacalhau/pkg/publisher"
	s3helper "github.com/bacalhau-project/bacalhau/pkg/s3"
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
	_, err := DecodeSpec(j.Spec.PublisherSpec)
	return err
}

func (publisher *Publisher) PublishResult(
	ctx context.Context,
	executionID string,
	j model.Job,
	resultPath string,
) (spec.Storage, error) {
	s3spec, err := DecodeSpec(j.Spec.PublisherSpec)
	if err != nil {
		return spec.Storage{}, err
	}

	if s3spec.Compress {
		return publisher.publishArchive(ctx, s3spec, executionID, j, resultPath)
	}
	return publisher.publishDirectory(ctx, s3spec, executionID, j, resultPath)
}

func (publisher *Publisher) publishArchive(
	ctx context.Context,
	params Params,
	executionID string,
	j model.Job,
	resultPath string,
) (spec.Storage, error) {
	client := publisher.clientProvider.GetClient(params.Endpoint, params.Region)
	key := ParsePublishedKey(params.Key, executionID, j, true)

	// Create a new GZIP writer that writes to the file.
	targetFile, err := os.CreateTemp(publisher.localDir, "bacalhau-archive-*.tar.gz")
	if err != nil {
		return spec.Storage{}, err
	}
	defer targetFile.Close()
	defer os.Remove(targetFile.Name())

	err = archiveDirectory(resultPath, targetFile)
	if err != nil {
		return spec.Storage{}, err
	}

	// reset the archived file to read and upload it
	_, err = targetFile.Seek(0, io.SeekStart)
	if err != nil {
		return spec.Storage{}, err
	}

	// Upload the GZIP archive to S3.
	res, err := client.Uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket:            aws.String(params.Bucket),
		Key:               aws.String(key),
		Body:              targetFile,
		ChecksumAlgorithm: types.ChecksumAlgorithmSha256,
	})
	if err != nil {
		return spec.Storage{}, err
	}
	log.Debug().Msgf("Uploaded s3://%s/%s", params.Bucket, aws.ToString(res.Key))

	return (&spec_s3.S3StorageSpec{
		Bucket:         params.Bucket,
		Key:            key,
		ChecksumSHA256: aws.ToString(res.ChecksumSHA256),
		VersionID:      aws.ToString(res.VersionID),
		Endpoint:       params.Endpoint,
		Region:         params.Region,
	}).AsSpec(fmt.Sprintf("s3://%s/%s", params.Bucket, key), "TODO")
}

func (publisher *Publisher) publishDirectory(
	ctx context.Context,
	params Params,
	executionID string,
	j model.Job,
	resultPath string,
) (spec.Storage, error) {
	client := publisher.clientProvider.GetClient(params.Endpoint, params.Region)
	key := ParsePublishedKey(params.Key, executionID, j, false)

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
			Bucket:            aws.String(params.Bucket),
			Key:               aws.String(key + filepath.ToSlash(relativePath)),
			Body:              data,
			ChecksumAlgorithm: types.ChecksumAlgorithmSha256,
		})
		if err != nil {
			return err
		}
		log.Debug().Msgf("Uploaded s3://%s/%s", params.Bucket, aws.ToString(res.Key))
		return nil
	})

	if err != nil {
		return spec.Storage{}, err
	}

	return (&spec_s3.S3StorageSpec{
		Bucket:   params.Bucket,
		Key:      key,
		Endpoint: params.Endpoint,
		Region:   params.Region,
	}).AsSpec(fmt.Sprintf("s3://%s/%s", params.Bucket, key), "TODO")

}
