//go:build integration || !unit

package s3

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	s3helper "github.com/bacalhau-project/bacalhau/pkg/s3"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

const bucket = "bacalhau-test-datasets"
const region = "eu-west-1"
const endpoint = "https://s3-eu-west-1.amazonaws.com"

var jobID = uuid.NewString()
var executionID = uuid.NewString()

// Ensure unique prefix
var timestamp = time.Now().UTC().Format("20060102T150405") // yyyyMMddThhmmss
var prefix = fmt.Sprintf("integration-tests-publisher/%s-%s/", timestamp, executionID)

type PublisherTestSuite struct {
	suite.Suite
	publisher *Publisher
	tempDir   string
}

func (s *PublisherTestSuite) SetupSuite() {
	cfg, err := s3helper.DefaultAWSConfig()
	s.Require().NoError(err)
	if !s3helper.HasValidCredentials(cfg) {
		s.T().Skip("No valid AWS credentials found")
	}

	clientProvider := s3helper.NewClientProvider(s3helper.ClientProviderParams{
		AWSConfig: cfg,
	})

	s.publisher = NewPublisher(PublisherParams{
		ClientProvider: clientProvider,
	})
	s.tempDir = s.T().TempDir()
}

func (s *PublisherTestSuite) TearDownSuite() {
	s.delete(prefix)
}

func TestPublisherTestSuite(t *testing.T) {
	suite.Run(t, new(PublisherTestSuite))
}

func (s *PublisherTestSuite) TestIsInstalled() {
	ctx := context.Background()
	res, err := s.publisher.IsInstalled(ctx)
	s.Require().NoError(err)
	s.True(res)
}

func (s *PublisherTestSuite) TestDateSubstitution() {

	job := mock.Job()
	job.ID = jobID
	job.Task().Publisher = &models.SpecConfig{
		Type: models.PublisherS3,
		Params: s3helper.PublisherSpec{
			Bucket: "test",
			Key:    "{date}/{time}",
		}.ToMap(),
	}

	str := ParsePublishedKey("{date}/{time}", &models.Execution{ID: "e1", Job: job}, false)
	parts := strings.Split(str, "/")

	n := time.Now()
	s.Require().Equal(fmt.Sprintf("%d%d%d", n.Year(), n.Month(), n.Day()), parts[0], "date was incorrect")

	// Check the time is all numbers
	_, err := strconv.Atoi(parts[1])
	s.Require().NoError(err, "time was not numerc")
}

