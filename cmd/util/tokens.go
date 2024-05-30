package util

import (
	"encoding/json"
	"os"

	"github.com/pkg/errors"

	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/bacalhau-project/bacalhau/pkg/storage/util"
	"github.com/bacalhau-project/bacalhau/pkg/util/closer"
)

type tokens map[string]string

func readTokens(path string) (tokens, error) {
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return map[string]string{}, nil
	} else if err != nil {
		return nil, errors.Wrap(err, "error opening file")
	}
	defer closer.CloseWithLogOnError(path, file)

	// Treat an empty file as an empty map: #3569.
	if info, err := file.Stat(); err == nil && info.Size() == 0 {
		return map[string]string{}, nil
	}

	var t tokens
	err = json.NewDecoder(file).Decode(&t)
	if err != nil {
		return nil, errors.Wrap(err, "error decoding JSON")
	}

	return t, nil
}

func writeTokens(path string, t tokens) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, util.OS_USER_RW)
	if err != nil {
		return err
	}
	defer closer.CloseWithLogOnError(path, file)

	return json.NewEncoder(file).Encode(t)
}

// Read the authorization crdential associated with the passed API base URL. If
// there is no credential currently stored, ReadToken will return nil with no
// error.
func ReadToken(path string, apiURL string) (*apimodels.HTTPCredential, error) {
	t, err := readTokens(path)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read tokens file")
	}

	cred := &apimodels.HTTPCredential{
		Scheme: "Bearer",
		Value:  t[apiURL],
	}

	if cred.Value != "" {
		return cred, nil
	} else {
		return nil, nil
	}
}

// Persistently store the authorization token against the passed API base URL.
// Callers may pass nil for the credential which will delete any existing stored
// token.
func WriteToken(path, apiURL string, cred *apimodels.HTTPCredential) error {
	t, err := readTokens(path)
	if err != nil {
		return err
	}

	if cred != nil {
		t[apiURL] = cred.Value
	} else {
		delete(t, apiURL)
	}

	return writeTokens(path, t)
}
