package storage

import "github.com/bacalhau-project/bacalhau/pkg/models"

const S3_STORAGE_COMPONENT = "S3STOR"

func NewErrBadS3StorageRequest(msg string) *models.BaseError {
	return models.NewBaseError(msg).WithCode(models.NewErrorCode(S3_STORAGE_COMPONENT, 400))
}
