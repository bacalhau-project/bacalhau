//go:build unit || !integration

package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type CacheSuite struct {
	suite.Suite
	ctx context.Context
}

func (s *CacheSuite) SetupSuite() {
	s.ctx = context.Background()
}

func TestCacheSuite(t *testing.T) {
	suite.Run(t, new(CacheSuite))
}

const (
	oneSecond  = time.Duration(1) * time.Second
	halfSecond = time.Duration(500) * time.Millisecond
	oneHour    = oneSecond * 3600
)

func (s *CacheSuite) TestBasicCache() {
	k := "test"

	c, err := GetOrCreateCache[string]("TestBasicCache", NewCacheOptions(2, 2, oneHour))
	require.NoError(s.T(), err)

	err = c.Set(k, "value", 1, time.Duration(2)*time.Second)
	require.NoError(s.T(), err)

	v, found := c.Get(k)
	require.Equal(s.T(), true, found)
	require.Equal(s.T(), "value", v)
}

func (s *CacheSuite) TestTooCostly() {
	k := "test"

	c, err := GetOrCreateCache[string]("TestTooCostly", NewCacheOptions(2, 1, oneHour))
	require.NoError(s.T(), err)

	err = c.Set(k, "value", 10, time.Duration(2)*time.Second)
	require.Error(s.T(), err)
	require.Equal(s.T(), errTooCostly, err)
}

func (s *CacheSuite) TestTooMany() {
	k := "test"

	c, err := GetOrCreateCache[string]("TestTooMany", NewCacheOptions(1, 10, oneHour))
	require.NoError(s.T(), err)

	err = c.Set(k, "value", 1, time.Duration(2)*time.Second)
	require.NoError(s.T(), err)

	err = c.Set(k, "value", 1, time.Duration(2)*time.Second)
	require.Error(s.T(), err)
	require.Equal(s.T(), errTooFull, err)
}

func (s *CacheSuite) TestExpiry() {
	k := "test"

	c, err := GetOrCreateCache[string]("TestExpiry", NewCacheOptions(1, 1, halfSecond))
	require.NoError(s.T(), err)

	err = c.Set(k, "value", 1, time.Duration(1)*time.Second)
	require.NoError(s.T(), err)

	time.Sleep(2 * time.Second)
	_, found := c.Get(k)
	require.Equal(s.T(), false, found)
}
