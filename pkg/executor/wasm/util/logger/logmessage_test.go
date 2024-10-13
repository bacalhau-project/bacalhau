//go:build unit || !integration

package wasmlogs

import (
	"bufio"
	"bytes"
	"encoding/gob"
	"encoding/json"
	"io"
	"os"
	"testing"
	"time"

	_ "github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type LogMessageTestSuite struct {
	suite.Suite
	file          *os.File
	compactBuffer bytes.Buffer
}

func TestLogMessageTestSuite(t *testing.T) {
	suite.Run(t, new(LogMessageTestSuite))
}

func (s *LogMessageTestSuite) SetupTest() {
	f, err := os.CreateTemp("", "")
	require.NoError(s.T(), err)

	s.file = f
}

func (s *LogMessageTestSuite) TeardownTest() {
	s.file.Close()
}

func (s *LogMessageTestSuite) TestLogMessageSimpleWriteRead() {
	lm := &LogMessage{
		Stream:    LogStreamStdout,
		Timestamp: time.Now().Unix(),
		Data:      []byte("Jack and Jill went up the hill to fetch a pail of water"),
	}

	writeToFile(lm, s.file, s.compactBuffer)

	// Reset the file for reading
	s.file.Seek(0, io.SeekStart)
	reader := bufio.NewReader(s.file)

	lm2, err := readFromFile(reader)
	require.NoError(s.T(), err)
	require.Equal(s.T(), lm.Timestamp, lm2.Timestamp)
	require.Equal(s.T(), lm.Stream, lm2.Stream)
	require.Equal(s.T(), lm.Data, lm2.Data)
}

func (s *LogMessageTestSuite) TestLogMessageBinaryWriteRead() {
	lm := &LogMessage{
		Stream:    LogStreamStdout,
		Timestamp: time.Now().Unix(),
		Data:      []byte("Jack and Jill went up the hill to fetch a pail of water"),
	}

	lmBytes := lm.ToBytes()
	buff := bytes.NewBuffer(lmBytes)

	lmx := LogMessage{}
	rdr := bufio.NewReader(buff)
	lmx.FromReader(*rdr)

	require.Equal(s.T(), lm.Stream, lmx.Stream)
	require.Equal(s.T(), lm.Timestamp, lmx.Timestamp)
	require.Equal(s.T(), lm.Data, lmx.Data)
}

func (s *LogMessageTestSuite) TestLogMessageMany() {
	lm := &LogMessage{
		Stream:    LogStreamStdout,
		Timestamp: time.Now().Unix(),
		Data:      []byte("Jack and Jill went up the hill to fetch a pail of water"),
	}

	for i := 0; i < 1000; i++ {
		writeToFile(lm, s.file, s.compactBuffer)
	}

	s.file.Seek(0, io.SeekStart)
	reader := bufio.NewReader(s.file)
	for i := 0; i < 1000; i++ {
		x, err := readFromFile(reader)
		x.Timestamp += 1
		require.NoError(s.T(), err)
	}
}

func BenchmarkLogMessageWrite(b *testing.B) {
	lm := &LogMessage{
		Stream:    LogStreamStdout,
		Timestamp: time.Now().Unix(),
		Data:      []byte("Jack and Jill went up the hill to fetch a pail of water"),
	}
	f, _ := os.CreateTemp("", "")
	var compact bytes.Buffer

	for n := 0; n < b.N; n++ {
		for i := 0; i < 1000; i++ {
			writeToFile(lm, f, compact)
		}
	}
}

func BenchmarkBinaryMessageWrite(b *testing.B) {
	lm := &LogMessage{
		Stream:    LogStreamStdout,
		Timestamp: time.Now().Unix(),
		Data:      []byte("Jack and Jill went up the hill to fetch a pail of water"),
	}
	f, _ := os.CreateTemp("", "")

	for n := 0; n < b.N; n++ {
		for i := 0; i < 1000; i++ {
			f.Write(lm.ToBytes())
		}
	}
}

func BenchmarkGOBMessageWrite(b *testing.B) {

	// enc := gob.NewEncoder(&network) // Will write to network.
	// dec := gob.NewDecoder(&network) // Will read from network.
	lm := &LogMessage{
		Stream:    LogStreamStdout,
		Timestamp: time.Now().Unix(),
		Data:      []byte("Jack and Jill went up the hill to fetch a pail of water"),
	}
	f, _ := os.CreateTemp("", "")
	enc := gob.NewEncoder(f)
	for n := 0; n < b.N; n++ {
		for i := 0; i < 1000; i++ {
			enc.Encode(lm)
		}
	}
}

func BenchmarkLogMessageWriteAndRead(b *testing.B) {
	lm := &LogMessage{
		Stream:    LogStreamStdout,
		Timestamp: time.Now().Unix(),
		Data:      []byte("Jack and Jill went up the hill to fetch a pail of water"),
	}
	f, _ := os.CreateTemp("", "")
	var compact bytes.Buffer
	for i := 0; i < 1000; i++ {
		writeToFile(lm, f, compact)
	}

	counter := 0
	f.Seek(0, io.SeekStart)
	reader := bufio.NewReader(f)
	for n := 0; n < b.N; n++ {
		for i := 0; i < 1000; i++ {
			l, _ := readFromFile(reader)
			if l.Timestamp > 0 {
				counter += 1
			}
		}
	}
}

