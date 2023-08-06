//go:build unit || !integration

package generic_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	_ "github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/util/generic"
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

	s.Require().Equal("Hello", <-channels[0])
	s.Require().Equal("Hello", <-channels[1])
	s.Require().Equal("Hello", <-channels[2])
}

func (s *BroadcasterTestSuite) TestBroadcasterAutoclose() {
	s.broadcaster.SetAutoclose(true)

	ch, err := s.broadcaster.Subscribe()
	s.Require().NoError(err)

	s.broadcaster.Unsubscribe(ch)
	s.Require().Equal(true, s.broadcaster.IsClosed())

	err = s.broadcaster.Broadcast("err?")
	s.Require().Error(err)
}

func (s *BroadcasterTestSuite) TestBroadcasterSubUnsub() {
	ch1, err1 := s.broadcaster.Subscribe()
	ch2, err2 := s.broadcaster.Subscribe()
	s.Require().NoError(err1)
	s.Require().NoError(err2)

	err := s.broadcaster.Broadcast("err?")
	s.Require().NoError(err)
	s.Require().Equal("err?", <-ch1)
	s.Require().Equal("err?", <-ch2)

	// Close the channel without unsubscribing it
	close(ch2)

	s.broadcaster.Broadcast("hello")
	s.Require().Equal("hello", <-ch1)

}
