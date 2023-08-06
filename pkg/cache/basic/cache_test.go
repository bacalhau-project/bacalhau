//go:build unit || !integration

package basic_test

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/cache"
	"github.com/bacalhau-project/bacalhau/pkg/cache/basic"
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

func (s *BasicCacheSuite) TestBasicCache() {
	k := "test"

	c, err := s.createTestCache("TestBasicCache", 2, oneHour, nil)
	s.Require().NoError(err)
	defer c.Close()

	err = c.Set(k, "value", 1, int64(oneSecond.Seconds()))
	s.Require().NoError(err)

	v, found := c.Get(k)
	s.Require().Equal(true, found)
	s.Require().Equal("value", v)
}

func (s *BasicCacheSuite) TestTooCostly() {
	k := "test"

	c, err := s.createTestCache("TestTooCostly", 1, oneHour, nil)
	s.Require().NoError(err)
	defer c.Close()

	err = c.Set(k, "value", 10, int64(oneSecond.Seconds()))
	s.Require().Error(err)
	s.Require().Equal(cache.ErrCacheTooCostly, err)
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
	s.Require().NoError(err)
	defer c.Close()

	err = c.Set(k, "value", 1, int64(oneHour.Seconds()))
	s.Require().NoError(err)

	runtime.Gosched()

	<-evictionComplete
	_, found := c.Get(k)
	s.Require().Equal(false, found)
}
