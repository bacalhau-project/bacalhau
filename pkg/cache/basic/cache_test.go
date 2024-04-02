//go:build unit || !integration

package basic_test

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/cache"
	"github.com/bacalhau-project/bacalhau/pkg/cache/basic"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type BasicCacheSuite struct {
	suite.Suite
	ctx context.Context
}

func (s *BasicCacheSuite) SetupSuite() {
	s.ctx = context.Background()
}

func TestBasicCacheSuite(t *testing.T) {
	suite.Run(t, new(BasicCacheSuite))
}

const (
	oneSecond = time.Duration(1) * time.Second
	oneMinute = oneSecond * 60
	oneHour   = oneMinute * 60
)

func (s *BasicCacheSuite) createTestCache(
	name string, maxCost uint64, freq time.Duration, f basic.EvictItemFunc,
) (cache.Cache[string], error) {
	options := []basic.Option{
		basic.WithMaxCost(maxCost),
		basic.WithCleanupFrequency(freq),
	}
	if f != nil {
		options = append(options, basic.WithEvictionFunction(f))
	}

	c, err := basic.NewCache[string](options...)

	if err != nil {
		return nil, err
	}
	return c, nil
}

func (s *BasicCacheSuite) createTestCacheWithDefaultTTL(
	name string, maxCost uint64, freq time.Duration, ttl time.Duration, f basic.EvictItemFunc,
) (cache.Cache[string], error) {
	options := []basic.Option{
		basic.WithMaxCost(maxCost),
		basic.WithCleanupFrequency(freq),
		basic.WithTTL(ttl),
	}
	if f != nil {
		options = append(options, basic.WithEvictionFunction(f))
	}
	c, err := basic.NewCache[string](options...)

	if err != nil {
		return nil, err
	}
	return c, nil
}

func (s *BasicCacheSuite) TestBasicCache() {
	k := "test"

	c, err := s.createTestCache("TestBasicCache", 2, oneHour, nil)
	require.NoError(s.T(), err)
	defer c.Close()

	err = c.Set(k, "value", 1, int64(oneSecond.Seconds()))
	require.NoError(s.T(), err)

	v, found := c.Get(k)
	require.Equal(s.T(), true, found)
	require.Equal(s.T(), "value", v)
}

func (s *BasicCacheSuite) TestTooCostly() {
	k := "test"

	c, err := s.createTestCache("TestTooCostly", 1, oneHour, nil)
	require.NoError(s.T(), err)
	defer c.Close()

	err = c.Set(k, "value", 10, int64(oneSecond.Seconds()))
	require.Error(s.T(), err)
	require.Equal(s.T(), cache.ErrCacheTooCostly, err)
}

func (s *BasicCacheSuite) TestExpiry() {
	k := "test"

	evictionComplete := make(chan struct{}, 1)
	f := func(key string, cost uint64, expiresAt int64, now int64) bool {
		willEvict := expiresAt != 0 && expiresAt <= now
		if key == k && willEvict {
			evictionComplete <- struct{}{}
		}
		return willEvict
	}

	c, err := s.createTestCache("TestExpiry", 1, 1, f)
	require.NoError(s.T(), err)
	defer c.Close()

	err = c.Set(k, "value", 1, 1)
	require.NoError(s.T(), err)

	runtime.Gosched()

	<-evictionComplete
	_, found := c.Get(k)
	require.Equal(s.T(), false, found)
}

func (s *BasicCacheSuite) TestExpiryDefaultTTL() {
	k := "test"

	evictionComplete := make(chan struct{}, 1)
	f := func(key string, cost uint64, expiresAt int64, now int64) bool {
		// We want to test the expiry time of the found item
		if key == k {
			require.Equal(s.T(), now+1, expiresAt)
			evictionComplete <- struct{}{}
			return true
		}
		return false
	}

	c, err := s.createTestCacheWithDefaultTTL("TestExpiry", 1, 1, 1*time.Second, f)
	require.NoError(s.T(), err)
	defer c.Close()

	err = c.SetWithDefaultTTL(k, "value", 1)
	require.NoError(s.T(), err)

	runtime.Gosched()

	<-evictionComplete
	_, found := c.Get(k)
	require.Equal(s.T(), false, found)
}
