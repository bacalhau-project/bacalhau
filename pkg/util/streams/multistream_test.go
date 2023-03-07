//go:build unit || !integration

package streams

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type testReader struct {
	data  [][]byte
	index int
}

func (t *testReader) Read(b []byte) (int, error) {
	if t.index >= len(t.data) {
		return 0, io.EOF
	}

	readCount := copy(b, t.data[t.index])
	t.index += 1

	return readCount, nil
}

type MultistreamTestSuite struct {
	suite.Suite
}

func TestMultistreamSuite(t *testing.T) {
	suite.Run(t, new(MultistreamTestSuite))
}

func (s *MultistreamTestSuite) TestMultiSourceStream() {
	ctx := context.Background()

	type testData struct {
		name               string
		readers            []TaggedReader
		shouldError        bool
		expectedTotalBytes int
		expectedStdOut     string
		expectedStdErr     string
	}

	testCases := []testData{
		{
			name: "Simple Stdout Read",
			readers: []TaggedReader{
				NewTaggedReader(
					StdOutTag,
					&testReader{data: [][]byte{
						{'1', '2', '3'},
					}}),
			},
			shouldError:        false,
			expectedTotalBytes: 3,
			expectedStdOut:     "123",
			expectedStdErr:     "",
		},
		{
			name: "Multi reads from stdout",
			readers: []TaggedReader{
				NewTaggedReader(
					StdOutTag,
					&testReader{data: [][]byte{
						{'1', '2', '3'},
						{'1', '2', '3'},
					}}),
			},
			shouldError:        false,
			expectedTotalBytes: 6,
			expectedStdOut:     "123123",
			expectedStdErr:     "",
		},
		{
			name: "Multireads from stdout and stderr",
			readers: []TaggedReader{
				NewTaggedReader(
					StdOutTag,
					&testReader{data: [][]byte{
						{'1', '2', '3'},
						{'1', '2', '3'},
					}}),
				NewTaggedReader(
					StdErrTag,
					&testReader{data: [][]byte{
						{'1', '2', '3'},
						{'1', '2', '3'},
					}}),
			},
			shouldError:        false,
			expectedTotalBytes: 12,
			expectedStdOut:     "123123",
			expectedStdErr:     "123123",
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			var buffer bytes.Buffer

			// Read from the readers and write the tagged output to the
			// byte buffer so we can check what was written afterwards.
			ms := NewMultiSourceStream(&buffer, tc.readers...)
			copiedBytes, err := ms.Copy(ctx)

			if tc.shouldError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tc.expectedTotalBytes, copiedBytes, "not enough data was copied")

			var stdOutBuilder strings.Builder
			var stdErrBuilder strings.Builder

			// Read the data frames out of the buffer one at a time so we can build
			// strings for stdout and stderr and check what was written
			lenBytes := make([]byte, 2)
			for buffer.Len() > 0 {

				_, _ = buffer.Read(lenBytes)
				length := binary.BigEndian.Uint16(lenBytes)

				tag, _ := buffer.ReadByte()

				dataBuf := make([]byte, length)
				buffer.Read(dataBuf)

				if Tag(tag) == StdOutTag {
					stdOutBuilder.WriteString(string(dataBuf))
				} else if Tag(tag) == StdErrTag {
					stdErrBuilder.WriteString(string(dataBuf))
				}
			}

			require.Equal(t, tc.expectedStdOut, stdOutBuilder.String())
			require.Equal(t, tc.expectedStdErr, stdErrBuilder.String())

		})
	}

}
