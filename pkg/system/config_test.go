package system

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMessageSigning(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("unexpected panic: %v", r)
		}
	}()

	InitConfigForTesting(t)

	msg := []byte("Hello, world!")
	sig, err := SignForClient(msg)
	require.NoError(t, err)

	ok, err := VerifyForClient(msg, sig)
	require.NoError(t, err)
	assert.True(t, ok)

	publicKey := GetClientPublicKey()
	ok, err = Verify(msg, sig, publicKey)
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestGetClientID(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("unexpected panic: %v", r)
		}
	}()

	InitConfigForTesting(t)
	id := GetClientID()
	require.NotEmpty(t, id)

	InitConfigForTesting(t)
	id2 := GetClientID()
	require.NotEmpty(t, id2)

	// Two different clients should have different IDs.
	require.NotEqual(t, id, id2)
}

func TestPublicKeyMatchesID(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("unexpected panic: %v", r)
		}
	}()

	InitConfigForTesting(t)

	id := GetClientID()
	publicKey := GetClientPublicKey()
	ok, err := PublicKeyMatchesID(publicKey, id)
	require.NoError(t, err)
	assert.True(t, ok)
}
