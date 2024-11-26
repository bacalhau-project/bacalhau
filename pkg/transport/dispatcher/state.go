package dispatcher

import (
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/lib/envelope"
	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
)

// dispatcherState manages all state tracking for the dispatcher including
// sequence numbers, pending messages, and recovery state
type dispatcherState struct {
	mu sync.RWMutex
	// Sequence tracking
	lastAckedSeqNum uint64
	lastObservedSeq uint64
	lastCheckpoint  uint64

	// Pending message management
	pending *pendingMessageStore
}

func newDispatcherState() *dispatcherState {
	return &dispatcherState{
		pending: newPendingMessageStore(),
	}
}

// Sequence number methods
func (s *dispatcherState) updateLastAcked(seqNum uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastAckedSeqNum = max(seqNum, s.lastAckedSeqNum)
}

func (s *dispatcherState) updateLastObserved(seqNum uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastObservedSeq = seqNum
}

// getCheckpointSeqNum returns the sequence number to use for the next checkpoint
func (s *dispatcherState) getCheckpointSeqNum() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var checkpointTarget uint64
	if s.pending.Size() == 0 {
		// If no pending messages, we can checkpoint up to lastObserved
		checkpointTarget = s.lastObservedSeq
	} else {
		checkpointTarget = s.lastAckedSeqNum
	}

	log.Debug().Uint64("lastCheckpoint", s.lastCheckpoint).
		Uint64("checkpointTarget", checkpointTarget).
		Uint64("lastObserved", s.lastObservedSeq).
		Uint64("lastAcked", s.lastAckedSeqNum).
		Int("pending", s.pending.Size()).Msg("getCheckpointSeqNum")

	if checkpointTarget > s.lastCheckpoint {
		return checkpointTarget
	}

	// If no new checkpoint is needed,
	// return 0 to indicate no checkpoint should be saved
	return 0
}

func (s *dispatcherState) updateLastCheckpoint(seqNum uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastCheckpoint = seqNum
}

func (s *dispatcherState) reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastAckedSeqNum = 0
	s.lastObservedSeq = 0
	s.lastCheckpoint = 0
	s.pending.Clear()
}

// Pending message management
type pendingMessage struct {
	eventSeqNum uint64
	publishTime time.Time
	future      ncl.PubFuture
}

// MarshalZerologObject implement LogObjectMarshaler for pendingMessage
func (pm *pendingMessage) MarshalZerologObject(e *zerolog.Event) {
	e.Uint64("eventSeq", pm.eventSeqNum).
		Str("subject", pm.future.Msg().Subject).
		Str("messageType", pm.future.Msg().Header.Get(envelope.KeyMessageType))
}

type pendingMessageStore struct {
	mu   sync.RWMutex
	msgs []*pendingMessage
}

func newPendingMessageStore() *pendingMessageStore {
	return &pendingMessageStore{
		msgs: make([]*pendingMessage, 0),
	}
}

func (s *pendingMessageStore) Add(msg *pendingMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.msgs = append(s.msgs, msg)
}

func (s *pendingMessageStore) RemoveUpTo(seqNum uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	i := 0
	for ; i < len(s.msgs); i++ {
		if s.msgs[i].eventSeqNum > seqNum {
			break
		}
	}
	if i > 0 {
		s.msgs = s.msgs[i:]
	}
}

func (s *pendingMessageStore) GetAll() []*pendingMessage {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*pendingMessage, len(s.msgs))
	copy(result, s.msgs)
	return result
}

func (s *pendingMessageStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.msgs = make([]*pendingMessage, 0)
}

func (s *pendingMessageStore) Size() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.msgs)
}
