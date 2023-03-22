//go:build unit || !integration

package wasmlogs

import (
	"context"
	"log"
	"os"
	"runtime"
	"testing"
	"time"

	_ "github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/test/scenario"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type LogMediatorTestSuite struct {
	scenario.ScenarioRunner
}

func TestLogMediatorTestSuite(t *testing.T) {
	suite.Run(t, new(LogMediatorTestSuite))
}

func (s *LogMediatorTestSuite) TestSimpleWrite() {
	ctx := context.Background()
	tmpFile, err := os.CreateTemp("", "logmediatortest")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(tmpFile.Name()) // clean up

	m, err := NewLogFileMediator(ctx, tmpFile.Name())
	require.NotNil(s.T(), m)
	require.NoError(s.T(), err)

	data := []byte("1234567890")

	msg := NewMessage("stdout", data)
	err = m.WriteLog(msg)

	runtime.Gosched()
	time.Sleep(time.Duration(50) * time.Millisecond)

	ch := m.ReadLogs()
	newMsg, more := <-ch
	require.Equal(s.T(), true, more)
	require.Equal(s.T(), msg.Data, newMsg.Data)
	require.Equal(s.T(), msg.Timestamp, newMsg.Timestamp)

	newMsg, more = <-ch
	require.Equal(s.T(), false, more)
	require.Nil(s.T(), newMsg)

	_ = m.Close()
}

func (s *LogMediatorTestSuite) TestWatchLogs() {
	ctx := context.Background()
	tmpFile, err := os.CreateTemp("", "logmediatortest")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(tmpFile.Name()) // clean up

	m, err := NewLogFileMediator(ctx, tmpFile.Name())
	require.NotNil(s.T(), m)
	require.NoError(s.T(), err)

	data := []byte("1234567890")
	msg := NewMessage("stdout", data)

	chClose := make(chan bool, 1)

	go func() {

		queue := m.WatchLogs()
		defer func() { chClose <- true }()

		watchMsg := queue.Dequeue()
		require.Equal(s.T(), msg.Data, watchMsg.Data)

		watchMsg = queue.Dequeue()
		require.Equal(s.T(), msg.Data, watchMsg.Data)

		watchMsg = queue.Dequeue()
		require.Equal(s.T(), msg.Data, watchMsg.Data)

		require.Equal(s.T(), true, queue.Count() == 0, queue.Count())
		require.Equal(s.T(), queue.IsEmpty(), true)
	}()

	err = m.WriteLog(msg)
	require.NoError(s.T(), err)

	err = m.WriteLog(msg)
	require.NoError(s.T(), err)

	err = m.WriteLog(msg)
	require.NoError(s.T(), err)

	<-chClose
	_ = m.Close()
}
