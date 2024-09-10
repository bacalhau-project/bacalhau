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
	configureLogging(zerolog.InfoLevel, zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
		defaultLogFormat(w)
		w.Out = &logging
		w.NoColor = true
	}))

	subsubpackage.TestLog("testing error logging", "testing message")

	actual := logging.String()
	// Like 12:47:40.875 | ERR pkg/logger/testpackage/subpackage/subsubpackage/testutil.go:12 > testing message error="testing error logging" [stack:[{"func":"TestLog","line":"10","source":"testutil.go"},{"func":"TestConfigureLogging","line":"27","source":"logger_test.go"},...]]
	t.Log(actual)

	assert.Contains(t, actual, "testing message", "Log statement doesn't contain the log message")
	assert.Contains(t, actual, `error="testing error logging"`, "Log statement doesn't contain the logged error")
	assert.Contains(t, actual, "pkg/logger/testpackage/subpackage/subsubpackage/testutil.go", "Log statement doesn't contain the full package path")
	assert.Contains(t, actual, `stack:[{"func":"TestLog","line":`, "Log statement didn't automatically include the error's stacktrace")
}
