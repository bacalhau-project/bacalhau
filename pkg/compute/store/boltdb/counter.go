package boltdb

import (
	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"go.uber.org/atomic"
)

type StateCounter struct {
	data map[store.ExecutionState]*atomic.Uint64
}

func NewStateCounter() *StateCounter {
	return &StateCounter{
		data: make(map[store.ExecutionState]*atomic.Uint64),
	}
}

func (s *StateCounter) IncrementState(key store.ExecutionState, amount uint64) {
	counter, ok := s.data[key]
	if !ok {
		s.data[key] = atomic.NewUint64(amount)
	} else {
		counter.Add(amount)
	}
}

func (s *StateCounter) DecrementState(key store.ExecutionState, amount uint64) {
	counter, ok := s.data[key]
	if !ok {
		// We shouldn't get to the point where we have a missing state that
		// we want to decrement, but to be defensive we will add it
		s.data[key] = atomic.NewUint64(0)
	} else {
		counter.Sub(amount)
	}
}

func (s *StateCounter) Include(other *StateCounter) {
	for k, v := range other.data {
		s.IncrementState(k, v.Load())
	}
}

func (s *StateCounter) Get(key store.ExecutionState) uint64 {
	v, ok := s.data[key]
	if !ok {
		return 0
	}
	return v.Load()
}
