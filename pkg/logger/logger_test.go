//go:build unit || !integration

package logger

import (
	"errors"
	"regexp"
	"strings"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/logger/testpackage/subpackage/subsubpackage"
	ipfslog2 "github.com/ipfs/go-log/v2"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
)

func TestConfigureLogging(t *testing.T) {
	oldLogger := log.Logger
	oldContextLogger := zerolog.DefaultContextLogger

	t.Cleanup(func() {
		log.Logger = oldLogger
		zerolog.DefaultContextLogger = oldContextLogger
	})

	var logging strings.Builder
	configureLogging(zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
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

// TestConfigureIpfsLogging checks that we configure IPFS logging correctly, forwarding logging to zerolog.
func TestConfigureIpfsLogging(t *testing.T) {
	var logging strings.Builder
	configureLogging(zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
		defaultLogFormat(w)
		w.Out = &logging
		w.NoColor = true
	}))

	l := ipfslog2.Logger("name")
	l.With("hello", "world", "err", errors.New("example")).Error("test")

	actual := logging.String()
	// Like 12:06:50.55 | ERR pkg/logger/logger_test.go:52 > test [err:example] [hello:world] [logger-name:name]
	t.Log(actual)

	assert.Regexp(t, regexp.MustCompile(`ERR pkg/logger/logger_test.go:\d* > test`), actual)
	assert.Contains(t, actual, "[hello:world]")
	assert.Contains(t, actual, "[logger-name:name]")
	assert.Contains(t, actual, "[err:example]")
}
