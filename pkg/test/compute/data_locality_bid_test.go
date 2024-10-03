//go:build integration || !unit

package compute

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
)

// TODO(forrest): [fixme] I don't think this test never actually calls its tests method
// it just reruns the AskForBidSuite.

type DataLocalityBidSuite struct {
	AskForBidSuite
}

func TestDataLocalityBidSuite(t *testing.T) {
	suite.Run(t, new(AskForBidSuite))
}

func (s *DataLocalityBidSuite) SetupTest() {
	s.config.BacalhauConfig.JobAdmissionControl.RejectStatelessJobs = true
	s.AskForBidSuite.SetupTest()
}

func (s *DataLocalityBidSuite) TestRejectStateless() {
	s.runAskForBidTest(bidResponseTestCase{
		rejected: true,
	})
}

func (s *DataLocalityBidSuite) TestAcceptStateful() {
	s.runAskForBidTest(bidResponseTestCase{
		execution: addInput(mock.Execution(), "cid"),
	})
}
