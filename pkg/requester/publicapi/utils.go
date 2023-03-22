package publicapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// A weakly-typed signed request. We use this type only in our code
// on both client and server to correctly validate signatures.
type signedRequest struct {
	// The data needed to cancel a running job on the network
	Payload *json.RawMessage `json:"payload" validate:"required"`

	// A base64-encoded signature of the data, signed by the client:
	ClientSignature string `json:"signature" validate:"required"`

	// The base64-encoded public key of the client:
	ClientPublicKey string `json:"client_public_key" validate:"required"`
}

// A strongly-typed signed request. We use this type only in our documentation
// to allow clients to understand the correct type of the payload.
type SignedRequest[PayloadType ContainsClientID] struct {
	// The data needed to cancel a running job on the network
	Payload PayloadType `json:"payload" validate:"required"`

	// A base64-encoded signature of the data, signed by the client:
	ClientSignature string `json:"signature" validate:"required"`

	// The base64-encoded public key of the client:
	ClientPublicKey string `json:"client_public_key" validate:"required"`
}

type ContainsClientID interface {
	GetClientID() string
}

func unmarshalSignedJob[PayloadType ContainsClientID](ctx context.Context, body io.Reader) (PayloadType, error) {
	var request signedRequest
	var payload PayloadType

	if err := json.NewDecoder(body).Decode(&request); err != nil {
		return payload, errors.Wrap(err, "error unmarshalling envelope")
	}

	// first verify the signature on the raw bytes
	if err := verifyRequestSignature(*request.Payload, request.ClientSignature, request.ClientPublicKey); err != nil {
		return payload, errors.Wrap(err, "error verifying request signature")
	}

	// then decode the job create payload
	if err := json.Unmarshal(*request.Payload, &payload); err != nil {
		return payload, errors.Wrap(err, "error unmarshalling payload")
	}

	// check that the client id in the payload actually matches the key
	if err := verifySignedJobRequest(payload.GetClientID(), request.ClientSignature, request.ClientPublicKey); err != nil {
		return payload, errors.Wrap(err, "error validating request")
	}

	return payload, nil
}

func verifyRequestSignature(msg json.RawMessage, clientSignature string, clientPubKey string) error {
	err := system.Verify(msg, clientSignature, clientPubKey)
	if err != nil {
		return errors.Wrap(err, "client's signature is invalid")
	}

	return nil
}

func verifySignedJobRequest(reqClientID string, clientSig string, clientPubKey string) error {
	if reqClientID == "" {
		return errors.New("job create payload must contain a client ID")
	}
	if clientSig == "" {
		return errors.New("client's signature is required")
	}
	if clientPubKey == "" {
		return errors.New("client's public key is required")
	}

	// Check that the client's public key matches the client ID:
	ok, err := system.PublicKeyMatchesID(clientPubKey, reqClientID)
	if err != nil {
		return errors.Wrap(err, "error verifying client ID")
	}
	if !ok {
		return errors.New("client's public key does not match client ID")
	}
	return nil
}

func httpError(ctx context.Context, res http.ResponseWriter, err error, statusCode int) {
	log.Ctx(ctx).Error().Err(err).Send()
	http.Error(res, bacerrors.ErrorToErrorResponse(err), statusCode)
}
