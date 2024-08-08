//go:build unit || !integration

package watcher

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type EventIteratorTestSuite struct {
	suite.Suite
}

func (suite *EventIteratorTestSuite) TestTrimHorizonIterator() {
	iterator := TrimHorizonIterator()
	suite.Equal(EventIteratorTrimHorizon, iterator.Type)
	suite.Equal(uint64(0), iterator.SequenceNumber)
	suite.Equal("trim_horizon", iterator.String())
}

func (suite *EventIteratorTestSuite) TestLatestIterator() {
	iterator := LatestIterator()
	suite.Equal(EventIteratorLatest, iterator.Type)
	suite.Equal(uint64(0), iterator.SequenceNumber)
	suite.Equal("latest", iterator.String())
}

func (suite *EventIteratorTestSuite) TestAtSequenceNumberIterator() {
	seqNum := uint64(100)
	iterator := AtSequenceNumberIterator(seqNum)
	suite.Equal(EventIteratorAtSequenceNumber, iterator.Type)
	suite.Equal(seqNum, iterator.SequenceNumber)
	suite.Equal("at_sequence_number(100)", iterator.String())
}

func (suite *EventIteratorTestSuite) TestAfterSequenceNumberIterator() {
	seqNum := uint64(200)
	iterator := AfterSequenceNumberIterator(seqNum)
	suite.Equal(EventIteratorAfterSequenceNumber, iterator.Type)
	suite.Equal(seqNum, iterator.SequenceNumber)
	suite.Equal("after_sequence_number(200)", iterator.String())
}

func (suite *EventIteratorTestSuite) TestEventIteratorString() {
	testCases := []struct {
		name     string
		iterator EventIterator
		expected string
	}{
		{"TrimHorizon", TrimHorizonIterator(), "trim_horizon"},
		{"Latest", LatestIterator(), "latest"},
		{"AtSequenceNumber", AtSequenceNumberIterator(300), "at_sequence_number(300)"},
		{"AfterSequenceNumber", AfterSequenceNumberIterator(400), "after_sequence_number(400)"},
		{"Unknown", EventIterator{Type: 99}, "unknown"},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.Equal(tc.expected, tc.iterator.String())
		})
	}
}

func TestEventIteratorSuite(t *testing.T) {
	suite.Run(t, new(EventIteratorTestSuite))
}
