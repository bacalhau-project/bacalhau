package s3

import (
	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
)

const S3_PUBLISHER = "S3Publisher"
const S3_INPUT_SPEC = "S3InputSpec"
const S3_DOWNLOADER = "S3Downloader"

const (
	S3BadRequest = "S3BadRequest"
)

func NewS3PublisherError(code bacerrors.ErrorCode, message string) bacerrors.Error {
	return bacerrors.New(message).
		WithCode(code).
		WithComponent(S3_PUBLISHER)
}

func NewS3InputSpecError(code bacerrors.ErrorCode, message string) bacerrors.Error {
	return bacerrors.New(message).
		WithCode(code).
		WithComponent(S3_INPUT_SPEC)
}

func NewS3DownloaderError(code bacerrors.ErrorCode, message string) bacerrors.Error {
	return bacerrors.New(message).
		WithCode(code).
		WithComponent(S3_DOWNLOADER)
}
