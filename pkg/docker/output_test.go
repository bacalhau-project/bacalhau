//go:build unit || !integration

package docker

import (
	"bytes"
	"encoding/binary"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/pkg/stdcopy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTimestampedStdCopy_BasicOutput(t *testing.T) {
	// Setup test data
	now := time.Now()
	stdoutContent := "stdout 1"
	stderrContent := "stderr 22"

	// Create test frames
	frames := []byte{}
	stdOutFrame := createTimestampedFrame(stdcopy.Stdout, now, stdoutContent)
	stdErrFrame := createTimestampedFrame(stdcopy.Stderr, now, stderrContent)
	frames = append(frames, stdOutFrame...)
	frames = append(frames, stdErrFrame...)

	// Setup output buffer
	var output bytes.Buffer

	// Call function under test
	written, err := TimestampedStdCopy(&output, bytes.NewReader(frames), now.Add(-1*time.Hour), false)

	// Assert results
	require.NoError(t, err)
	// Should write 2 frame headers (8 bytes each) + the lengths of the contents
	assert.Equal(t, int64(stdWriterPrefixLen*2+len(stdoutContent)+len(stderrContent)), written)

	assertStdCopyCompatibleOutput(t, &output, []byte(stdoutContent), []byte(stderrContent))
}

func TestTimestampedStdCopy_TimestampFiltering(t *testing.T) {
	// Setup test data with different timestamps
	now := time.Now().UTC()
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)

	pastContent := "This is past content"
	nowContent := "This is current content"
	futureContent := "This is future content"

	// Create test frames
	frames := []byte{}
	frames = append(frames, createTimestampedFrame(stdcopy.Stdout, past, pastContent)...)
	frames = append(frames, createTimestampedFrame(stdcopy.Stdout, now, nowContent)...)
	frames = append(frames, createTimestampedFrame(stdcopy.Stdout, future, futureContent)...)

	// Test 1: Filter out past content
	var output1 bytes.Buffer
	written1, err := TimestampedStdCopy(&output1, bytes.NewReader(frames), now, false)
	require.NoError(t, err)
	assert.Equal(t, int64(stdWriterPrefixLen*2+len(nowContent)+len(futureContent)), written1)
	assertStdCopyCompatibleOutput(t, &output1, append([]byte(nowContent), []byte(futureContent)...), nil)

	// Test 2: Filter out everything
	var output2 bytes.Buffer
	written2, err := TimestampedStdCopy(&output2, bytes.NewReader(frames), future.Add(1*time.Hour), false)
	require.NoError(t, err)
	assert.Equal(t, int64(0), written2)
	assertStdCopyCompatibleOutput(t, &output2, nil, nil)

	// Test 3: Include all content
	var output3 bytes.Buffer
	written3, err := TimestampedStdCopy(&output3, bytes.NewReader(frames), past.Add(-1*time.Second), false)
	require.NoError(t, err)
	assert.Equal(t, int64(stdWriterPrefixLen*3+len(pastContent)+len(nowContent)+len(futureContent)), written3)
	assertStdCopyCompatibleOutput(t, &output3, append([]byte(pastContent), append([]byte(nowContent), []byte(futureContent)...)...), nil)
}

func TestTimestampedStdCopy_EmptyInput(t *testing.T) {
	var output bytes.Buffer
	written, err := TimestampedStdCopy(&output, bytes.NewReader([]byte{}), time.Time{}, false)
	require.NoError(t, err)
	assert.Equal(t, int64(0), written)
	assertStdCopyCompatibleOutput(t, &output, nil, nil)
}

func TestTimestampedStdCopy_SystemError(t *testing.T) {
	// Setup error frame
	now := time.Now()
	errorMsg := "daemon error message"
	frame := createTimestampedFrame(stdcopy.Systemerr, now, errorMsg)

	var output bytes.Buffer
	_, err := TimestampedStdCopy(&output, bytes.NewReader(frame), now.Add(-1*time.Hour), false)

	require.Error(t, err)
	assert.Contains(t, err.Error(), errorMsg)
}

func TestTimestampedStdCopy_PartialRead(t *testing.T) {
	// Setup test data
	now := time.Now()
	content := "This is test content"

	frame := createTimestampedFrame(stdcopy.Stdout, now, content)

	// Create a reader that returns data in chunks
	chunkSize := 10
	partialReader := &chunkReader{
		data:      frame,
		chunkSize: chunkSize,
	}

	var output bytes.Buffer
	written, err := TimestampedStdCopy(&output, partialReader, now.Add(-1*time.Hour), false)

	require.NoError(t, err)
	assert.Equal(t, int64(stdWriterPrefixLen+len(content)), written)
	assertStdCopyCompatibleOutput(t, &output, []byte(content), nil)
}

func TestTimestampedStdCopy_InvalidTimestamp(t *testing.T) {
	// Create a frame with invalid timestamp format
	header := make([]byte, stdWriterPrefixLen)
	header[stdWriterFdIndex] = byte(stdcopy.Stdout)

	invalidTimestamp := "not-a-timestamp" + strings.Repeat(" ", timestampLen-len("not-a-timestamp"))
	content := "content"
	data := invalidTimestamp + content
	dataLen := len(data)

	binary.BigEndian.PutUint32(header[stdWriterSizeIndex:], uint32(dataLen))
	frame := append(header, []byte(data)...)

	var output bytes.Buffer
	_, err := TimestampedStdCopy(&output, bytes.NewReader(frame), time.Time{}, false)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing time")
}

