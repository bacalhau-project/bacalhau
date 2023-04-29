//go:build unit || !integration

package cache_test

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/cache"
	"github.com/benbjohnson/clock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type CacheSuite struct {
	suite.Suite
	ctx   context.Context
	clock *clock.Mock
}

func (s *CacheSuite) SetupSuite() {
	s.ctx = context.Background()
}

func (s *CacheSuite) SetupTest() {
	s.clock = clock.NewMock()
	s.clock.Set(time.Now())
}

func TestCacheSuite(t *testing.T) {
	suite.Run(t, new(CacheSuite))
}

const (
	oneSecond = 1
	oneHour   = 3600
)

func (s *CacheSuite) createTestCache(
	name string, maxCost uint64, freq clock.Duration,
) (*cache.Cache[string], error) {
	c, err := cache.NewCache[string](
		name,
		cache.NewCacheOptionsWithFactories(
			maxCost, freq, s.clock.Ticker, s.clock.Now,
		),
	)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (s *CacheSuite) TestBasicCache() {
	k := "test"

	c, err := s.createTestCache("TestBasicCache", 2, oneHour)
	require.NoError(s.T(), err)

	err = c.Set(k, "value", 1, oneSecond)
	require.NoError(s.T(), err)

	v, found := c.Get(k)
	require.Equal(s.T(), true, found)
	require.Equal(s.T(), "value", v)
}

func (s *CacheSuite) TestTooCostly() {
	k := "test"

	c, err := s.createTestCache("TestTooCostly", 1, oneHour)
	require.NoError(s.T(), err)

	err = c.Set(k, "value", 10, oneSecond)
	require.Error(s.T(), err)
	require.Equal(s.T(), cache.ErrCacheTooCostly, err)
}

func (s *CacheSuite) TestExpiry() {
	k := "test"

	c, err := s.createTestCache("TestExpiry", 1, oneHour)
	require.NoError(s.T(), err)

	err = c.Set(k, "value", 1, oneHour)
	require.NoError(s.T(), err)

	// After moving the block past expiry (one hour) we would
	// expect it to be removed shortly
	runtime.Gosched()
	s.clock.Add(oneHour * 2)

	runtime.Gosched()

	_, found := c.Get(k)
	require.Equal(s.T(), false, found)
}
