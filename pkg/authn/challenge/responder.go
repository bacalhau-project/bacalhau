package challenge

import (
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/bacalhau-project/bacalhau/pkg/system"
)

type Responder struct {
	Signer system.Signer
}

func (c *Responder) Respond(input *json.RawMessage) ([]byte, error) {
	var req request
	err := json.Unmarshal(*input, &req)
	if err != nil {
		return nil, err
	}

	res, err := c.generateChallenge(req)
	if err != nil {
		return nil, err
	}

	return json.Marshal(res)
}

func (c *Responder) generateChallenge(req request) (response, error) {
	if req.InputPhrase == nil || len(req.InputPhrase) == 0 {
		return response{}, errors.New("unexpected challenge input")
	}

	signature, err := c.Signer.Sign(req.InputPhrase)
	if err != nil {
		return response{}, err
	}

	return response{
		PhraseSignature: signature,
		PublicKey:       c.Signer.PublicKeyString(),
	}, nil
}
