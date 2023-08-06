//go:build unit || !integration

package util_test

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/util"
)

type EnvTestSuite struct {
	suite.Suite
}

func TestEnvTestSuite(t *testing.T) {
	suite.Run(t, new(EnvTestSuite))
}

func (s *EnvTestSuite) TestBoolEnv() {
	v := util.GetEnvAs[bool]("TEST_BOOL", true, strconv.ParseBool)
	s.Require().True(v)

	v = util.GetEnvAs[bool]("TEST_BOOL", false, strconv.ParseBool)
	s.Require().False(v)

	s.T().Setenv("TEST_BOOL", "1")
	v = util.GetEnvAs[bool]("TEST_BOOL", true, strconv.ParseBool)
	s.Require().True(v)

	s.T().Setenv("TEST_BOOL", "0")
	v = util.GetEnvAs[bool]("TEST_BOOL", true, strconv.ParseBool)
	s.Require().False(v)
}

func (s *EnvTestSuite) TestDurationEnv() {
	v := util.GetEnvAs[time.Duration]("TEST_DUR", time.Minute, time.ParseDuration)
	s.Require().Equal(time.Minute, v)

	s.T().Setenv("TEST_DUR", "1h")
	v = util.GetEnvAs[time.Duration]("TEST_DUR", time.Minute, time.ParseDuration)
	s.Require().Equal(time.Hour, v)
}

func (s *EnvTestSuite) TestIntegerEnv() {
	partial := func(base int) func(string) (int64, error) {
		return func(k string) (int64, error) {
			return strconv.ParseInt(k, base, 0)
		}
	}
	v := util.GetEnvAs[int64]("TEST_INT", 0, partial(10))
	s.Require().Equal(int64(0), v)

	s.T().Setenv("TEST_INT", "100")
	v = util.GetEnvAs[int64]("TEST_INT", 0, partial(10))
	s.Require().Equal(int64(100), v)

}

func (s *EnvTestSuite) TestStringEnv() {
	v := util.GetEnv("TEST_STR", "hello")
	s.Require().Equal("hello", v)

	s.T().Setenv("TEST_STR", "world")
	v = util.GetEnv("TEST_STR", "hello")
	s.Require().Equal("world", v)
}
