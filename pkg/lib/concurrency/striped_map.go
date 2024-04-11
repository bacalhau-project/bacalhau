package concurrency

import (
	"hash/crc32"
	"sync"
	"sync/atomic"
)

const (
	defaultStripeCount int = 16
)

type StripedMap[T any] struct {
	stripeCount int
	maps        []map[string]T
	counts      []atomic.Int32
	locks       []sync.RWMutex
	total       atomic.Int32
}

func NewStripedMap[T any](numStripes int) *StripedMap[T] {
	count := numStripes
	if count <= 0 {
		count = defaultStripeCount
	}

	s := &StripedMap[T]{
		stripeCount: count,
	}

	for i := 0; i < count; i++ {
		s.maps = append(s.maps, make(map[string]T))
		s.locks = append(s.locks, sync.RWMutex{})
		s.counts = append(s.counts, atomic.Int32{})
	}

	return s
}

func (s *StripedMap[T]) Put(key string, value T) {
	idx := s.hash(key)

	_, found := s.Get(key)

	s.locks[idx].Lock()
	s.maps[idx][key] = value

	// Only increment counters if we are not updating an existing key.
	if !found {
		s.counts[idx].Add(1)
		s.total.Add(1)
	}
	s.locks[idx].Unlock()
}

func (s *StripedMap[T]) Get(key string) (T, bool) {
	idx := s.hash(key)
	s.locks[idx].RLock()
	defer s.locks[idx].RUnlock()

	v, ok := s.maps[idx][key]
	return v, ok
}

func (s *StripedMap[T]) Delete(key string) {
	idx := s.hash(key)
	_, found := s.Get(key)
	if !found {
		// Return early if the key does not exist.
		return
	}

	s.locks[idx].Lock()
	defer s.locks[idx].Unlock()

	s.counts[idx].Add(-1)
	s.total.Add(-1)
	delete(s.maps[idx], key)
}

func (s *StripedMap[T]) Len() int {
	return int(s.total.Load())
}

func (s *StripedMap[T]) LengthsPerStripe() map[int]int {
	m := make(map[int]int)

	for i := 0; i < s.stripeCount; i++ {
		s.locks[i].RLock()
		defer s.locks[i].RUnlock()

		m[i] = int(s.counts[i].Load())
	}

	return m
}

func (s *StripedMap[T]) hash(key string) int {
	hashSum := crc32.ChecksumIEEE([]byte(key))
	return int(hashSum) % s.stripeCount
}
