package boltjobstore

import (
	"errors"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"go.etcd.io/bbolt"
)

const BOLTDB_COMPONENT = "BDB"

func NewBoltDbError(err error) *models.BaseError {
	switch {
	case errors.Is(err, bbolt.ErrBucketNotFound):
		return models.NewBaseError(err.Error()).WithCode(models.NewErrorCode(BOLTDB_COMPONENT, 404))
	case errors.Is(err, bbolt.ErrBucketExists):
		return models.NewBaseError(err.Error()).WithCode(models.NewErrorCode(BOLTDB_COMPONENT, 409))
	case errors.Is(err, bbolt.ErrTxNotWritable):
		return models.NewBaseError(err.Error()).WithCode(models.NewErrorCode(BOLTDB_COMPONENT, 500))
	case errors.Is(err, bbolt.ErrIncompatibleValue):
		return models.NewBaseError(err.Error()).WithCode(models.NewErrorCode(BOLTDB_COMPONENT, 500))
	case errors.Is(err, bbolt.ErrKeyRequired):
		return models.NewBaseError(err.Error()).WithCode(models.NewErrorCode(BOLTDB_COMPONENT, 500))
	case errors.Is(err, bbolt.ErrKeyTooLarge):
		return models.NewBaseError(err.Error()).WithCode(models.NewErrorCode(BOLTDB_COMPONENT, 500))
	case errors.Is(err, bbolt.ErrValueTooLarge):
		return models.NewBaseError(err.Error()).WithCode(models.NewErrorCode(BOLTDB_COMPONENT, 500))
	default:
		return models.NewBaseError(err.Error()).WithCode(models.NewErrorCode(BOLTDB_COMPONENT, 500))
	}
}
