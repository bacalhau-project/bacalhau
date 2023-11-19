//go:build unit || !integration

package template

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewParserNoReplacements(t *testing.T) {
	parser, err := NewParser(ParserParams{})
	require.NoError(t, err, "Creating parser should not produce an error")
	assert.Empty(t, parser.replacements, "Parser should have no replacements")
}

func TestNewParserWithReplacements(t *testing.T) {
	replacements := map[string]string{"key1": "value1", "key2": "value2"}
	parser, err := NewParser(ParserParams{Replacements: replacements})
	require.NoError(t, err, "Creating parser should not produce an error")
	assert.Equal(t, replacements, parser.replacements, "Replacements should match provided map")
}

func TestNewParserWithEnvVars(t *testing.T) {
	setEnvVars(t, map[string]string{"ENV_VAR": "env_value", "OTHER_ENV_VAR": "other_env_value"})
	defer unsetEnvVars(t, []string{"ENV_VAR"})

	t.Run("WithNoEnvVars", func(t *testing.T) {
		parser, err := NewParser(ParserParams{})
		require.NoError(t, err, "Creating parser should not produce an error")
		assert.Empty(t, parser.replacements, "Parser should have no replacements")
	})

	t.Run("WithAllEnvVars", func(t *testing.T) {
		parser, err := NewParser(ParserParams{EnvPattern: "*"})
		require.NoError(t, err, "Creating parser should not produce an error")
		assert.Equal(t, "env_value", parser.replacements["ENV_VAR"], "Environment variable should be in replacements")
		assert.Equal(t, "other_env_value", parser.replacements["OTHER_ENV_VAR"], "Environment variable should be in replacements")
	})

	t.Run("WithAllEnvVarsAsterisk", func(t *testing.T) {
		parser, err := NewParser(ParserParams{EnvPattern: "*"})
		require.NoError(t, err, "Creating parser should not produce an error")
		assert.Equal(t, "env_value", parser.replacements["ENV_VAR"], "Environment variable should be in replacements")
		assert.Equal(t, "other_env_value", parser.replacements["OTHER_ENV_VAR"], "Environment variable should be in replacements")
	})

	t.Run("WithNonMatchingEnvVars", func(t *testing.T) {
		parser, err := NewParser(ParserParams{EnvPattern: "NON_EXISTENT_*"})
		require.NoError(t, err, "Creating parser should not produce an error")
		assert.Empty(t, parser.replacements, "Parser should have no replacements")
	})

	t.Run("WithSingleMatchingEnvVars", func(t *testing.T) {
		parser, err := NewParser(ParserParams{EnvPattern: "OTHER_*"})
		require.NoError(t, err, "Creating parser should not produce an error")
		assert.Empty(t, parser.replacements["ENV_VAR"], "Environment variable should not be in replacements")
		assert.Equal(t, "other_env_value", parser.replacements["OTHER_ENV_VAR"], "Environment variable should be in replacements")
	})

	t.Run("WithMultipleMatchingEnvVars", func(t *testing.T) {
		parser, err := NewParser(ParserParams{EnvPattern: "ENV_*|OTHER_*"})
		require.NoError(t, err, "Creating parser should not produce an error")
		assert.Equal(t, "env_value", parser.replacements["ENV_VAR"], "Environment variable should be in replacements")
		assert.Equal(t, "other_env_value", parser.replacements["OTHER_ENV_VAR"], "Environment variable should be in replacements")
	})
}

func TestParseWithReplacements(t *testing.T) {
	replacements := map[string]string{"key": "value"}
	parser, err := NewParser(ParserParams{Replacements: replacements})
	require.NoError(t, err, "Creating parser should not produce an error")

	result, err := parser.Parse("This is a {{.key}}")
	require.NoError(t, err, "Parsing should not produce an error")
	assert.Equal(t, "This is a value", result, "Content should be replaced correctly")
}

func TestParseBytesWithReplacements(t *testing.T) {
	replacements := map[string]string{"key": "value"}
	parser, err := NewParser(ParserParams{Replacements: replacements})
	require.NoError(t, err, "Creating parser should not produce an error")

	result, err := parser.ParseBytes([]byte("This is a {{.key}}"))
	require.NoError(t, err, "Parsing should not produce an error")
	assert.Equal(t, []byte("This is a value"), result, "Content should be replaced correctly")
}

func TestParseWithEnvVars(t *testing.T) {
	setEnvVars(t, map[string]string{"ENV_KEY": "env_value"})
	defer unsetEnvVars(t, []string{"ENV_KEY"})

	parser, err := NewParser(ParserParams{EnvPattern: "*"})
	require.NoError(t, err, "Creating parser should not produce an error")
	result, err := parser.Parse("This is an {{.ENV_KEY}}")
	require.NoError(t, err, "Parsing should not produce an error")
	assert.Equal(t, "This is an env_value", result, "Environment variable should be replaced correctly")
}

func TestParseReplacementsTakePrecedenceOverEnvVars(t *testing.T) {
	setEnvVars(t, map[string]string{"PLACEHOLDER": "env_value"})
	defer unsetEnvVars(t, []string{"PLACEHOLDER"})

	// parser with only env vars
	parser, err := NewParser(ParserParams{EnvPattern: "*"})
	require.NoError(t, err, "Creating parser should not produce an error")

	result, err := parser.Parse("Value is {{.PLACEHOLDER}}")
	require.NoError(t, err, "Parsing should not produce an error")
	assert.Equal(t, "Value is env_value", result, "Environment variable should be replaced correctly")

	// parser with replacements and env vars
	replacements := map[string]string{"PLACEHOLDER": "replacement_value"}
	parser, err = NewParser(ParserParams{Replacements: replacements, EnvPattern: "*"})
	require.NoError(t, err, "Creating parser should not produce an error")

	result, err = parser.Parse("Value is {{.PLACEHOLDER}}")
	require.NoError(t, err, "Parsing should not produce an error")
	assert.Equal(t, "Value is replacement_value", result, "Replacement value should take precedence over environment variable")
}

func TestParseNoPlaceholders(t *testing.T) {
	parser, err := NewParser(ParserParams{Replacements: map[string]string{"key": "value"}})
	require.NoError(t, err, "Creating parser should not produce an error")
	input := "This string has no placeholders."

	result, err := parser.Parse(input)
	require.NoError(t, err, "Parsing should not produce an error")
	assert.Equal(t, input, result, "Output should be identical to input when there are no placeholders")
}

func TestParseUnknownPlaceholders(t *testing.T) {
	parser, err := NewParser(ParserParams{Replacements: map[string]string{"knownKey": "value"}})
	require.NoError(t, err, "Creating parser should not produce an error")
	input := "This {{.unknownKey}} remains unchanged."

	_, err = parser.Parse(input)
	require.Error(t, err, "Parsing should produce an error")
}

func setEnvVars(t *testing.T, vars map[string]string) {
	for key, value := range vars {
		require.NoError(t, os.Setenv(key, value), "Setting environment variable should not produce an error")
	}
}

func unsetEnvVars(t *testing.T, vars []string) {
	for _, key := range vars {
		require.NoError(t, os.Unsetenv(key), "Unsetting environment variable should not produce an error")
	}
}
