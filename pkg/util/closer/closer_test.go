package closer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"testing"

	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCloseWithLogOnError_noErrors(t *testing.T) {
	old := log.Logger
	t.Cleanup(func() {
		log.Logger = old
	})

	var sb strings.Builder
	CloseWithLogOnError(t.Name(), closer{nil})
	assert.Equal(t, "", sb.String())
}

func TestCloseWithLogOnError_logsErrors(t *testing.T) {
	old := log.Logger
	t.Cleanup(func() {
		log.Logger = old
	})

	var b bytes.Buffer
	log.Logger = log.With().Str("foo", "bar").Logger().Output(&b)

	CloseWithLogOnError(t.Name(), closer{fmt.Errorf("error message")})

	var content map[string]string
	require.NoError(t, json.Unmarshal(b.Bytes(), &content))

	assert.Equal(t, "bar", content["foo"])
	assert.NotEmpty(t, content["message"])
	assert.NotEmpty(t, content["caller"])
	assert.True(t, strings.HasSuffix(content["caller"], "closer/closer_test.go:38"), "%s should point to the function call", content["caller"])
}

func TestCloseWithLogOnError_ignoresAlreadyClosed(t *testing.T) {
	tests := []error{os.ErrClosed, net.ErrClosed}
	for _, test := range tests {
		t.Run(test.Error(), func(t *testing.T) {
			old := log.Logger
			t.Cleanup(func() {
				log.Logger = old
			})

			var sb strings.Builder
			CloseWithLogOnError(t.Name(), closer{test})
			assert.Equal(t, "", sb.String())
		})
	}
}

var _ io.Closer = closer{}

type closer struct {
	err error
}

func (c closer) Close() error {
	return c.err
}
