package boltjobstore

import (
	"errors"
	"net/http"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"go.etcd.io/bbolt"
)

const BoltDBComponent = "BoltDB"

const (
	BoltDBBucketNotFound    models.ErrorCode = "BoltDBBucketNotFound"
	BoltDBBucketExists      models.ErrorCode = "BoltDBBucketExists"
	BoltDBTxNotWritable     models.ErrorCode = "BoltDBTxNotWritable"
	BoltDBIncompatibleValue models.ErrorCode = "BoltDBIncompatibleValue"
	BoltDBKeyRequired       models.ErrorCode = "BoltDBKeyRequired"
	BoltDBKeyTooLarge       models.ErrorCode = "BoltDBKeyTooLarge"
	BoltDBValueTooLarge     models.ErrorCode = "BoltDBValueTooLarge"
)

func NewBoltDbError(err error) *models.BaseError {
	switch {
	case errors.Is(err, bbolt.ErrBucketNotFound):
		return models.NewBaseError(err.Error()).
			WithCode(BoltDBBucketNotFound).
			WithHTTPStatusCode(http.StatusNotFound).
			WithComponent(BoltDBComponent)
	case errors.Is(err, bbolt.ErrBucketExists):
		return models.NewBaseError(err.Error()).
			WithCode(BoltDBBucketExists).
			WithHTTPStatusCode(http.StatusConflict).
			WithComponent(BoltDBComponent)
	case errors.Is(err, bbolt.ErrTxNotWritable):
		return models.NewBaseError(err.Error()).
			WithCode(BoltDBTxNotWritable).
			WithHTTPStatusCode(http.StatusInternalServerError).
			WithComponent(BoltDBComponent)
	case errors.Is(err, bbolt.ErrIncompatibleValue):
		return models.NewBaseError(err.Error()).
			WithCode(BoltDBIncompatibleValue).
			WithComponent(BoltDBComponent)
	case errors.Is(err, bbolt.ErrKeyRequired):
		return models.NewBaseError(err.Error()).
			WithCode(BoltDBKeyRequired).
			WithComponent(BoltDBComponent)
	case errors.Is(err, bbolt.ErrKeyTooLarge):
		return models.NewBaseError(err.Error()).
			WithCode(BoltDBKeyTooLarge).
			WithComponent(BoltDBComponent)
	case errors.Is(err, bbolt.ErrValueTooLarge):
		return models.NewBaseError(err.Error()).
			WithCode(BoltDBValueTooLarge).
			WithComponent(BoltDBComponent)
	default:
		return models.NewBaseError(err.Error()).
			WithCode(models.InternalError).
			WithComponent(BoltDBComponent)
	}
}
