package test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/configenv"
	"github.com/bacalhau-project/bacalhau/pkg/downloader"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	s3publisher "github.com/bacalhau-project/bacalhau/pkg/publisher/s3"
	s3helper "github.com/bacalhau-project/bacalhau/pkg/s3"
	s3storage "github.com/bacalhau-project/bacalhau/pkg/storage/s3"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const defaultBucket = "bacalhau-test-datasets"
const defaultRegion = "eu-west-1"
const defaultEndpoint = "https://s3-eu-west-1.amazonaws.com"

type HelperSuiteParams struct {
	Bucket     string
	Region     string
	Endpoint   string
	BasePrefix string
}

type HelperSuite struct {
	suite.Suite
	// Fields that are initialized once in the constructor
	Bucket         string
	Region         string
	Endpoint       string
	BasePrefix     string
	ClientProvider *s3helper.ClientProvider
	Publisher      *s3publisher.Publisher
	Storage        *s3storage.StorageProvider

	// Fields that are initialized in SetupSuite for every test
	Prefix      string
	JobID       string
	ExecutionID string
	RunID       string
	TempDir     string
	Ctx         context.Context
}

func NewTestHelper(t *testing.T, params HelperSuiteParams) *HelperSuite {
	if params.Bucket == "" {
		params.Bucket = defaultBucket
	}
	if params.Region == "" {
		params.Region = defaultRegion
	}
	if params.Endpoint == "" {
		params.Endpoint = defaultEndpoint
	}
	params.BasePrefix = strings.Trim(params.BasePrefix, "/")

	awsConfig, err := s3helper.DefaultAWSConfig()
	require.NoError(t, err)

	clientProvider := s3helper.NewClientProvider(s3helper.ClientProviderParams{
		AWSConfig: awsConfig,
	})

	publisher := s3publisher.NewPublisher(s3publisher.PublisherParams{
		ClientProvider: clientProvider,
	})

	storage := s3storage.NewStorage(s3storage.StorageProviderParams{
		ClientProvider: clientProvider,
	})

	return &HelperSuite{
		Bucket:         params.Bucket,
		Region:         params.Region,
		Endpoint:       params.Endpoint,
		BasePrefix:     params.BasePrefix,
		ClientProvider: clientProvider,
		Publisher:      publisher,
		Storage:        storage,
	}
}

// SetupSuite creates a unique prefix for the test suite to avoid collisions.
func (s *HelperSuite) SetupSuite() {
	s.Require().NoError(config.Set(configenv.Local))
	if !s.HasValidCredentials() {
		s.T().Skip("No valid AWS credentials found")
	}

	// Create a fake file with a unique name to test if we have access to the default bucket
	// This is needed because the bucket may exist but as a client we don't have access to read/write
	fakeFile := fmt.Sprintf("%s/%s", s.Bucket, uuid.NewString())
	_, err := s.GetClient().S3.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket: aws.String(s.Bucket),
		Key:    aws.String(fakeFile),
		Body:   strings.NewReader(""),
	})
	defer s.DeleteObjects(fakeFile)

	if err != nil {
		s.T().Skip("No access to S3 bucket " + s.Bucket)
	}

	// unique runID added to prefix to avoid collisions
	timestamp := time.Now().UTC().Format("20060102T150405") // yyyyMMddThhmmss // cspell:disable-line
	s.JobID = uuid.NewString()
	s.ExecutionID = uuid.NewString()
	s.RunID = timestamp + "-" + uuid.NewString()
	s.Prefix = s.BasePrefix + fmt.Sprintf("/%s/", s.RunID)
	s.TempDir = s.T().TempDir()
	s.Ctx = context.Background()
}

// HasValidCredentials returns true if the S3 client is installed.
func (s *HelperSuite) HasValidCredentials() bool {
	return s.ClientProvider.IsInstalled()
}

// GetClient returns a client for the bucket's region and endpoint.
func (s *HelperSuite) GetClient() *s3helper.ClientWrapper {
	return s.ClientProvider.GetClient(s.Endpoint, s.Region)
}

// PreparePublisherSpec returns a publisher spec with the bucket, prefix, and endpoint.
func (s *HelperSuite) PreparePublisherSpec(compressed bool) s3helper.PublisherSpec {
	prefix := s.Prefix + uuid.NewString() + "_"
	if compressed {
		prefix += "compressed.tar.gz"
	} else {
		prefix += "uncompressed/"
	}
	return s3helper.PublisherSpec{
		Bucket:   s.Bucket,
		Key:      prefix,
		Compress: compressed,
		Region:   s.Region,
		Endpoint: s.Endpoint,
	}
}

