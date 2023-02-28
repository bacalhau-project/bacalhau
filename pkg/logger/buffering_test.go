package logger

import (
	"bufio"
	"bytes"
	"encoding/json"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestBufferingLogger_allowsBufferedLogsToBeWritten(t *testing.T) {
	w := bufferLogs()
	assert.NotNil(t, logBufferedLogs)

	logger := zerolog.New(w).With().Caller().Logger()
	logger.Info().Msg("test message")
	// repeated so we can check the details are from when the message was logged not buffered
	logger.Info().Msg("test message")

	var sb bytes.Buffer
	LogBufferedLogs(&sb)
	assert.Nil(t, logBufferedLogs)

	scan := bufio.NewScanner(&sb)

	t.Log(sb.String())

	var lines []map[string]string
	for scan.Scan() {
		var line map[string]string
		require.NoError(t, json.Unmarshal(scan.Bytes(), &line))

		lines = append(lines, line)
	}

	require.Len(t, lines, 2)

	assert.Equal(t, lines[0]["level"], lines[1]["level"])
	assert.Equal(t, lines[0]["message"], lines[1]["message"])
	assert.NotEqual(t, lines[0]["caller"], lines[1]["caller"])
}
