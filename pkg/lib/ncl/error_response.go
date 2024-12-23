package ncl

import (
	"fmt"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/lib/envelope"
)

const (
	BacErrorMessageType = "BacError"

	// KeyStatusCode is the key for the status code
	KeyStatusCode = "Bacalhau-StatusCode"

	// KeyErrorCode is the key for the error code
	KeyErrorCode = "Bacalhau-ErrorCode"
)

// BacErrorToEnvelope converts the error to an envelope
func BacErrorToEnvelope(err bacerrors.Error) *envelope.Message {
	errMsg := envelope.NewMessage(err)
	errMsg.WithMetadataValue(envelope.KeyMessageType, BacErrorMessageType)
	errMsg.WithMetadataValue(KeyStatusCode, fmt.Sprintf("%d", err.HTTPStatusCode()))
	errMsg.WithMetadataValue(KeyErrorCode, string(err.Code()))
	errMsg.WithMetadataValue(KeyEventTime, time.Now().Format(time.RFC3339))
	return errMsg
}
