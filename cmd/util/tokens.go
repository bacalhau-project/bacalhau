package util

import (
	"encoding/json"
	"os"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/storage/util"
	"github.com/bacalhau-project/bacalhau/pkg/util/closer"
)

type tokens map[string]string

func readTokens(path string) (tokens, error) {
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return map[string]string{}, nil
	} else if err != nil {
		return nil, err
	}
	defer closer.CloseWithLogOnError(path, file)

	var t tokens
	err = json.NewDecoder(file).Decode(&t)
	if err != nil {
		return nil, err
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

func ReadToken(apiURL string) (string, error) {
	path, err := config.Get[string](types.AuthTokensPath)
	if err != nil {
		return "", err
	}

	t, err := readTokens(path)
	if err != nil {
		return "", err
	}
	return t[apiURL], nil
}

func WriteToken(apiURL, token string) error {
	path, err := config.Get[string](types.AuthTokensPath)
	if err != nil {
		return err
	}

	t, err := readTokens(path)
	if err != nil {
		return err
	}

	t[apiURL] = token

	return writeTokens(path, t)
}
