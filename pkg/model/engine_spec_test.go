//go:build unit || !integration

package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testEngineSpec struct {
	Name   string
	Colors []string
}

func TestEngineBuilderAndDecode(t *testing.T) {
	expectedType := "TestEngineSpec"
	expectedName := "an_engine_name"
	expectedColors := []string{"red", "blue", "green"}

	builder := EngineBuilder{}
	spec := builder.WithType(expectedType).
		WithParam("Name", expectedName).
		WithParam("Colors", expectedColors).
		Build()

	assert.Equal(t, expectedType, spec.Type, "Expected Type to be '%s', got '%s'", expectedType, spec.Type)
	assert.Equal(t, expectedName, spec.Params["Name"], "Expected Name to be '%s', got '%s'", expectedName, spec.Params["Name"])
	assert.Equal(t, expectedColors, spec.Params["Colors"], "Expected Color to be '%s', got '%s'", expectedColors, spec.Params["Color"])

	decodedSpec, err := DecodeEngineSpec[testEngineSpec](spec)
	require.NoError(t, err)

	assert.Equal(t, expectedName, decodedSpec.Name)
	assert.Equal(t, expectedColors, decodedSpec.Colors)

	specBytes, err := spec.Serialize()
	require.NoError(t, err)
	roundTripSpec, err := DeserializeEngineSpec(specBytes)
	require.NoError(t, err)

	decodedRoundTripSpec, err := DecodeEngineSpec[testEngineSpec](roundTripSpec)
	require.NoError(t, err)

	assert.Equal(t, expectedName, decodedRoundTripSpec.Name)
	assert.Equal(t, expectedColors, decodedRoundTripSpec.Colors)
}
