package nclprotocol

import (
	"context"
	"sync/atomic"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/lib/envelope"
	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
)

// SequenceTracker tracks the last successfully processed message sequence number.
// Used by connection managers to checkpoint progress and resume message processing
// after restarts. Thread-safe through atomic operations.
type SequenceTracker struct {
	lastSeqNum atomic.Uint64
}

// NewSequenceTracker creates a new sequence tracker starting at sequence 0
func NewSequenceTracker() *SequenceTracker {
	return &SequenceTracker{}
}

// WithLastSeqNum sets the initial sequence number for resuming processing
func (s *SequenceTracker) WithLastSeqNum(seqNum uint64) *SequenceTracker {
	s.lastSeqNum.Store(seqNum)
	return s
}

// UpdateLastSeqNum updates the latest processed sequence number atomically
func (s *SequenceTracker) UpdateLastSeqNum(seqNum uint64) {
	s.lastSeqNum.Store(seqNum)
}

// GetLastSeqNum returns the last processed sequence number atomically
func (s *SequenceTracker) GetLastSeqNum() uint64 {
	return s.lastSeqNum.Load()
}

// OnProcessed implements ncl.ProcessingNotifier to track message sequence numbers.
// Called after each successful message processing operation.
func (s *SequenceTracker) OnProcessed(ctx context.Context, message *envelope.Message) {
	if message.Metadata.Has(KeySeqNum) {
		s.UpdateLastSeqNum(message.Metadata.GetUint64(KeySeqNum))
	} else {
		log.Trace().Msgf("No sequence number found in message metadata %v", message.Metadata)
	}
}

// Ensure SequenceTracker implements ProcessingNotifier
var _ ncl.ProcessingNotifier = &SequenceTracker{}
