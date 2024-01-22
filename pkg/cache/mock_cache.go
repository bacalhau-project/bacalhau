package cache

// MockCache is a simple map-based cache usable
// in tests without using a real cache.
type MockCache[T any] struct {
	inner map[string]MockCacheItem[T]
}

type MockCacheItem[T any] struct {
	Value  T
	Cost   uint64
	Expiry int64
}

func NewMockCache[T any]() MockCache[T] {
	return MockCache[T]{
		inner: make(map[string]MockCacheItem[T]),
	}
}

func (m MockCache[T]) Get(key string) (T, bool) {
	v, ok := m.inner[key]
	if ok {
		return v.Value, ok
	}
	return *new(T), ok
}

func (m MockCache[T]) Set(key string, value T, cost uint64, expiresInSeconds int64) error {
	m.inner[key] = MockCacheItem[T]{
		Value:  value,
		Cost:   cost,
		Expiry: expiresInSeconds,
	}
	return nil
}

func (m MockCache[T]) Delete(key string) {
	delete(m.inner, key)
}

func (m MockCache[T]) Close() {}
