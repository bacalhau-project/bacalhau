//go:build integration || !unit

package s3

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	s3helper "github.com/bacalhau-project/bacalhau/pkg/s3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

const bucket = "bacalhau-test-datasets"
const prefix = "integration-tests-publisher/"
const region = "eu-west-1"

var jobID = uuid.NewString()
var executionID = uuid.NewString()

type PublisherTestSuite struct {
	suite.Suite
	publisher *Publisher
	tempDir   string
}

func (s *PublisherTestSuite) SetupSuite() {
	cfg, err := s3helper.DefaultAWSConfig()
	s.Require().NoError(err)

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

func (s *PublisherTestSuite) TestValidateJob() {
	for _, tc := range []struct {
		name    string
		config  PublisherConfig
		invalid bool
	}{
		{
			name: "valid",
			config: PublisherConfig{
				Bucket: bucket,
				Key:    prefix + uuid.New().String(),
			},
		}, {
			name: "valid with endpoint and region",
			config: PublisherConfig{
				Bucket:   bucket,
				Key:      prefix + uuid.New().String(),
				Endpoint: "http://localhost:4566",
				Region:   "eu-west-1",
			},
		},
		{
			name: "invalid bucket",
			config: PublisherConfig{
				Bucket: "",
				Key:    prefix + uuid.New().String(),
			},
			invalid: true,
		}, {
			name: "invalid key",
			config: PublisherConfig{
				Bucket: bucket,
				Key:    "",
			},
			invalid: true,
		},
	} {
		s.Run(tc.name, func() {
			err := s.publisher.ValidateJob(context.Background(), model.Job{
				Spec: model.Spec{
					PublisherSpec: model.PublisherSpec{
						Type:   model.PublisherS3,
						Config: tc.config.ToMap(),
					},
				},
			})
			if tc.invalid {
				s.Error(err)
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *PublisherTestSuite) TestInvalidValidateJobType() {
	s.Require().Error(s.publisher.ValidateJob(context.Background(), model.Job{
		Spec: model.Spec{
			PublisherSpec: model.PublisherSpec{
				Type: model.PublisherIpfs,
				Config: PublisherConfig{
					Bucket: bucket,
					Key:    prefix + uuid.New().String(),
				}.ToMap(),
			},
		},
	}))
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
			name:        "compressed with naming patter",
			key:         prefix + "pattern_compressed/{jobID}/{executionID}",
			expectedKey: prefix + "pattern_compressed/" + jobID + "/" + executionID + ".tar.gz",
			archived:    true,
		},
	} {
		s.Run(tc.name, func() {
			if skipMessage != "" {
				s.T().Skip(skipMessage)
			}
			ctx := context.Background()
			storageSpec, err := s.publish(ctx, PublisherConfig{
				Bucket:  bucket,
				Key:     tc.key,
				Region:  region,
				Archive: tc.archived,
			})

			if err != nil {
				var ae smithy.APIError
				if errors.As(err, &ae) && ae.ErrorCode() == "AccessDenied" {
					skipMessage = "No access to S3 bucket " + bucket
					s.T().Skip(skipMessage)
				}
			}
			s.Require().NoError(err)

			s.Equal(tc.expectedKey, storageSpec.S3.Key)
			s.Equal(bucket, storageSpec.S3.Bucket)
			s.Equal(region, storageSpec.S3.Region)

			if tc.archived {
				s.NotEmptyf(storageSpec.S3.ChecksumSHA256, "ChecksumSHA256 should not be empty")
				s.NotEmptyf(storageSpec.S3.VersionID, "VersionID should not be empty")
				dir := s.decompress(storageSpec)
				s.equalLocalContent("1", filepath.Join(dir, "1.txt"))
				s.equalLocalContent("2", filepath.Join(dir, "2.txt"))
				s.equalLocalContent("3", filepath.Join(dir, "nested", "3.txt"))
				s.equalLocalContent("4", filepath.Join(dir, "nested", "4.txt"))

			} else {
				s.Empty(storageSpec.S3.ChecksumSHA256, "ChecksumSHA256 should be empty")
				s.Empty(storageSpec.S3.VersionID, "VersionID should be empty")
				s.equalS3Content("1", storageSpec, "1.txt")
				s.equalS3Content("2", storageSpec, "2.txt")
				s.equalS3Content("3", storageSpec, "nested/3.txt")
				s.equalS3Content("4", storageSpec, "nested/4.txt")
			}
		})
	}
}

func (s *PublisherTestSuite) publish(ctx context.Context, publisherConfig PublisherConfig) (model.StorageSpec, error) {
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

	job := model.Job{
		Metadata: model.Metadata{
			ID: jobID,
		},
		Spec: model.Spec{
			PublisherSpec: model.PublisherSpec{
				Type:   model.PublisherS3,
				Config: publisherConfig.ToMap(),
			},
		},
	}
	return s.publisher.PublishResult(ctx, executionID, job, resultPath)
}

func (s *PublisherTestSuite) equalS3Content(expected string, uploaded model.StorageSpec, suffix string) {
	ctx := context.Background()
	client := s.publisher.clientProvider.GetClient("", region)
	resp, err := client.S3.GetObject(ctx, &s3.GetObjectInput{
		Bucket:       aws.String(uploaded.S3.Bucket),
		Key:          aws.String(uploaded.S3.Key + suffix),
		ChecksumMode: types.ChecksumModeEnabled,
	})
	s.Require().NoError(err)
	defer resp.Body.Close()
	if uploaded.S3.ChecksumSHA256 != "" {
		s.Equal(uploaded.S3.ChecksumSHA256, aws.ToString(resp.ChecksumSHA256))
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

func (s *PublisherTestSuite) decompress(uploaded model.StorageSpec) string {
	outputFile, err := os.CreateTemp(s.tempDir, "")
	s.Require().NoError(err)
	defer outputFile.Close()

	_, err = s.publisher.clientProvider.GetClient("", region).Downloader.Download(context.Background(),
		outputFile, &s3.GetObjectInput{
			Bucket: aws.String(uploaded.S3.Bucket),
			Key:    aws.String(uploaded.S3.Key),
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
				if err := os.MkdirAll(target, model.DownloadFolderPerm); err != nil {
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
