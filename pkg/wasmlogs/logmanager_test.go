//go:build unit || !integration

package wasmlogs

import (
	"os"
	"sync"
	"testing"
	"time"

	_ "github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/net/context"
)

type LogManagerTestSuite struct {
	suite.Suite
}

func TestLogManagerTestSuite(t *testing.T) {
	suite.Run(t, new(LogManagerTestSuite))
}

func (s *LogManagerTestSuite) TestLogManager() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tempFile, err := os.CreateTemp("", "test-log-mgr-simple.log")
	require.NoError(s.T(), err)

	mgr, err := NewLogManager(ctx, tempFile.Name())
	require.NoError(s.T(), err)
	require.NotNil(s.T(), mgr)

	done := make(chan bool, 1)

	stdout, stderr := mgr.GetWriters()

	stdout.Write([]byte("Hello"))
	stderr.Write([]byte("stderr"))
	stdout.Write([]byte("World"))

	// Want all of the streams to have finished writing...
	time.Sleep(time.Duration(500) * time.Millisecond)

	go func() {
		stdoutRdr, stderrRdr, err := mgr.GetReaders(false)
		require.NoError(s.T(), err)

		data := make([]byte, 2048)

		n, err := stdoutRdr.Read(data)
		require.NoError(s.T(), err)
		require.Equal(s.T(), "Hello", string(data[:n]))

		n, err = stderrRdr.Read(data)
		require.NoError(s.T(), err)
		require.Equal(s.T(), "stderr", string(data[:n]))

		n, err = stdoutRdr.Read(data)
		require.NoError(s.T(), err)
		require.Equal(s.T(), "World", string(data[:n]))

		n, err = stderrRdr.Read(data)
		require.Error(s.T(), err) // expect io.EOF
		require.Equal(s.T(), n, 0)

		done <- true
	}()

	<-done
	mgr.Close()
}

func (s *LogManagerTestSuite) TestLogManagerStream() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tempFile, err := os.CreateTemp("", "test-log-mgr-simple.log")
	require.NoError(s.T(), err)

	mgr, err := NewLogManager(ctx, tempFile.Name())
	require.NoError(s.T(), err)
	require.NotNil(s.T(), mgr)

	var mu sync.Mutex
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		data := make([]byte, 4096)

		mu.Lock()
		stdoutRdr, _, err := mgr.GetReaders(true)
		require.NoError(s.T(), err)
		mu.Unlock()

		// Read before anything was written should result in us having used
		// the livestream
		n, err := stdoutRdr.Read(data)
		require.NoError(s.T(), err)
		require.Equal(s.T(), "Hello", string(data[:n]))

		wg.Done()
	}()

	mu.Lock()
	stdout, stderr := mgr.GetWriters()
	mu.Unlock()

	_, err = stdout.Write([]byte("Hello"))
	require.NoError(s.T(), err)

	_, err = stderr.Write([]byte("stderr"))
	require.NoError(s.T(), err)

	// Give the channels time to communicate
	time.Sleep(time.Duration(500) * time.Millisecond)

	wg.Wait()
	mgr.Close()
}
