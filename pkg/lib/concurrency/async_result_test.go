//go:build unit || !integration

package concurrency

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAsyncResultMarshalling(t *testing.T) {
	original := AsyncResult[string]{
		Value: "test",
		Err:   errors.New("sample error"),
	}

	// Test marshalling
	marshalled, err := json.Marshal(original)
	assert.NoError(t, err)

	// Test unmarshalling
	var unmarshalled AsyncResult[string]
	err = json.Unmarshal(marshalled, &unmarshalled)
	assert.NoError(t, err)

	// Verify that the unmarshalled object matches the original
	assert.Equal(t, original.Value, unmarshalled.Value)
	assert.Equal(t, original.Err.Error(), unmarshalled.Err.Error())
}

func TestAsyncResultMarshallingNoError(t *testing.T) {
	original := AsyncResult[string]{
		Value: "test",
	}

	// Test marshalling
	marshalled, err := json.Marshal(original)
	assert.NoError(t, err)

	// Test unmarshalling
	var unmarshalled AsyncResult[string]
	err = json.Unmarshal(marshalled, &unmarshalled)
	assert.NoError(t, err)

	// Verify that the unmarshalled object matches the original
	assert.Equal(t, original.Value, unmarshalled.Value)
	assert.Nil(t, unmarshalled.Err)
}
