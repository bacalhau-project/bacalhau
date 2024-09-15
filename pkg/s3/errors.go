package s3

import "github.com/bacalhau-project/bacalhau/pkg/models"

const S3_PUBLISHER = "S3Publisher"
const S3_STORAGE = "S3Storage"

const (
	S3BadRequest = "S3BadRequest"
)

func NewS3PublisherError(code models.ErrorCode, message string) *models.BaseError {
	return models.NewBaseError(message).
		WithCode(code).
		WithComponent(S3_PUBLISHER)
}

func NewS3StorageError(code models.ErrorCode, message string) *models.BaseError {
	return models.NewBaseError(message).
		WithCode(code).
		WithComponent(S3_STORAGE)
}
