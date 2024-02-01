package challenge

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/json"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/pkg/errors"
)

func Respond(input *json.RawMessage) ([]byte, error) {
	var req request
	err := json.Unmarshal(*input, &req)
	if err != nil {
		return nil, err
	}

	res, err := generateChallenge(req)
	if err != nil {
		return nil, err
	}

	return json.Marshal(res)
}

func generateChallenge(req request) (response, error) {
	if req.InputPhrase == nil || len(req.InputPhrase) == 0 {
		return response{}, errors.New("unexpected challenge input")
	}

	userPrivKey, err := config.GetClientPrivateKey()
	if err != nil {
		return response{}, err
	}

	userPubKey := base64.StdEncoding.EncodeToString(x509.MarshalPKCS1PublicKey(&userPrivKey.PublicKey))

	signature, err := system.Sign(req.InputPhrase, userPrivKey)
	if err != nil {
		return response{}, err
	}

	return response{
		PhraseSignature: signature,
		PublicKey:       userPubKey,
	}, nil
}
