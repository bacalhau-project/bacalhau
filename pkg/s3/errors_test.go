//go:build unit || !integration

/* spell-checker: disable */

package s3

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/smithy-go"
	smithyhttp "github.com/aws/smithy-go/transport/http"
	pkgerrors "github.com/pkg/errors"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type S3ErrorTestSuite struct {
	suite.Suite
}

func TestS3ErrorTestSuite(t *testing.T) {
	suite.Run(t, new(S3ErrorTestSuite))
}

func (suite *S3ErrorTestSuite) TestExtractErrorMetadata() {
	tests := []struct {
		name     string
		err      error
		expected errorMetadata
	}{
		{
			name: "operation error with HostID",
			err: pkgerrors.Wrap(&smithy.OperationError{
				ServiceID:     "S3",
				OperationName: "ListObjectsV2",
				Err: &awshttp.ResponseError{
					ResponseError: &smithyhttp.ResponseError{
						Response: &smithyhttp.Response{
							Response: &http.Response{
								StatusCode: 404,
							},
						},
						Err: fmt.Errorf("HostID: lSs3QvEo/W6rdJsL4Y6iw2t8upy5V0uRzByCgjRQ 84yRBkbcXPU2iC3HVclnx1v811K0h2a9WA=, api error NoSuchBucket: The specified bucket does not exist"),
					},
					RequestID: "YRSWGFERH6VN7DK0",
				},
			}, "failed to publish s3 results"),
			expected: errorMetadata{
				service:    "S3",
				operation:  "ListObjectsV2",
				statusCode: 404,
				requestID:  "YRSWGFERH6VN7DK0",
				errorCode:  bacerrors.UnknownError,
				message:    "failed to publish s3 results: operation error S3: ListObjectsV2, https response error StatusCode: 404, RequestID: YRSWGFERH6VN7DK0, api error NoSuchBucket: The specified bucket does not exist",
			},
		},
		{
			name: "Simple error",
			err:  errors.New("simple error"),
			expected: errorMetadata{
				service:   "S3",
				errorCode: bacerrors.UnknownError,
				message:   "simple error",
			},
		},
		{
			name: "Error with HostID",
			err:  errors.New("operation error S3: GetObject, https response error StatusCode: 403, RequestID: ABC123, HostID: XYZ789, ForbiddenError: Access Denied"),
			expected: errorMetadata{
				service:   "S3",
				errorCode: bacerrors.UnknownError,
				message:   "operation error S3: GetObject, https response error StatusCode: 403, RequestID: ABC123, ForbiddenError: Access Denied",
			},
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			result := extractErrorMetadata(tt.err)
			suite.Equal(tt.expected, result)
		})
	}
}

func (suite *S3ErrorTestSuite) TestNewS3PublisherError() {
	err := NewS3PublisherError(bacerrors.BadRequestError, "test message")
	suite.Equal(bacerrors.BadRequestError, err.Code())
	suite.Equal(PublisherComponent, err.Component())
	suite.Equal("test message", err.Error())
}

func (suite *S3ErrorTestSuite) TestNewS3InputSpecError() {
	err := NewS3InputSourceError(bacerrors.BadRequestError, "test message")
	suite.Equal(bacerrors.BadRequestError, err.Code())
	suite.Equal(InputSourceComponent, err.Component())
	suite.Equal("test message", err.Error())
}

func (suite *S3ErrorTestSuite) TestNewS3DownloaderError() {
	err := NewS3DownloaderError(bacerrors.BadRequestError, "test message")
	suite.Equal(bacerrors.BadRequestError, err.Code())
	suite.Equal(DownloadComponent, err.Component())
	suite.Equal("test message", err.Error())
}

func (suite *S3ErrorTestSuite) TestNewS3PublisherServiceError() {
	originalErr := errors.New("test error")
	err := NewS3PublisherServiceError(originalErr)
	suite.Equal(PublisherComponent, err.Component())
	suite.Contains(err.Error(), "failed to publish s3 result")
	suite.Contains(err.Error(), "test error")
}

func (suite *S3ErrorTestSuite) TestNewS3InputSourceServiceError() {
	originalErr := errors.New("test error")
	err := NewS3InputSourceServiceError(originalErr)
	suite.Equal(InputSourceComponent, err.Component())
	suite.Contains(err.Error(), "failed to fetch s3 input")
	suite.Contains(err.Error(), "test error")
}

func (suite *S3ErrorTestSuite) TestNewS3ResultSignerServiceError() {
	originalErr := errors.New("test error")
	err := NewS3ResultSignerServiceError(originalErr)
	suite.Equal(ResultSignerComponent, err.Component())
	suite.Contains(err.Error(), "failed to fetch s3 result")
	suite.Contains(err.Error(), "test error")
}

func (suite *S3ErrorTestSuite) TestErrorMetadataToDetails() {
	metadata := errorMetadata{
		service:    "S3",
		errorCode:  bacerrors.BadRequestError,
		statusCode: 400,
		requestID:  "TEST123",
		operation:  "GetObject",
		message:    "Bad Request",
	}

	details := metadata.toDetails()

	suite.Equal("S3", details["Service"])
	suite.Equal(string(bacerrors.BadRequestError), details[models.DetailsKeyErrorCode])
	suite.Equal("TEST123", details["AWSRequestID"])
	suite.Equal("GetObject", details["Operation"])
}

func (suite *S3ErrorTestSuite) TestNewS3ServiceError() {
	originalErr := &smithy.OperationError{
		ServiceID:     "S3",
		OperationName: "GetObject",
		Err: &awshttp.ResponseError{
			ResponseError: &smithyhttp.ResponseError{
				Response: &smithyhttp.Response{
					Response: &http.Response{
						StatusCode: 403,
					},
				},
				Err: fmt.Errorf("ForbiddenError: Access Denied"),
			},
			RequestID: "TEST123",
		},
	}

	err := newS3ServiceError(originalErr, PublisherComponent)

	suite.Equal(PublisherComponent, err.Component())
	suite.Equal(403, err.HTTPStatusCode())
	suite.Contains(err.Error(), "ForbiddenError: Access Denied")

	details := err.Details()
	suite.Equal("S3", details["Service"])
	suite.Equal("TEST123", details["AWSRequestID"])
	suite.Equal("GetObject", details["Operation"])
}
