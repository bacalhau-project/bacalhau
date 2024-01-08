package fake

// MockCache is a simple map-based cache usable
// in tests without using a real cache.
type FakeCache[T any] struct {
	inner              map[string]FakeCacheItem[T]
	GetCalls           int
	SetCalls           int
	SuccessfulGetCalls int
	FailedGetCalls     int
}

type FakeCacheItem[T any] struct {
	Value  T
	Cost   uint64
	Expiry int64
}

func NewFakeCache[T any]() *FakeCache[T] {
	return &FakeCache[T]{
		inner: make(map[string]FakeCacheItem[T]),
	}
}

func (m *FakeCache[T]) Get(key string) (T, bool) {
	m.GetCalls += 1
	v, ok := m.inner[key]
	if ok {
		m.SuccessfulGetCalls += 1
		return v.Value, ok
	}
	m.FailedGetCalls += 1
	return *new(T), ok
}

func (m *FakeCache[T]) Set(key string, value T, cost uint64, expiresInSeconds int64) error {
	m.SetCalls += 1
	m.inner[key] = FakeCacheItem[T]{
		Value:  value,
		Cost:   cost,
		Expiry: expiresInSeconds,
	}
	return nil
}

func (m *FakeCache[T]) Delete(key string) {
	delete(m.inner, key)
}

func (m *FakeCache[T]) Close() {}

func (m *FakeCache[T]) ItemCount() int {
	return len(m.inner)
}
