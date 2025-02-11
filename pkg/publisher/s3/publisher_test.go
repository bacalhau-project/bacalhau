//go:build integration || !unit

package s3_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/aws/smithy-go"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/lib/gzip"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	s3publisher "github.com/bacalhau-project/bacalhau/pkg/publisher/s3"
	s3helper "github.com/bacalhau-project/bacalhau/pkg/s3"
	s3test "github.com/bacalhau-project/bacalhau/pkg/s3/test"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
)

type PublisherTestSuite struct {
	*s3test.HelperSuite
}

func TestPublisherTestSuite(t *testing.T) {
	helperSuite := s3test.NewTestHelper(t, s3test.HelperSuiteParams{
		BasePrefix: "integration-tests-publisher",
	})
	suite.Run(t, &PublisherTestSuite{HelperSuite: helperSuite})
}

// TearDownSuite deletes all objects in the bucket with the prefix.
func (s *PublisherTestSuite) TearDownSuite() {
	s.Destroy()
}

func (s *PublisherTestSuite) TestIsInstalled() {
	ctx := context.Background()
	res, err := s.Publisher.IsInstalled(ctx)
	s.Require().NoError(err)
	s.True(res)
}

func (s *PublisherTestSuite) TestDateSubstitution() {

	key := s.Prefix + "{date}/{time}"
	job := mock.Job()
	job.Task().Publisher = &models.SpecConfig{
		Type: models.PublisherS3,
		Params: s3helper.PublisherSpec{
			Bucket: s.Bucket,
			Key:    key,
		}.ToMap(),
	}

	str := s3publisher.ParsePublishedKey(key, &models.Execution{ID: "e1", Job: job}, false)
	parts := strings.Split(str, "/")

	n := time.Now()
	s.Require().Equal(fmt.Sprintf("%d%02d%02d", n.Year(), n.Month(), n.Day()), parts[2], "date was incorrect")

	// Check the time is all numbers
	_, err := strconv.Atoi(parts[3])
	s.Require().NoError(err, "time was not numeric")
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
				Bucket: s.Bucket,
				Key:    s.Prefix + uuid.New().String(),
			},
		}, {
			name: "valid with endpoint and region",
			config: s3helper.PublisherSpec{
				Bucket:   s.Bucket,
				Key:      s.Prefix + uuid.New().String(),
				Endpoint: "http://127.0.0.1:4566",
				Region:   "eu-west-1",
			},
		},
		{
			name: "invalid bucket",
			config: s3helper.PublisherSpec{
				Bucket: "",
				Key:    s.Prefix + uuid.New().String(),
			},
			invalid: true,
		}, {
			name: "invalid key",
			config: s3helper.PublisherSpec{
				Bucket: s.Bucket,
				Key:    "",
			},
			invalid: true,
		},
	} {
		s.Run(tc.name, func() {
			job := mock.Job()
			job.ID = s.JobID
			job.Task().Publisher = &models.SpecConfig{
				Type:   models.PublisherS3,
				Params: tc.config.ToMap(),
			}

			err := s.Publisher.ValidateJob(context.Background(), *job)
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
			Bucket: s.Bucket,
			Key:    s.Prefix + uuid.New().String(),
		}.ToMap(),
	}
	s.Require().Error(s.Publisher.ValidateJob(context.Background(), *job))
}

