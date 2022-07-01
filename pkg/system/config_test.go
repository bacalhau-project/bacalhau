package system

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMessageSigning(t *testing.T) {
	InitConfigForTesting(t)

	msg := []byte("Hello, world!")
	sig, err := SignForUser(msg)
	assert.NoError(t, err)

	ok, err := VerifyForUser(msg, sig)
	assert.NoError(t, err)
	assert.True(t, ok)
}

func TestGetClientID(t *testing.T) {
	InitConfigForTesting(t)
	id, err := GetClientID()
	assert.NoError(t, err)
	assert.NotEmpty(t, id)

	InitConfigForTesting(t)
	id2, err := GetClientID()
	assert.NoError(t, err)
	assert.NotEmpty(t, id2)

	// Two different clients should have different IDs.
	assert.NotEqual(t, id, id2)
}
