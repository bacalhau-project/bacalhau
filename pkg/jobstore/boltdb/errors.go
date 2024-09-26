package boltjobstore

import (
	"errors"
	"net/http"

	"go.etcd.io/bbolt"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
)

const BoltDBComponent = "BoltDB"

const (
	BoltDBBucketNotFound    bacerrors.ErrorCode = "BoltDBBucketNotFound"
	BoltDBBucketExists      bacerrors.ErrorCode = "BoltDBBucketExists"
	BoltDBTxNotWritable     bacerrors.ErrorCode = "BoltDBTxNotWritable"
	BoltDBIncompatibleValue bacerrors.ErrorCode = "BoltDBIncompatibleValue"
	BoltDBKeyRequired       bacerrors.ErrorCode = "BoltDBKeyRequired"
	BoltDBKeyTooLarge       bacerrors.ErrorCode = "BoltDBKeyTooLarge"
	BoltDBValueTooLarge     bacerrors.ErrorCode = "BoltDBValueTooLarge"
)

func NewBoltDbError(err error) bacerrors.Error {
	switch {
	case errors.Is(err, bbolt.ErrBucketNotFound):
		return bacerrors.New(err.Error()).
			WithCode(BoltDBBucketNotFound).
			WithHTTPStatusCode(http.StatusNotFound).
			WithComponent(BoltDBComponent)
	case errors.Is(err, bbolt.ErrBucketExists):
		return bacerrors.New(err.Error()).
			WithCode(BoltDBBucketExists).
			WithHTTPStatusCode(http.StatusConflict).
			WithComponent(BoltDBComponent)
	case errors.Is(err, bbolt.ErrTxNotWritable):
		return bacerrors.New(err.Error()).
			WithCode(BoltDBTxNotWritable).
			WithHTTPStatusCode(http.StatusInternalServerError).
			WithComponent(BoltDBComponent)
	case errors.Is(err, bbolt.ErrIncompatibleValue):
		return bacerrors.New(err.Error()).
			WithCode(BoltDBIncompatibleValue).
			WithComponent(BoltDBComponent)
	case errors.Is(err, bbolt.ErrKeyRequired):
		return bacerrors.New(err.Error()).
			WithCode(BoltDBKeyRequired).
			WithComponent(BoltDBComponent)
	case errors.Is(err, bbolt.ErrKeyTooLarge):
		return bacerrors.New(err.Error()).
			WithCode(BoltDBKeyTooLarge).
			WithComponent(BoltDBComponent)
	case errors.Is(err, bbolt.ErrValueTooLarge):
		return bacerrors.New(err.Error()).
			WithCode(BoltDBValueTooLarge).
			WithComponent(BoltDBComponent)
	default:
		return bacerrors.New(err.Error()).
			WithCode(bacerrors.BadRequestError).
			WithComponent(BoltDBComponent)
	}
}
