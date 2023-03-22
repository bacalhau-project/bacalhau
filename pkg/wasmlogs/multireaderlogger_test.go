//go:build unit || !integration

package wasmlogs

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	_ "github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/test/scenario"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type MultiReaderLoggerSuite struct {
	scenario.ScenarioRunner
}

func TestMultiReaderLoggerSuite(t *testing.T) {
	suite.Run(t, new(MultiReaderLoggerSuite))
}

type TestMediator struct {
	StdOutData []byte
	StdErrData []byte
	Timestamps []int64
}

func (t *TestMediator) WriteLog(msg *Message) error {
	if msg.Stream == "stdout" {
		t.StdOutData = append(t.StdOutData, msg.Data...)
	} else if msg.Stream == "stderr" {
		t.StdErrData = append(t.StdErrData, msg.Data...)
	}

	t.Timestamps = append(t.Timestamps, msg.Timestamp)
	fmt.Println(t.Timestamps)
	return nil
}

func (t *TestMediator) ReadLogs() chan *Message {
	return nil
}

func (t *TestMediator) WatchLogs() chan *Message {
	return nil
}

func (s *MultiReaderLoggerSuite) TestMultiReaderLogger() {

	type testcase struct {
		name             string
		readers          map[string]io.Reader
		expected_strings map[string]string
		timestampCount   int
	}

	tests := []testcase{
		{
			name: "Both stdout and stderr",
			readers: map[string]io.Reader{
				"stdout": bytes.NewBuffer([]byte("Hello\n")),
				"stderr": bytes.NewBuffer([]byte("World\n")),
			},
			expected_strings: map[string]string{
				"stdout": "Hello\n",
				"stderr": "World\n",
			},
			timestampCount: 2,
		},
		{
			name: "Just stdout",
			readers: map[string]io.Reader{
				"stdout": bytes.NewBuffer([]byte("Hello\n")),
			},
			expected_strings: map[string]string{
				"stdout": "Hello\n",
			},
			timestampCount: 1,
		},
		{
			name:             "No data",
			readers:          map[string]io.Reader{},
			expected_strings: map[string]string{},
			timestampCount:   0,
		},
	}

	for _, test := range tests {

		s.Run(test.name, func() {
			t := TestMediator{}
			m := NewMultiReaderLogger(test.readers, &t)
			m.Copy()

			require.Equal(s.T(), test.timestampCount, len(t.Timestamps))

			stdOut := string(t.StdOutData)
			require.Contains(s.T(), stdOut, string(test.expected_strings["stdout"]))

			stdErr := string(t.StdErrData)
			require.Contains(s.T(), stdErr, string(test.expected_strings["stderr"]))

		})
	}
}
