package test

import (
	"fmt"

	"go.uber.org/mock/gomock"

	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
)

type eventSeqNumMatcher struct {
	expectedSeqNum uint64
}

func (m eventSeqNumMatcher) Matches(x interface{}) bool {
	event, ok := x.(watcher.Event)
	if !ok {
		return false
	}
	return event.SeqNum == m.expectedSeqNum
}

func (m eventSeqNumMatcher) String() string {
	return fmt.Sprintf("is event with SeqNum %d", m.expectedSeqNum)
}

// EventWithSeqNum is a matcher that matches an event with a specific sequence number
func EventWithSeqNum(seqNum uint64) gomock.Matcher {
	return eventSeqNumMatcher{expectedSeqNum: seqNum}
}