// PrepareResultsPath creates local directories and files that mimic a result
// directory structure.
func (s *HelperSuite) PrepareResultsPath() string {
	resultPath, err := os.MkdirTemp(s.TempDir, "")
	s.Require().NoError(err)

	// Create stdout, stderr, and exitCode files
	s.Require().NoError(
		os.WriteFile(filepath.Join(resultPath, downloader.DownloadFilenameStdout), []byte(uuid.NewString()), downloader.DownloadFilePerm))
	s.Require().NoError(
		os.WriteFile(filepath.Join(resultPath, downloader.DownloadFilenameStderr), []byte(""), downloader.DownloadFilePerm))
	s.Require().NoError(
		os.WriteFile(filepath.Join(resultPath, downloader.DownloadFilenameExitCode), []byte("0"), downloader.DownloadFilePerm))

	// Create files in /outputs directory
	outputs := filepath.Join(resultPath, "outputs")
	s.Require().NoError(os.Mkdir(outputs, downloader.DownloadFolderPerm))
	for _, file := range []string{"1", "2"} {
		filePath := filepath.Join(outputs, file+".txt")
		err = os.WriteFile(filePath, []byte(file), downloader.DownloadFilePerm)
		s.Require().NoError(err)
	}

	// Create files in /outputs/nested directory
	nested := filepath.Join(outputs, "nested")
	s.Require().NoError(os.Mkdir(nested, downloader.DownloadFolderPerm))
	for _, file := range []string{"3", "4"} {
		filePath := filepath.Join(nested, file+".txt")
		err = os.WriteFile(filePath, []byte(file), downloader.DownloadFilePerm)
		s.Require().NoError(err)
	}
	return resultPath
}

// PublishResult publishes the resultPath to S3 and returns the published key.
func (s *HelperSuite) PublishResult(publisherConfig s3helper.PublisherSpec, resultPath string) (models.SpecConfig, error) {
	job := mock.Job()
	job.ID = s.JobID // to get predictable published key
	job.Task().Publisher = &models.SpecConfig{
		Type:   models.PublisherS3,
		Params: publisherConfig.ToMap(),
	}
	return s.Publisher.PublishResult(s.Ctx, &models.Execution{ID: s.ExecutionID, Job: job}, resultPath)
}

// PublishResultSilently publishes the resultPath to S3 and skip if no access.
func (s *HelperSuite) PublishResultSilently(publisherConfig s3helper.PublisherSpec, resultPath string) models.SpecConfig {
	// publish result to S3
	storageSpec, err := s.PublishResult(publisherConfig, resultPath)
	if err != nil {
		var ae smithy.APIError
		if errors.As(err, &ae) && ae.ErrorCode() == "AccessDenied" {
			s.T().Skip("No access to S3 bucket " + s.Bucket)
		}
	}
	s.Require().NoError(err)
	return storageSpec
}

// PrepareAndPublish publishes the resultPath to S3 and returns the published key.
func (s *HelperSuite) PrepareAndPublish(compressed bool) (models.SpecConfig, string) {
	publisherConfig := s.PreparePublisherSpec(compressed)
	resultPath := s.PrepareResultsPath()
	storageSpec := s.PublishResultSilently(publisherConfig, resultPath)
	return storageSpec, resultPath
}

// GetResult fetches the result from S3 and returns the local path.
func (s *HelperSuite) GetResult(published *models.SpecConfig) string {
	volume, err := s.Storage.PrepareStorage(
		s.Ctx,
		s.T().TempDir(),
		models.InputSource{
			Source: published,
			Target: "/", // ignored as it is the mount point within the job
		})
	s.Require().NoError(err)

	// if the input was an archive, then the returned source is the parent directory and not the archive itself.
	// we need to return the path to the archive.
	if strings.HasSuffix(published.Params["Key"].(string), ".tar.gz") {
		entries, err := os.ReadDir(volume.Source)
		s.Require().NoError(err)
		s.Require().NotEmpty(entries)
		return filepath.Join(volume.Source, entries[0].Name())
	}

	return volume.Source
}

func (s *HelperSuite) Destroy() {
	s.DeleteObjects(s.Prefix)
}

func (s *HelperSuite) DeleteObjects(prefix string) {
	svc := s.GetClient().S3
	objects := make([]types.ObjectIdentifier, 0)
	listInput := &s3.ListObjectsV2Input{
		Bucket: aws.String(s.Bucket),
		Prefix: aws.String(prefix),
	}
	listPaginator := s3.NewListObjectsV2Paginator(svc, listInput)
	for listPaginator.HasMorePages() {
		output, err := listPaginator.NextPage(context.Background())
		if err != nil {
			s.T().Logf("Failed to list objects while deleting %s: %v", prefix, err)
			return
		}
		for _, obj := range output.Contents {
			objects = append(objects, types.ObjectIdentifier{
				Key: obj.Key,
			})
		}
	}

	// Delete the objects
	deleteInput := &s3.DeleteObjectsInput{
		Bucket: aws.String(s.Bucket),
		Delete: &types.Delete{
			Objects: objects,
			Quiet:   true,
		},
	}
	_, err := svc.DeleteObjects(context.Background(), deleteInput)
	if err != nil {
		s.T().Logf("Failed to delete objects while deleting %s: %v", prefix, err)
	}
}
