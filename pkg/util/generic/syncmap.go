package generic

import (
	"fmt"
	"strings"
	"sync"
)

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

func (m *SyncMap[K, V]) Keys() []K {
	var keys []K
	m.Iter(func(key K, value V) bool {
		keys = append(keys, key)
		return true
	})
	return keys
}

func (m *SyncMap[K, V]) String() string {
	var sb strings.Builder
	sb.Write([]byte(`{`))
	m.Range(func(key, value any) bool {
		sb.Write([]byte(fmt.Sprintf(`%s=%s`, key, value)))
		return true
	})
	sb.Write([]byte(`}`))
	return sb.String()
}
