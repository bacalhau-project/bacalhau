package signatures

import (
	"context"
	"encoding/json"
	"io"

	"github.com/pkg/errors"

	"github.com/bacalhau-project/bacalhau/pkg/lib/marshaller"
	"github.com/bacalhau-project/bacalhau/pkg/system"
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

func SignRequest(s system.Signer, reqData any) (req signedRequest, err error) {
	jsonData, err := marshaller.JSONMarshalWithMax(reqData)
	if err != nil {
		return
	}
	rawJSON := json.RawMessage(jsonData)

	signature, err := s.Sign(rawJSON)
	if err != nil {
		return
	}

	req = signedRequest{
		Payload:         &rawJSON,
		ClientSignature: signature,
		ClientPublicKey: s.PublicKeyString(),
	}
	return
}

func UnmarshalSigned[PayloadType ContainsClientID](ctx context.Context, body io.Reader) (PayloadType, error) {
	var request signedRequest
	var payload PayloadType

	if err := json.NewDecoder(body).Decode(&request); err != nil {
		return payload, errors.Wrap(err, "error unmarshalling envelope")
	}

	if request.Payload == nil {
		return payload, errors.New("no payload contained in signed message")
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
	if err := verifySignedRequest(payload.GetClientID(), request.ClientSignature, request.ClientPublicKey); err != nil {
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

func verifySignedRequest(reqClientID string, clientSig string, clientPubKey string) error {
	if reqClientID == "" {
		return errors.New("payload must contain a client ID")
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
