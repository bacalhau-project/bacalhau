//go:build unit || !integration

package cache_test

import (
	"math"
	"sync"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/cache"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type CounterSuite struct {
	suite.Suite
}

func TestCounterSuite(t *testing.T) {
	suite.Run(t, new(CounterSuite))
}

func (s *CounterSuite) TestSimpleAdd() {
	c := cache.NewCounter(100)
	c.Inc(1)
	require.Equal(s.T(), uint64(1), c.Current())

	c.Inc(10)
	require.Equal(s.T(), uint64(11), c.Current())

	c.Dec(10)
	require.Equal(s.T(), uint64(1), c.Current())

	require.Equal(s.T(), true, c.HasSpaceFor(99))
	require.Equal(s.T(), false, c.HasSpaceFor(100))

	c.Inc(99)
	require.Equal(s.T(), true, c.IsFull())
}

func BenchmarkCounter(b *testing.B) {
	b.StopTimer()

	c := cache.NewCounter(math.MaxInt64)

	var start, end sync.WaitGroup
	start.Add(1)
	end.Add(100)

	n := b.N / 100

	for i := 0; i < 100; i++ {
		go func() {
			start.Wait()
			for p := 0; p < n; p++ {
				c.Inc(10)
				c.Dec(9)
			}
			end.Done()
		}()
	}

	b.StartTimer()

	// Start the test
	start.Done()

	// wait for all routines to finish
	end.Wait()

	// Each of the 100 routines run will result in a counter with
	// current == 1 (total 100) total of n times.
	require.Equal(b, uint64(n*100), c.Current())
}
