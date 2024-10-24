package s3

import (
	"errors"
	"regexp"
	"strings"

	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/smithy-go"
	pkgerrors "github.com/pkg/errors"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

const PublisherComponent = "S3Publisher"
const InputSourceComponent = "S3InputSource"
const DownloadComponent = "S3Downloader"
const ResultSignerComponent = "S3ResultSigner"

const (
	BadRequestErrorCode = "S3BadRequest"
)

func NewS3PublisherError(code bacerrors.ErrorCode, message string) bacerrors.Error {
	return bacerrors.New("%s", message).
		WithCode(code).
		WithComponent(PublisherComponent)
}

func NewS3InputSourceError(code bacerrors.ErrorCode, message string) bacerrors.Error {
	return bacerrors.New("%s", message).
		WithCode(code).
		WithComponent(InputSourceComponent)
}

func NewS3DownloaderError(code bacerrors.ErrorCode, message string) bacerrors.Error {
	return bacerrors.New("%s", message).
		WithCode(code).
		WithComponent(DownloadComponent)
}

func NewS3PublisherServiceError(err error) bacerrors.Error {
	return newS3ServiceError(pkgerrors.Wrap(err, "failed to publish s3 result"), PublisherComponent)
}

func NewS3InputSourceServiceError(err error) bacerrors.Error {
	return newS3ServiceError(pkgerrors.Wrap(err, "failed to fetch s3 input"), InputSourceComponent)
}

func NewS3ResultSignerServiceError(err error) bacerrors.Error {
	return newS3ServiceError(pkgerrors.Wrap(err, "failed to fetch s3 result"), ResultSignerComponent)
}

func newS3ServiceError(err error, component string) bacerrors.Error {
	errMetadata := extractErrorMetadata(err)
	return bacerrors.New("%s", errMetadata.message).
		WithComponent(component).
		WithCode(errMetadata.errorCode).
		WithHTTPStatusCode(errMetadata.statusCode).
		WithDetails(errMetadata.toDetails())
}

type errorMetadata struct {
	service    string
	errorCode  bacerrors.ErrorCode
	statusCode int
	requestID  string
	operation  string
	message    string
}

// toDetails converts the error metadata to a map of details.
func (m errorMetadata) toDetails() map[string]string {
	details := map[string]string{
		models.DetailsKeyErrorCode: string(m.errorCode),
		"Service":                  m.service,
	}
	if m.requestID != "" {
		details["AWSRequestID"] = m.requestID
	}
	if m.operation != "" {
		details["Operation"] = m.operation
	}
	return details
}

// extractErrorMetadata extracts the error code and message from the error.
// It trie
func extractErrorMetadata(err error) errorMetadata {
	metadata := errorMetadata{
		service:   "S3",
		errorCode: bacerrors.UnknownError,
		message:   err.Error(),
	}

	// Parse the error message and remove the HostID if present as it is noisy.
	errMsg := err.Error()
	// Regular expression to match and remove the HostID
	re := regexp.MustCompile(`, HostID: [^,]+`)
	cleanedErrMsg := re.ReplaceAllString(errMsg, "")
	// Remove any double commas that might result from removing HostID
	cleanedErrMsg = strings.ReplaceAll(cleanedErrMsg, ",,", ",")
	// Trim any leading or trailing whitespace and commas
	cleanedErrMsg = strings.Trim(cleanedErrMsg, " ,")
	metadata.message = cleanedErrMsg

	var opErr *smithy.OperationError
	if errors.As(err, &opErr) {
		metadata.operation = opErr.Operation()
		metadata.service = opErr.Service()

		var respError *awshttp.ResponseError
		if errors.As(opErr.Err, &respError) {
			metadata.statusCode = respError.HTTPStatusCode()
			metadata.requestID = respError.ServiceRequestID()
		}

		var apiErr smithy.APIError
		if errors.As(opErr.Err, &apiErr) {
			metadata.errorCode = bacerrors.Code(apiErr.ErrorCode())
		}
	}

	return metadata
}
