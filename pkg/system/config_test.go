package system

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMessageSigning(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("unexpected panic: %v", r)
		}
	}()

	InitConfigForTesting(t)

	msg := []byte("Hello, world!")
	sig, err := SignForUser(msg)
	assert.NoError(t, err)
	assert.True(t, VerifyForUser(msg, sig))
}

func TestGetClientID(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("unexpected panic: %v", r)
		}
	}()

	InitConfigForTesting(t)
	id := GetClientID()
	assert.NotEmpty(t, id)

	InitConfigForTesting(t)
	id2 := GetClientID()
	assert.NotEmpty(t, id2)

	// Two different clients should have different IDs.
	assert.NotEqual(t, id, id2)
}
