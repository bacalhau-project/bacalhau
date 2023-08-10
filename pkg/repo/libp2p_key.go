package repo

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/storage/util"
)

const BitsForKeyPair = 2048

func (fsr *FsRepo) InitLibp2pPrivateKey(keyPort int) (crypto.PrivKey, error) {
	if exists, err := fsr.Exists(); err != nil {
		return nil, fmt.Errorf("failed to check if repo exists: %w", err)
	} else if !exists {
		return nil, fmt.Errorf("repo is uninitialized, cannot init libp2p private key")
	}

	keyName := fmt.Sprintf("private_key.%d", keyPort)

	// We include the port in the filename so that in devstack multiple nodes
	// running on the same host get different identities
	privKeyPath := filepath.Join(fsr.path, keyName)

	if _, err := os.Stat(privKeyPath); errors.Is(err, os.ErrNotExist) {
		// Private key does not exist - create and write it

		// Creates a new RSA key pair for this host.
		prvKey, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, BitsForKeyPair, rand.Reader)
		if err != nil {
			log.Error().Err(err)
			return nil, err
		}

		keyOut, err := os.OpenFile(privKeyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, util.OS_USER_RW)
		if err != nil {
			return nil, fmt.Errorf("failed to open key.pem for writing: %v", err)
		}
		privBytes, err := crypto.MarshalPrivateKey(prvKey)
		if err != nil {
			return nil, fmt.Errorf("unable to marshal private key: %v", err)
		}
		// base64 encode privBytes
		b64 := base64.StdEncoding.EncodeToString(privBytes)
		_, err = keyOut.WriteString(b64 + "\n")
		if err != nil {
			return nil, fmt.Errorf("failed to write to key file: %v", err)
		}
		if err := keyOut.Close(); err != nil {
			return nil, fmt.Errorf("error closing key file: %v", err)
		}
		log.Debug().Msgf("wrote %s", privKeyPath)
	} else {
		return nil, err
	}

	// Now that we've ensured the private key is written to disk, read it! This
	// ensures that loading it works even in the case where we've just created
	// it.

	// read the private key
	keyBytes, err := os.ReadFile(privKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %v", err)
	}
	// base64 decode keyBytes
	b64, err := base64.StdEncoding.DecodeString(string(keyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to decode private key: %v", err)
	}
	// parse the private key
	prvKey, err := crypto.UnmarshalPrivateKey(b64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %v", err)
	}

	return prvKey, nil
}