func BenchmarkLogMessageBinaryWriteAndRead(b *testing.B) {
	lm := &LogMessage{
		Stream:    LogStreamStdout,
		Timestamp: time.Now().Unix(),
		Data:      []byte("Jack and Jill went up the hill to fetch a pail of water"),
	}
	f, _ := os.CreateTemp("", "")

	for i := 0; i < 1000; i++ {
		f.Write(lm.ToBytes())
	}

	counter := 0
	f.Seek(0, io.SeekStart)
	reader := bufio.NewReader(f)
	for n := 0; n < b.N; n++ {
		for i := 0; i < 1000; i++ {
			l := LogMessage{}
			l.FromReader(*reader)
			if l.Timestamp > 0 {
				counter += 1
			}
		}
	}
}

func BenchmarkLogMessageGOBWriteAndRead(b *testing.B) {
	lm := &LogMessage{
		Stream:    LogStreamStdout,
		Timestamp: time.Now().Unix(),
		Data:      []byte("Jack and Jill went up the hill to fetch a pail of water"),
	}
	f, _ := os.CreateTemp("", "")

	enc := gob.NewEncoder(f)

	for i := 0; i < 1000; i++ {
		enc.Encode(lm)
	}

	counter := 0
	f.Seek(0, io.SeekStart)

	dec := gob.NewDecoder(f)
	for n := 0; n < b.N; n++ {
		for i := 0; i < 1000; i++ {
			l := LogMessage{}
			dec.Decode(&l)
			if l.Timestamp > 0 {
				counter += 1
			}
		}
	}
}

func BenchmarkLogMessageSeekWriteAndRead(b *testing.B) {
	lm := &LogMessage{
		Stream:    LogStreamStdout,
		Timestamp: time.Now().Unix(),
		Data:      []byte("Jack and Jill went up the hill to fetch a pail of water"),
	}
	f, _ := os.CreateTemp("", "")
	var compact bytes.Buffer
	flip := true
	for i := 0; i < 1000; i++ {
		if flip {
			lm.Stream = LogStreamStdout
		} else {
			lm.Stream = LogStreamStderr
		}
		writeToFile(lm, f, compact)
		flip = !flip
	}

	counter := 0
	f.Seek(0, io.SeekStart)
	reader := bufio.NewReader(f)
	for n := 0; n < b.N; n++ {
		for i := 0; i < 1000; i++ {
			l, _ := readFromFile(reader)
			if l.Stream == LogStreamStdout {
				counter += 1
			}
		}
	}
}

func BenchmarkLogMessageBinarySeekWriteAndRead(b *testing.B) {
	lm := &LogMessage{
		Stream:    LogStreamStdout,
		Timestamp: time.Now().Unix(),
		Data:      []byte("Jack and Jill went up the hill to fetch a pail of water"),
	}
	f, _ := os.CreateTemp("", "")

	flip := true
	for i := 0; i < 1000; i++ {
		if flip {
			lm.Stream = LogStreamStdout
		} else {
			lm.Stream = LogStreamStderr
		}
		f.Write(lm.ToBytes())
		flip = !flip
	}

	counter := 0
	f.Seek(0, io.SeekStart)
	reader := bufio.NewReader(f)
	for n := 0; n < b.N; n++ {
		for i := 0; i < 1000; i++ {
			l := LogMessage{}
			_ = l.FromReader(*reader)
			if l.Stream == LogStreamStdout {
				counter += 1
			}
		}
	}
}

func BenchmarkLogMessageGOBSeekWriteAndRead(b *testing.B) {
	lm := &LogMessage{
		Stream:    LogStreamStdout,
		Timestamp: time.Now().Unix(),
		Data:      []byte("Jack and Jill went up the hill to fetch a pail of water"),
	}
	f, _ := os.CreateTemp("", "")
	enc := gob.NewEncoder(f)

	flip := true
	for i := 0; i < 1000; i++ {
		if flip {
			lm.Stream = LogStreamStdout
		} else {
			lm.Stream = LogStreamStderr
		}
		enc.Encode(lm)
		flip = !flip
	}

	counter := 0
	f.Seek(0, io.SeekStart)
	dec := gob.NewDecoder(f)
	for n := 0; n < b.N; n++ {
		for i := 0; i < 1000; i++ {
			l := LogMessage{}
			dec.Decode(&l)
			if l.Stream == LogStreamStdout {
				counter += 1
			}
		}
	}
}

func writeToFile(msg *LogMessage, file *os.File, compactBuffer bytes.Buffer) {
	compactBuffer.Reset()
	data, _ := json.Marshal(msg)

	_ = json.Compact(&compactBuffer, data)

	compactBuffer.Write([]byte{'\n'})

	// write msg to file and also broadcast the message
	_, _ = file.Write(compactBuffer.Bytes())
}

func readFromFile(reader *bufio.Reader) (*LogMessage, error) {
	data, _ := reader.ReadBytes('\n')

	var msg LogMessage
	_ = json.Unmarshal(data, &msg)

	return &msg, nil
}