func (s *PublisherTestSuite) TestValidateJob() {
	for _, tc := range []struct {
		name    string
		config  s3helper.PublisherSpec
		invalid bool
	}{
		{
			name: "valid",
			config: s3helper.PublisherSpec{
				Bucket: bucket,
				Key:    prefix + uuid.New().String(),
			},
		}, {
			name: "valid with endpoint and region",
			config: s3helper.PublisherSpec{
				Bucket:   bucket,
				Key:      prefix + uuid.New().String(),
				Endpoint: "http://127.0.0.1:4566",
				Region:   "eu-west-1",
			},
		},
		{
			name: "invalid bucket",
			config: s3helper.PublisherSpec{
				Bucket: "",
				Key:    prefix + uuid.New().String(),
			},
			invalid: true,
		}, {
			name: "invalid key",
			config: s3helper.PublisherSpec{
				Bucket: bucket,
				Key:    "",
			},
			invalid: true,
		},
	} {
		s.Run(tc.name, func() {
			job := mock.Job()
			job.ID = jobID
			job.Task().Publisher = &models.SpecConfig{
				Type:   models.PublisherS3,
				Params: tc.config.ToMap(),
			}

			err := s.publisher.ValidateJob(context.Background(), *job)
			if tc.invalid {
				s.Error(err)
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *PublisherTestSuite) TestInvalidValidateJobType() {
	job := mock.Job()
	job.Task().Publisher = &models.SpecConfig{
		Type: "notS3",
		Params: s3helper.PublisherSpec{
			Bucket: bucket,
			Key:    prefix + uuid.New().String(),
		}.ToMap(),
	}
	s.Require().Error(s.publisher.ValidateJob(context.Background(), *job))
}

func (s *PublisherTestSuite) TestPublish() {
	if !s3helper.HasValidCredentials(s.publisher.clientProvider.GetConfig()) {
		s.T().Skip("No valid AWS credentials found")
	}

	// to fast skip remaining tests in case we don't have valid credentials with enough permissions
	skipMessage := ""

	for _, tc := range []struct {
		name        string
		key         string
		expectedKey string
		archived    bool
		region      string
		endpoint    string
		shouldFail  bool
	}{
		{
			name:        "uncompressed",
			key:         prefix + "simple_uncompressed",
			expectedKey: prefix + "simple_uncompressed/",
		},
		{
			name:        "uncompressed with trailing slash",
			key:         prefix + "simple_uncompressed-with-trailing-slash/",
			expectedKey: prefix + "simple_uncompressed-with-trailing-slash/",
		},
		{
			name:        "uncompressed with nested path",
			key:         prefix + "nested_uncompressed/with/nested/path/",
			expectedKey: prefix + "nested_uncompressed/with/nested/path/",
		},
		{
			name:        "uncompressed with naming pattern",
			key:         prefix + "pattern_uncompressed/{jobID}/{executionID}/",
			expectedKey: prefix + "pattern_uncompressed/" + jobID + "/" + executionID + "/",
		},
		{
			name:        "compressed",
			key:         prefix + "simple_compressed",
			expectedKey: prefix + "simple_compressed.tar.gz",
			archived:    true,
		},
		{
			name:        "compressed with extension",
			key:         prefix + "simple_compressed.tar.gz",
			expectedKey: prefix + "simple_compressed.tar.gz",
			archived:    true,
		},
		{
			name:        "compressed with nested path",
			key:         prefix + "nested_compressed/with/nested/path",
			expectedKey: prefix + "nested_compressed/with/nested/path.tar.gz",
			archived:    true,
		},
		{
			name:        "compressed with naming pattern",
			key:         prefix + "pattern_compressed/{jobID}/{executionID}",
			expectedKey: prefix + "pattern_compressed/" + jobID + "/" + executionID + ".tar.gz",
			archived:    true,
		},
		{
			name:        "explicit endpoint and region",
			key:         prefix + "simple_compressed_endpoint_and_region.tar.gz",
			expectedKey: prefix + "simple_compressed_endpoint_and_region.tar.gz",
			endpoint:    endpoint,
			region:      region,
			archived:    true,
		},
		{
			name:       "explicit wrong region",
			key:        prefix + "simple_compressed_wrong_region.tar.gz",
			region:     "us-east-1",
			archived:   true,
			shouldFail: true,
		},
	} {
		s.Run(tc.name, func() {
			if skipMessage != "" {
				s.T().Skip(skipMessage)
			}
			ctx := context.Background()
			params := s3helper.PublisherSpec{
				Bucket:   bucket,
				Key:      tc.key,
				Compress: tc.archived,
			}
			if tc.region == "" && tc.endpoint == "" {
				params.Region = region
			} else {
				params.Region = tc.region
				params.Endpoint = tc.endpoint
			}
			storageSpec, err := s.publish(ctx, params)

			if err != nil {
				var ae smithy.APIError
				if errors.As(err, &ae) && ae.ErrorCode() == "AccessDenied" {
					skipMessage = "No access to S3 bucket " + bucket
					s.T().Skip(skipMessage)
				}
				if tc.shouldFail {
					return
				}
			}
			s.Require().NoError(err)

			sourceSpec, err := s3helper.DecodeSourceSpec(&storageSpec)
			s.Require().NoError(err)

			s.Equal(tc.expectedKey, sourceSpec.Key)
			s.Equal(bucket, sourceSpec.Bucket)
			s.Equal(params.Region, sourceSpec.Region)
			s.Equal(params.Endpoint, sourceSpec.Endpoint)

			if tc.archived {
				s.NotEmptyf(sourceSpec.ChecksumSHA256, "ChecksumSHA256 should not be empty")
				s.NotEmptyf(sourceSpec.VersionID, "VersionID should not be empty")
				dir := s.decompress(sourceSpec)
				s.equalLocalContent("1", filepath.Join(dir, "1.txt"))
				s.equalLocalContent("2", filepath.Join(dir, "2.txt"))
				s.equalLocalContent("3", filepath.Join(dir, "nested", "3.txt"))
				s.equalLocalContent("4", filepath.Join(dir, "nested", "4.txt"))

			} else {
				s.Empty(sourceSpec.ChecksumSHA256, "ChecksumSHA256 should be empty")
				s.Empty(sourceSpec.VersionID, "VersionID should be empty")
				s.equalS3Content("1", sourceSpec, "1.txt")
				s.equalS3Content("2", sourceSpec, "2.txt")
				s.equalS3Content("3", sourceSpec, "nested/3.txt")
				s.equalS3Content("4", sourceSpec, "nested/4.txt")
			}
		})
	}
}

func (s *PublisherTestSuite) publish(ctx context.Context, publisherConfig s3helper.PublisherSpec) (models.SpecConfig, error) {
	resultPath, err := os.MkdirTemp(s.tempDir, "")
	s.Require().NoError(err)

	for _, file := range []string{"1", "2"} {
		filePath := filepath.Join(resultPath, file+".txt")
		err = os.WriteFile(filePath, []byte(file), 0644)
		s.Require().NoError(err)
	}

	nested := filepath.Join(resultPath, "nested")
	s.Require().NoError(os.Mkdir(nested, 0755))
	for _, file := range []string{"3", "4"} {
		filePath := filepath.Join(nested, file+".txt")
		err = os.WriteFile(filePath, []byte(file), 0644)
		s.Require().NoError(err)
	}

	job := mock.Job()
	job.ID = jobID
	job.Task().Publisher = &models.SpecConfig{
		Type:   models.PublisherS3,
		Params: publisherConfig.ToMap(),
	}
	return s.publisher.PublishResult(ctx, &models.Execution{ID: executionID, Job: job}, resultPath)
}

func (s *PublisherTestSuite) equalS3Content(expected string, uploaded s3helper.SourceSpec, suffix string) {
	ctx := context.Background()
	client := s.publisher.clientProvider.GetClient("", region)
	resp, err := client.S3.GetObject(ctx, &s3.GetObjectInput{
		Bucket:       aws.String(uploaded.Bucket),
		Key:          aws.String(uploaded.Key + suffix),
		ChecksumMode: types.ChecksumModeEnabled,
	})
	s.Require().NoError(err)
	defer resp.Body.Close()
	if uploaded.ChecksumSHA256 != "" {
		s.Equal(uploaded.ChecksumSHA256, aws.ToString(resp.ChecksumSHA256))
	}

	// Read the object body into a byte buffer
	buf := bytes.NewBuffer([]byte{})
	_, err = buf.ReadFrom(resp.Body)
	s.Require().NoError(err)
	s.Equal(expected, buf.String())
}

func (s *PublisherTestSuite) equalLocalContent(expected string, path string) {
	bytes, err := os.ReadFile(path)
	s.Require().NoError(err)
	s.Equal(expected, string(bytes))
}

func (s *PublisherTestSuite) decompress(uploaded s3helper.SourceSpec) string {
	outputFile, err := os.CreateTemp(s.tempDir, "")
	s.Require().NoError(err)
	defer outputFile.Close()

	_, err = s.publisher.clientProvider.GetClient("", region).Downloader.Download(context.Background(),
		outputFile, &s3.GetObjectInput{
			Bucket: aws.String(uploaded.Bucket),
			Key:    aws.String(uploaded.Key),
		})
	s.Require().NoError(err)

	destinationDir, err := os.MkdirTemp(s.tempDir, "")
	s.Require().NoError(err)
	s.Require().NoError(unarchiveToDirectory(outputFile.Name(), destinationDir))
	return destinationDir
}

func (s *PublisherTestSuite) delete(key string) {
	svc := s.publisher.clientProvider.GetClient("", region).S3
	objects := make([]types.ObjectIdentifier, 0)
	listInput := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(key),
	}
	listPaginator := s3.NewListObjectsV2Paginator(svc, listInput)
	for listPaginator.HasMorePages() {
		output, err := listPaginator.NextPage(context.Background())
		if err != nil {
			s.T().Logf("Failed to list objects while deleting %s: %v", key, err)
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
		Bucket: aws.String(bucket),
		Delete: &types.Delete{
			Objects: objects,
			Quiet:   true,
		},
	}
	_, err := svc.DeleteObjects(context.Background(), deleteInput)
	if err != nil {
		s.T().Logf("Failed to delete objects while deleting %s: %v", key, err)
	}
}

func unarchiveToDirectory(sourcePath string, targetDir string) error {
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	gzr, err := gzip.NewReader(sourceFile)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// The target location where the dir/file should be created.
		target := filepath.Join(targetDir, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			// Create the directory.
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0755); err != nil {
					return err
				}
			}
		case tar.TypeReg:
			// Create the file.
			outFile, err := os.Create(target)
			if err != nil {
				return err
			}
			defer outFile.Close()

			// Write the contents of the file.
			if _, err := io.Copy(outFile, tr); err != nil {
				return err
			}
		default:
			return err
		}
	}
	return nil
}
