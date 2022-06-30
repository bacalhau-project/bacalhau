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
