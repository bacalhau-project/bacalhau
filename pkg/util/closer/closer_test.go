//go:build unit || !integration

package closer

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCloseWithLogOnError_noErrors(t *testing.T) {
	logOutput := testLogger(t)

	CloseWithLogOnError(t.Name(), closer{nil, strings.NewReader("")})
	assert.Equal(t, "", logOutput.String())
}

func TestCloseWithLogOnError_logsErrors(t *testing.T) {
	logOutput := testLogger(t, func(context zerolog.Context) zerolog.Context {
		return context.Str("foo", "bar")
	})

	CloseWithLogOnError(t.Name(), closer{fmt.Errorf("error message"), strings.NewReader("")})

	var content map[string]string
	require.NoError(t, model.JSONUnmarshalWithMax(logOutput.Bytes(), &content))

	assert.Equal(t, "bar", content["foo"])
	assert.NotEmpty(t, content["message"])
	assert.NotEmpty(t, content["caller"])
	assert.True(t, strings.HasSuffix(content["caller"], "closer/closer_test.go:34"), "%s should point to the CloseWithLogOnError function call in this test", content["caller"])
}

func TestCloseWithLogOnError_ignoresAlreadyClosed(t *testing.T) {
	tests := []error{os.ErrClosed, net.ErrClosed}
	for _, test := range tests {
		t.Run(test.Error(), func(t *testing.T) {
			logOutput := testLogger(t)

			CloseWithLogOnError(t.Name(), closer{test, strings.NewReader("")})
			assert.Equal(t, "", logOutput.String())
		})
	}
}

func TestDrainAndCloseWithLogOnError_noErrors(t *testing.T) {
	logOutput := testLogger(t)

	reader := strings.NewReader("")
	DrainAndCloseWithLogOnError(context.Background(), "example", closer{nil, reader})
	assert.Equal(t, "", logOutput.String())
	_, err := reader.ReadByte()
	assert.Equal(t, io.EOF, err)
}

func TestDrainAndCloseWithLogOnError_logsDrainError(t *testing.T) {
	logOutput := testLogger(t, func(context zerolog.Context) zerolog.Context {
		return context.Str("foo", "marker")
	})

	DrainAndCloseWithLogOnError(context.Background(), "example", closer{nil, failingReader{fmt.Errorf("error message")}})

	var content map[string]string
	require.NoError(t, model.JSONUnmarshalWithMax(logOutput.Bytes(), &content))

	assert.Equal(t, "marker", content["foo"])
	assert.NotEmpty(t, content["message"])
	assert.NotEmpty(t, content["caller"])
	assert.True(t, strings.HasSuffix(content["caller"], "closer/closer_test.go:72"), "%s should point to the DrainAndCloseWithLogOnError function call in this test", content["caller"])
}

func testLogger(t *testing.T, mods ...func(context zerolog.Context) zerolog.Context) *bytes.Buffer {
	old := log.Logger
	oldContext := zerolog.DefaultContextLogger
	t.Cleanup(func() {
		log.Logger = old
		zerolog.DefaultContextLogger = oldContext
	})

	l := log.With()
	for _, mod := range mods {
		l = mod(l)
	}

	var b bytes.Buffer
	log.Logger = l.Logger().Output(&b)
	zerolog.DefaultContextLogger = &log.Logger

	return &b
}

var _ io.Closer = closer{}
var _ io.Reader = closer{}

type closer struct {
	closeErr error
	reader   io.Reader
}

func (c closer) Read(p []byte) (int, error) {
	return c.reader.Read(p)
}

func (c closer) Close() error {
	return c.closeErr
}

var _ io.Reader = failingReader{}

type failingReader struct {
	err error
}

func (f failingReader) Read([]byte) (int, error) {
	return 0, f.err
}
