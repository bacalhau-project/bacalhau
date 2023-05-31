//go:build integration || !unit

package compute

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/testing"
)

type DataLocalityBidSuite struct {
	AskForBidSuite
}

func TestDataLocalityBidSuite(t *testing.T) {
	suite.Run(t, new(AskForBidSuite))
}

func (s *DataLocalityBidSuite) SetupTest() {
	s.config.JobSelectionPolicy.RejectStatelessJobs = true
	s.AskForBidSuite.SetupTest()
}

func (s *DataLocalityBidSuite) TestRejectStateless() {
	s.runAskForBidTest(bidResponseTestCase{
		rejected: true,
	})
}

func (s *DataLocalityBidSuite) TestAcceptStateful() {
	s.runAskForBidTest(bidResponseTestCase{
		job: addInput(s.T(), generateJob(s.T()), storagetesting.TestCID1),
	})
}
