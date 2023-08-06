//go:build unit || !integration

package wasmlogs

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	_ "github.com/bacalhau-project/bacalhau/pkg/logger"
)

type LogManagerTestSuite struct {
	suite.Suite
	ctx    context.Context
	cancel context.CancelFunc
	id     string
}

func TestLogManagerTestSuite(t *testing.T) {
	suite.Run(t, new(LogManagerTestSuite))
}

func (s *LogManagerTestSuite) SetupTest() {
	c, cancel := context.WithCancel(context.Background())
	s.ctx = c
	s.cancel = cancel

	id, _ := uuid.NewUUID()
	s.id = id.String()
}

func (s *LogManagerTestSuite) TestLogManagerCancellation() {
	lm, _ := NewLogManager(s.ctx, s.id)
	s.cancel()
	lm.Close()
}

func (s *LogManagerTestSuite) TestLogManagerWrittenToDisk() {
	lm, _ := NewLogManager(s.ctx, s.id)

	done := make(chan bool, 1)
	go func() {
		stdout, stderr := lm.GetWriters()
		stdout.Write([]byte("hello"))
		stderr.Write([]byte("world"))
		done <- true
	}()

	<-done

	f, err := os.OpenFile(lm.file.Name(), os.O_RDWR, 0755)
	s.Require().NoError(err)

	fmt.Println(lm.file.Name())
	time.Sleep(time.Duration(100) * time.Millisecond)

	reader := bufio.NewReader(f)
	ln, err := reader.ReadString('\n')
	s.Require().NoError(err)

	msg := LogMessage{}
	json.Unmarshal([]byte(ln), &msg)
	s.Require().Equal(string(msg.Data), "hello")

	lm.Close()
}
