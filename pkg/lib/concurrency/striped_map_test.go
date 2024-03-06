//go:build unit || !integration

package concurrency_test

import (
	"math/rand"
	"strconv"
	"sync"
	"testing"
	"time"

	cc "github.com/bacalhau-project/bacalhau/pkg/lib/concurrency"
	"github.com/stretchr/testify/suite"
)

type StripedMapSuite struct {
	suite.Suite
}

func TestStripedMapSuite(t *testing.T) {
	suite.Run(t, new(StripedMapSuite))
}

func (s *StripedMapSuite) TestNewStripedMap() {
	m := cc.NewStripedMap[int](0)
	s.Require().NotNil(m)
	s.Require().Equal(16, len(m.LengthsPerStripe()))

	m = cc.NewStripedMap[int](32)
	s.Require().NotNil(m)
	s.Require().Equal(32, len(m.LengthsPerStripe()))
}

func (s *StripedMapSuite) TestBasic() {
	m := cc.NewStripedMap[int](16)
	s.Require().NotNil(m)

	m.Put("a", 1)
	m.Put("b", 2)
	m.Put("c", 3)

	v, ok := m.Get("a")
	s.Require().True(ok)
	s.Require().Equal(1, v)

	v, ok = m.Get("d")
	s.Require().False(ok)
	s.Require().Equal(0, v)

	// This should never change unless we change the
	// hash function that we use to allocate buckets.
	mm := m.LengthsPerStripe()
	s.Require().Equal(1, mm[3])
	s.Require().Equal(1, mm[9])
	s.Require().Equal(1, mm[15])

	s.Require().Equal(3, m.Len())
	m.Delete("a")
	s.Require().Equal(2, m.Len())
}

func (s *StripedMapSuite) TestEdgeCases() {
	m := cc.NewStripedMap[int](16)
	s.Require().NotNil(m)

	m.Put("a", 1)
	m.Put("a", 1)
	m.Put("b", 2)
	m.Put("c", 3)
	m.Put("c", 3)

	v, ok := m.Get("a")
	s.Require().True(ok)
	s.Require().Equal(1, v)

	v, ok = m.Get("d")
	s.Require().False(ok)
	s.Require().Equal(0, v)

	s.Require().Equal(3, m.Len())
	m.Delete("d")
	s.Require().Equal(3, m.Len())
}

func subtest(m *cc.StripedMap[int]) {
	// No longer necessary in Go 1.2 to call Seed
	// rand.Seed(time.Now().UnixNano())

	min, max := 20, 200
	iters := rand.Intn(max-min+1) + min

	min, max = 0, 1000
	for i := 0; i < iters; i++ {
		putVal := rand.Intn(max-min+1) + min
		getVal := rand.Intn(max-min+1) + min

		m.Put(strconv.Itoa(i), putVal)
		_, _ = m.Get(strconv.Itoa(getVal))

		// Occassionally delete some stuff
		pc := rand.Int31n(100)
		if pc > 90 {
			m.Delete(strconv.Itoa(getVal))
		}
	}
}

func (s *StripedMapSuite) TestConcurrent() {
	wg := sync.WaitGroup{}
	m := cc.NewStripedMap[int](16)

	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			subtest(m)
			wg.Done()
		}()
	}

	// We don't want to wait forever, so we'll wait for
	// at max a few seconds

	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()

	select {
	case <-c:
		return
	case <-time.After(3 * time.Second):
		s.Fail("striped map concurrency test took too long to complete, is it blocked?")
	}
}
