package s3

import "github.com/bacalhau-project/bacalhau/pkg/models"

const S3 = "S3"

const (
	S3BadRequest = "S3BadRequest"
)

func NewBadS3RequestError(message string) *models.BaseError {
	return models.NewBaseError(message).
		WithCode(S3BadRequest).
		WithComponent(S3)
}
