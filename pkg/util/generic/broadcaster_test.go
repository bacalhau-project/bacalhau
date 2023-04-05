//go:build unit || !integration

package generic_test

import (
	"testing"

	_ "github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/util/generic"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type BroadcasterTestSuite struct {
	suite.Suite
	broadcaster *generic.Broadcaster[string]
}

func TestBroadcasterTestSuite(t *testing.T) {
	suite.Run(t, new(BroadcasterTestSuite))
}

func (s *BroadcasterTestSuite) SetupTest() {
	s.broadcaster = generic.NewBroadcaster[string](3)
}

func (s *BroadcasterTestSuite) TestBroadcasterSimple() {
	channels := make([]chan string, 3)
	for i := 0; i < 3; i++ {
		channels[i], _ = s.broadcaster.Subscribe()
	}

	s.broadcaster.Broadcast("Hello")

	require.Equal(s.T(), "Hello", <-channels[0])
	require.Equal(s.T(), "Hello", <-channels[1])
	require.Equal(s.T(), "Hello", <-channels[2])
}

func (s *BroadcasterTestSuite) TestBroadcasterAutoclose() {
	s.broadcaster.SetAutoclose(true)

	ch, err := s.broadcaster.Subscribe()
	require.NoError(s.T(), err)

	s.broadcaster.Unsubscribe(ch)
	require.Equal(s.T(), true, s.broadcaster.IsClosed())

	err = s.broadcaster.Broadcast("err?")
	require.Error(s.T(), err)
}

func (s *BroadcasterTestSuite) TestBroadcasterSubUnsub() {
	ch1, err1 := s.broadcaster.Subscribe()
	ch2, err2 := s.broadcaster.Subscribe()
	require.NoError(s.T(), err1)
	require.NoError(s.T(), err2)

	err := s.broadcaster.Broadcast("err?")
	require.NoError(s.T(), err)
	require.Equal(s.T(), "err?", <-ch1)
	require.Equal(s.T(), "err?", <-ch2)

	// Close the channel without unsubscribing it
	close(ch2)

	s.broadcaster.Broadcast("hello")
	require.Equal(s.T(), "hello", <-ch1)

}
