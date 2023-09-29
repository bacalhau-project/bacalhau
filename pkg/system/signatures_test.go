//go:build unit || !integration

package system_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
	setup.SetupBacalhauRepoForTesting(t)

	msg := []byte("Hello, world!")
	sig, err := system.SignForClient(msg)
	require.NoError(t, err)

	ok, err := system.VerifyForClient(msg, sig)
	require.NoError(t, err)
	require.True(t, ok)

	publicKey := system.GetClientPublicKey()
	err = system.Verify(msg, sig, publicKey)
	require.NoError(t, err)
}

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
