package cache

type Cache[T any] interface {
	Get(key string) (T, bool)
	Set(key string, value T, cost uint64, expiresInSeconds int64) error
	SetWithDefaultTTL(key string, value T, cost uint64) error
	Delete(key string)
	Close()
}
