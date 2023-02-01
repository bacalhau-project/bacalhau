package generic

import "sync"

// A SyncMap is a concurrency-safe sync.Map that uses strongly-typed
// method signatures to ensure the types of its stored data are known.
type SyncMap[K comparable, V any] struct {
	sync.Map
}

func SyncMapFromMap[K comparable, V any](m map[K]V) *SyncMap[K, V] {
	ret := &SyncMap[K, V]{}
	for k, v := range m {
		ret.Put(k, v)
	}

	return ret
}

func (m *SyncMap[K, V]) Get(key K) (V, bool) {
	value, ok := m.Load(key)
	if !ok {
		var empty V
		return empty, false
	}
	return value.(V), true
}

func (m *SyncMap[K, V]) Put(key K, value V) {
	m.Store(key, value)
}

func (m *SyncMap[K, V]) Iter(ranger func(key K, value V) bool) {
	m.Range(func(key, value any) bool {
		k := key.(K)
		v := value.(V)
		return ranger(k, v)
	})
}