func (s *PublisherTestSuite) TestPublish() {
	// to fast skip remaining tests in case we don't have valid credentials with enough permissions
	skipMessage := ""

	for _, tc := range []struct {
		name        string
		key         string
		expectedKey string
		encoding    s3helper.Encoding
		region      string
		endpoint    string
		shouldFail  bool
	}{
		{
			name:        "plain encoding",
			key:         s.Prefix + "simple_plain",
			expectedKey: s.Prefix + "simple_plain/",
			encoding:    s3helper.EncodingPlain,
		},
		{
			name:        "plain encoding with trailing slash",
			key:         s.Prefix + "simple_plain-with-trailing-slash/",
			expectedKey: s.Prefix + "simple_plain-with-trailing-slash/",
			encoding:    s3helper.EncodingPlain,
		},
		{
			name:        "plain encoding with nested path",
			key:         s.Prefix + "nested_plain/with/nested/path/",
			expectedKey: s.Prefix + "nested_plain/with/nested/path/",
			encoding:    s3helper.EncodingPlain,
		},
		{
			name:        "plain encoding with naming pattern",
			key:         s.Prefix + "pattern_plain/{jobID}/{executionID}/",
			expectedKey: s.Prefix + "pattern_plain/" + s.JobID + "/" + s.ExecutionID + "/",
			encoding:    s3helper.EncodingPlain,
		},
		{
			name:        "gzip encoding",
			key:         s.Prefix + "simple_gzip",
			expectedKey: s.Prefix + "simple_gzip.tar.gz",
			encoding:    s3helper.EncodingGzip,
		},
		{
			name:        "gzip encoding with extension",
			key:         s.Prefix + "simple_gzip.tar.gz",
			expectedKey: s.Prefix + "simple_gzip.tar.gz",
			encoding:    s3helper.EncodingGzip,
		},
		{
			name:        "gzip encoding with nested path",
			key:         s.Prefix + "nested_gzip/with/nested/path",
			expectedKey: s.Prefix + "nested_gzip/with/nested/path.tar.gz",
			encoding:    s3helper.EncodingGzip,
		},
		{
			name:        "gzip encoding with naming pattern",
			key:         s.Prefix + "pattern_gzip/{jobID}/{executionID}",
			expectedKey: s.Prefix + "pattern_gzip/" + s.JobID + "/" + s.ExecutionID + ".tar.gz",
			encoding:    s3helper.EncodingGzip,
		},
		{
			name:        "explicit endpoint and region",
			key:         s.Prefix + "simple_gzip_endpoint_and_region.tar.gz",
			expectedKey: s.Prefix + "simple_gzip_endpoint_and_region.tar.gz",
			endpoint:    s.Endpoint,
			region:      s.Region,
		},
		{
			name:       "explicit wrong region",
			key:        s.Prefix + "simple_gzip_wrong_region.tar.gz",
			region:     "us-east-1",
			shouldFail: true,
		},
		{
			name:        "default encoding to gzip",
			key:         s.Prefix + "simple_gzip_default",
			expectedKey: s.Prefix + "simple_gzip_default.tar.gz",
		},
	} {
		s.Run(tc.name, func() {
			if skipMessage != "" {
				s.T().Skip(skipMessage)
			}
			params := s3helper.PublisherSpec{
				Bucket:   s.Bucket,
				Key:      tc.key,
				Encoding: tc.encoding,
			}
			if tc.region == "" && tc.endpoint == "" {
				params.Region = s.Region
			} else {
				params.Region = tc.region
				params.Endpoint = tc.endpoint
			}

			resultPath := s.PrepareResultsPath()
			execution := s.MockExecution(params)
			storageSpec, err := s.PublishResult(execution, resultPath)

			if err != nil {
				var ae smithy.APIError
				if errors.As(err, &ae) && ae.ErrorCode() == "AccessDenied" {
					skipMessage = "No access to S3 bucket " + s.Bucket
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
			s.Equal(s.Bucket, sourceSpec.Bucket)
			s.Equal(params.Region, sourceSpec.Region)
			s.Equal(params.Endpoint, sourceSpec.Endpoint)

			if tc.encoding != s3helper.EncodingPlain {
				s.NotEmptyf(sourceSpec.ChecksumSHA256, "ChecksumSHA256 should not be empty")
				s.NotEmptyf(sourceSpec.VersionID, "VersionID should not be empty")
			} else {
				s.Empty(sourceSpec.ChecksumSHA256, "ChecksumSHA256 should be empty")
				s.Empty(sourceSpec.VersionID, "VersionID should be empty")
			}

			fetchedResults := s.GetResult(execution, &storageSpec)
			if tc.encoding == s3helper.EncodingGzip || tc.encoding == "" {
				uncompressedResults, err := os.MkdirTemp(s.TempDir, "")
				s.Require().NoError(err)
				s.Require().NoError(gzip.Decompress(fetchedResults, uncompressedResults))
				fetchedResults = uncompressedResults
			}

			s3test.AssertEqualDirectories(s.T(), resultPath, fetchedResults)
		})
	}
}
