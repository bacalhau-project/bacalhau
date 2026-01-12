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
	case errors.Is(err, bbolt.ErrBucketNotFound): //nolint:staticcheck // TODO: migrate to bbolt/errors package
		return bacerrors.Newf("%s", err).
			WithCode(BoltDBBucketNotFound).
			WithHTTPStatusCode(http.StatusNotFound).
			WithComponent(BoltDBComponent)
	case errors.Is(err, bbolt.ErrBucketExists): //nolint:staticcheck // TODO: migrate to bbolt/errors package
		return bacerrors.Newf("%s", err).
			WithCode(BoltDBBucketExists).
			WithHTTPStatusCode(http.StatusConflict).
			WithComponent(BoltDBComponent)
	case errors.Is(err, bbolt.ErrTxNotWritable): //nolint:staticcheck // TODO: migrate to bbolt/errors package
		return bacerrors.Newf("%s", err).
			WithCode(BoltDBTxNotWritable).
			WithHTTPStatusCode(http.StatusInternalServerError).
			WithComponent(BoltDBComponent)
	case errors.Is(err, bbolt.ErrIncompatibleValue): //nolint:staticcheck // TODO: migrate to bbolt/errors package
		return bacerrors.Newf("%s", err).
			WithCode(BoltDBIncompatibleValue).
			WithComponent(BoltDBComponent)
	case errors.Is(err, bbolt.ErrKeyRequired): //nolint:staticcheck // TODO: migrate to bbolt/errors package
		return bacerrors.Newf("%s", err).
			WithCode(BoltDBKeyRequired).
			WithComponent(BoltDBComponent)
	case errors.Is(err, bbolt.ErrKeyTooLarge): //nolint:staticcheck // TODO: migrate to bbolt/errors package
		return bacerrors.Newf("%s", err).
			WithCode(BoltDBKeyTooLarge).
			WithComponent(BoltDBComponent)
	case errors.Is(err, bbolt.ErrValueTooLarge): //nolint:staticcheck // TODO: migrate to bbolt/errors package
		return bacerrors.Newf("%s", err).
			WithCode(BoltDBValueTooLarge).
			WithComponent(BoltDBComponent)
	default:
		return bacerrors.Newf("%s", err).
			WithCode(bacerrors.BadRequestError).
			WithComponent(BoltDBComponent)
	}
}
