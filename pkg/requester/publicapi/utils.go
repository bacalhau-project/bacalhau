package publicapi

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/filecoin-project/bacalhau/pkg/system"
)

func verifyRequestSignature(msg json.RawMessage, clientSignature string, clientPubKey string) error {
	err := system.Verify(msg, clientSignature, clientPubKey)
	if err != nil {
		return fmt.Errorf("client's signature is invalid: %w", err)
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
		return fmt.Errorf("error verifying client ID: %w", err)
	}
	if !ok {
		return errors.New("client's public key does not match client ID")
	}
	return nil
}
