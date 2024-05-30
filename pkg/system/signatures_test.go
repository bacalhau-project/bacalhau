//go:build unit || !integration

package system_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/setup"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

func TestMessageSigning(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("unexpected panic: %v", r)
		}
	}()
	_, cfg := setup.SetupBacalhauRepoForTesting(t)
	sk, err := config.GetClientPrivateKey(cfg.User.KeyPath)
	require.NoError(t, err)
	signer := system.NewMessageSigner(sk)

	msg := []byte("Hello, world!")
	sig, err := signer.Sign(msg)
	require.NoError(t, err)

	publicKey := signer.PublicKeyString()
	err = system.Verify(msg, sig, publicKey)
	require.NoError(t, err)
}

// TODO(forrest) [techdebt] the below cases are commented out as they have nothing to do with signatures
// they are asserting that when a repo is created a private key is generated. Test like this should go in the repo
// package once we allow the repo to take responsibility for creating and storing all client data.
/*
func TestGetClientID(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("unexpected panic: %v", r)
		}
	}()

	var err error
	var firstId string
	var secondId string
	t.Run("first", func(t *testing.T) {
		setup.SetupBacalhauRepoForTesting(t)
		firstId, err = config.GetClientID()
		require.NoError(t, err)
		require.NotEmpty(t, firstId)
	})

	t.Run("second", func(t *testing.T) {
		setup.SetupBacalhauRepoForTesting(t)
		secondId, err = config.GetClientID()
		require.NoError(t, err)
		require.NotEmpty(t, secondId)

		// Two different clients should have different IDs.
		assert.NotEqual(t, firstId, secondId)
	})
}

func TestPublicKeyMatchesID(t *testing.T) {
	setup.SetupBacalhauRepoForTesting(t)
	id, err := config.GetClientID()
	require.NoError(t, err)
	publicKey, err := config.GetClientPublicKeyString()
	require.NoError(t, err)
	ok, err := system.PublicKeyMatchesID(publicKey, id)
	require.NoError(t, err)
	require.True(t, ok)
}

*/
