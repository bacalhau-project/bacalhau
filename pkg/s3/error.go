package s3

import "github.com/bacalhau-project/bacalhau/pkg/models"

const S3_PUBLISHER_COMPONENT = "S3PUB"

func NewErrBadS3Request(msg string) *models.BaseError {
	return models.NewBaseError(msg).WithCode(models.NewErrorCode(S3_PUBLISHER_COMPONENT, 400))
}
