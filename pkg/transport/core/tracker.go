package core

import (
	"context"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/lib/envelope"
	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
	"github.com/bacalhau-project/bacalhau/pkg/transport"
)

type SequenceTracker struct {
	lastSeqNum uint64
}

func NewSequenceTracker() *SequenceTracker {
	return &SequenceTracker{}
}

// WithLastSeqNum sets the last sequence number
func (s *SequenceTracker) WithLastSeqNum(seqNum uint64) *SequenceTracker {
	s.lastSeqNum = seqNum
	return s
}

func (s *SequenceTracker) UpdateLastSeqNum(seqNum uint64) {
	s.lastSeqNum = seqNum
}

func (s *SequenceTracker) GetLastSeqNum() uint64 {
	return s.lastSeqNum
}

func (s *SequenceTracker) OnProcessed(ctx context.Context, message *envelope.Message) {
	if message.Metadata.Has(transport.KeySeqNum) {
		s.UpdateLastSeqNum(message.Metadata.GetUint64(transport.KeySeqNum))
	} else {
		log.Trace().Msgf("No sequence number found in message metadata %v", message.Metadata)
	}
}

// compile-time check for interface implementations of ncl.Checkpointer
var _ ncl.ProcessingNotifier = &SequenceTracker{}
