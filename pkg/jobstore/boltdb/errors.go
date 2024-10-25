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

func NewBoltDBError(err error) bacerrors.Error {
	switch {
	case errors.Is(err, bbolt.ErrBucketNotFound):
		return bacerrors.New("%s", err).
			WithCode(BoltDBBucketNotFound).
			WithHTTPStatusCode(http.StatusNotFound).
			WithComponent(BoltDBComponent)
	case errors.Is(err, bbolt.ErrBucketExists):
		return bacerrors.New("%s", err).
			WithCode(BoltDBBucketExists).
			WithHTTPStatusCode(http.StatusConflict).
			WithComponent(BoltDBComponent)
	case errors.Is(err, bbolt.ErrTxNotWritable):
		return bacerrors.New("%s", err).
			WithCode(BoltDBTxNotWritable).
			WithHTTPStatusCode(http.StatusInternalServerError).
			WithComponent(BoltDBComponent)
	case errors.Is(err, bbolt.ErrIncompatibleValue):
		return bacerrors.New("%s", err).
			WithCode(BoltDBIncompatibleValue).
			WithComponent(BoltDBComponent)
	case errors.Is(err, bbolt.ErrKeyRequired):
		return bacerrors.New("%s", err).
			WithCode(BoltDBKeyRequired).
			WithComponent(BoltDBComponent)
	case errors.Is(err, bbolt.ErrKeyTooLarge):
		return bacerrors.New("%s", err).
			WithCode(BoltDBKeyTooLarge).
			WithComponent(BoltDBComponent)
	case errors.Is(err, bbolt.ErrValueTooLarge):
		return bacerrors.New("%s", err).
			WithCode(BoltDBValueTooLarge).
			WithComponent(BoltDBComponent)
	default:
		return bacerrors.New("%s", err).
			WithCode(bacerrors.BadRequestError).
			WithComponent(BoltDBComponent)
	}
}
