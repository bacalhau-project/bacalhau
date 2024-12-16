//go:build unit || !integration

package logger

import (
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"

	"github.com/bacalhau-project/bacalhau/pkg/logger/testpackage/subpackage/subsubpackage"
)

func TestConfigureLogging(t *testing.T) {
	oldLogger := log.Logger
	oldContextLogger := zerolog.DefaultContextLogger

	t.Cleanup(func() {
		log.Logger = oldLogger
		zerolog.DefaultContextLogger = oldContextLogger
	})

	var logging strings.Builder
	ConfigureLoggingLevel(zerolog.InfoLevel)
	configureLogging(zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
		defaultLogFormat(w)
		w.Out = &logging
		w.NoColor = true
	}))

	subsubpackage.TestLog("testing error logging", "testing message")

	actual := logging.String()
	// Like 12:47:40.875 | ERR  > testing message error="testing error logging" stack=[{"func":"TestLog","line":"10","source":"testutil.go"},{"func":"TestConfigureLogging","line":"27","source":"logger_test.go"},...]
	t.Log(actual)

	assert.Contains(t, actual, "testing message", "Log statement doesn't contain the log message")
	assert.Contains(t, actual, `error="testing error logging"`, "Log statement doesn't contain the logged error")
	assert.Contains(t, actual, `stack=[{"func":"TestLog","line":`, "Log statement didn't automatically include the error's stacktrace")
}

func TestParseAndConfigureLogging(t *testing.T) {
	err := ParseAndConfigureLogging("default", "debug")
	assert.NoError(t, err)
	assert.Equal(t, zerolog.DebugLevel, zerolog.GlobalLevel())

	err = ParseAndConfigureLogging("json", "info")
	assert.NoError(t, err)
	assert.Equal(t, zerolog.InfoLevel, zerolog.GlobalLevel())

	err = ParseAndConfigureLogging("invalid", "error")
	assert.Error(t, err)

	err = ParseAndConfigureLogging("default", "invalid")
	assert.Error(t, err)
}

func TestParseLogMode(t *testing.T) {
	tests := []struct {
		input    string
		expected LogMode
		hasError bool
	}{
		{"default", LogModeDefault, false},
		{"json", LogModeJSON, false},
		{"cmd", LogModeCmd, false},
		{"invalid", "", true},
	}

	for _, test := range tests {
		result, err := ParseLogMode(test.input)
		if test.hasError {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
			assert.Equal(t, test.expected, result)
		}
	}
}

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected zerolog.Level
		hasError bool
	}{
		{"debug", zerolog.DebugLevel, false},
		{"info", zerolog.InfoLevel, false},
		{"warn", zerolog.WarnLevel, false},
		{"error", zerolog.ErrorLevel, false},
		{"invalid", zerolog.NoLevel, true},
	}

	for _, test := range tests {
		result, err := ParseLogLevel(test.input)
		if test.hasError {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
			assert.Equal(t, test.expected, result)
		}
	}
}
