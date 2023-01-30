package publicapi

import (
	"errors"
	"fmt"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
)

func verifyRequestSignature(req *submitRequest) error {
	// Check that the signature is valid:
	err := system.Verify(*req.JobCreatePayload, req.ClientSignature, req.ClientPublicKey)
	if err != nil {
		return fmt.Errorf("client's signature is invalid: %w", err)
	}

	return nil
}

func verifySubmitRequest(req *submitRequest, payload *model.JobCreatePayload) error {
	if payload.ClientID == "" {
		return errors.New("job create payload must contain a client ID")
	}
	if req.ClientSignature == "" {
		return errors.New("client's signature is required")
	}
	if req.ClientPublicKey == "" {
		return errors.New("client's public key is required")
	}

	// Check that the client's public key matches the client ID:
	ok, err := system.PublicKeyMatchesID(req.ClientPublicKey, payload.ClientID)
	if err != nil {
		return fmt.Errorf("error verifying client ID: %w", err)
	}
	if !ok {
		return errors.New("client's public key does not match client ID")
	}
	return nil
}