func TestTimestampedStdCopy_TooSmallFrame(t *testing.T) {
	// Create a frame with size smaller than timestampLen
	header := make([]byte, stdWriterPrefixLen)
	header[stdWriterFdIndex] = byte(stdcopy.Stdout)

	// Set frame size to less than timestampLen
	binary.BigEndian.PutUint32(header[stdWriterSizeIndex:], uint32(timestampLen-1))

	var output bytes.Buffer
	_, err := TimestampedStdCopy(&output, bytes.NewReader(header), time.Time{}, false)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "frame is too short")
}

func TestTimestampedStdCopy_WriterError(t *testing.T) {
	// Setup test data
	now := time.Now()
	content := "This is test content"

	frame := createTimestampedFrame(stdcopy.Stdout, now, content)

	// Create a writer that always returns an error
	errorWriter := &errorWriter{err: io.ErrClosedPipe}

	_, err := TimestampedStdCopy(errorWriter, bytes.NewReader(frame), now.Add(-1*time.Hour), false)

	require.Error(t, err)
	assert.Equal(t, io.ErrClosedPipe, err)
}

func TestTimestampedStdCopy_LargeFrames(t *testing.T) {
	// Create a large content that exceeds the initial buffer size
	now := time.Now()
	largeContent := strings.Repeat("X", startingBufLen)

	frame := createTimestampedFrame(stdcopy.Stdout, now, largeContent)

	var output bytes.Buffer
	written, err := TimestampedStdCopy(&output, bytes.NewReader(frame), now.Add(-1*time.Hour), false)

	require.NoError(t, err)
	assert.Equal(t, int64(stdWriterPrefixLen+len(largeContent)), written)
	assertStdCopyCompatibleOutput(t, &output, []byte(largeContent), nil)
}

func TestTimestampedStdCopy_IncompleteFrame(t *testing.T) {
	// Create a normal frame
	now := time.Now()
	content := "This is test content"
	frame := createTimestampedFrame(stdcopy.Stdout, now, content)

	// Cut the frame to simulate incomplete read
	incompleteFrame := frame[:len(frame)-10]

	var output bytes.Buffer
	written, err := TimestampedStdCopy(&output, bytes.NewReader(incompleteFrame), now.Add(-1*time.Hour), false)

	require.NoError(t, err)            // EOF with incomplete frame should just return
	assert.Equal(t, int64(0), written) // Nothing should be written
}

func TestTimestampedStdCopy_ShortWrite(t *testing.T) {
	// Setup test data
	now := time.Now()
	content := "This is test content"

	frame := createTimestampedFrame(stdcopy.Stdout, now, content)

	// Create a writer that always writes less than requested
	shortWriter := &shortWriter{}

	_, err := TimestampedStdCopy(shortWriter, bytes.NewReader(frame), now.Add(-1*time.Hour), false)

	require.Error(t, err)
	assert.Equal(t, io.ErrShortWrite, err)
}

// Docker timestamps are RFC3339Nano in UTC timezone 0-padded for alignment.
// Ex 2025-03-31T20:49:22.115836289Z, 2025-03-31T20:49:22.115800000Z
func toDockerTimestamp(timestamp time.Time) string {
	timestampStr := timestamp.UTC().Format(time.RFC3339Nano)
	paddingLen := timestampLen - len(timestampStr)
	if paddingLen > 0 {
		paddingStr := strings.Repeat("0", paddingLen)
		timestampStr = timestampStr[:len(timestampStr)-len("Z")] + paddingStr[:] + "Z"
	}
	return timestampStr
}

func createTimestampedFrame(stdType stdcopy.StdType, timestamp time.Time, content string) []byte {
	timestampStr := toDockerTimestamp(timestamp.UTC())
	// Timestamps are separated from the actual data with a space
	data := timestampStr + " " + content
	dataLen := len(data)

	// Create header
	header := make([]byte, stdWriterPrefixLen)
	header[stdWriterFdIndex] = byte(stdType)
	binary.BigEndian.PutUint32(header[stdWriterSizeIndex:], uint32(dataLen))

	// Combine header and data
	frame := append(header, []byte(data)...)
	return frame
}

// verify that the output stream can be parsed by the original StdCopy method
func assertStdCopyCompatibleOutput(t *testing.T, actualOutput io.Reader, expectedStdOut []byte, expectedStdErr []byte) {
	var actualStdOutBuf, actualStdErrBuf bytes.Buffer
	_, err := stdcopy.StdCopy(&actualStdOutBuf, &actualStdErrBuf, actualOutput)
	require.NoError(t, err)
	assert.Equal(t, expectedStdOut, actualStdOutBuf.Bytes())
	assert.Equal(t, expectedStdErr, actualStdErrBuf.Bytes())
}

// chunkReader implements io.Reader and returns data in chunks
type chunkReader struct {
	data      []byte
	offset    int
	chunkSize int
}

func (r *chunkReader) Read(p []byte) (int, error) {
	if r.offset >= len(r.data) {
		return 0, io.EOF
	}

	end := r.offset + r.chunkSize
	if end > len(r.data) {
		end = len(r.data)
	}

	n := copy(p, r.data[r.offset:end])
	r.offset += n

	return n, nil
}

// errorWriter implements io.Writer and always returns an error
type errorWriter struct {
	err error
}

func (w *errorWriter) Write(p []byte) (int, error) {
	return 0, w.err
}

// shortWriter implements io.Writer and always writes less than requested
type shortWriter struct{}

func (w *shortWriter) Write(p []byte) (int, error) {
	if len(p) > 0 {
		return len(p) - 1, nil
	}
	return 0, nil
}
