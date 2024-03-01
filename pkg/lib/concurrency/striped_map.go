package concurrency

import (
	"hash/crc32"
	"sync"
)

const (
	defaultStripeCount int = 16
)

type StripedMap[T any] struct {
	stripeCount int
	maps        []map[string]T
	locks       []sync.RWMutex
}

func NewStripedMap[T any](numStripes int) *StripedMap[T] {
	count := numStripes
	if count == 0 {
		count = defaultStripeCount
	}

	s := &StripedMap[T]{
		stripeCount: count,
	}

	for i := 0; i < count; i++ {
		s.maps = append(s.maps, make(map[string]T))
		s.locks = append(s.locks, sync.RWMutex{})
	}

	return s
}

func (s *StripedMap[T]) Put(key string, value T) {
	idx := s.hash(key)

	s.locks[idx].Lock()
	s.maps[idx][key] = value
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

	s.locks[idx].Lock()
	defer s.locks[idx].Unlock()

	delete(s.maps[idx], key)
}

func (s *StripedMap[T]) Len() int {
	count := 0

	for i := 0; i < s.stripeCount; i++ {
		s.locks[i].RLock()
		defer s.locks[i].RUnlock()
		count += len(s.maps[i])
	}

	return count
}

func (s *StripedMap[T]) LengthsPerStripe() map[int]int {
	m := make(map[int]int)

	for i := 0; i < s.stripeCount; i++ {
		s.locks[i].RLock()
		defer s.locks[i].RUnlock()

		m[i] = len(s.maps[i])
	}

	return m
}

func (s *StripedMap[T]) hash(key string) int {
	hashSum := crc32.ChecksumIEEE([]byte(key))
	return int(hashSum) % s.stripeCount
}
