//go:build unit || !integration

package boltdb

import (
	"context"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/stretchr/testify/suite"
	"go.etcd.io/bbolt"

	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	watchertest "github.com/bacalhau-project/bacalhau/pkg/lib/watcher/test"
)

type BoltDBEventStoreTestSuite struct {
	suite.Suite
	db         *bbolt.DB
	store      *EventStore
	serializer *watcher.JSONSerializer
	clock      *clock.Mock
	ctx        context.Context
	cancel     context.CancelFunc
}

func (s *BoltDBEventStoreTestSuite) SetupTest() {
	s.db = watchertest.CreateBoltDB(s.T())
	s.clock = clock.NewMock()
	s.ctx, s.cancel = context.WithTimeout(context.Background(), 2*time.Second)

	s.serializer = watchertest.CreateSerializer(s.T())
	store, err := NewEventStore(s.db,
		WithLongPollingTimeout(100*time.Millisecond),
		WithGCAgeThreshold(2*time.Hour),
		WithGCCadence(3*time.Hour),
		WithClock(s.clock),
		WithEventSerializer(s.serializer),
		WithCacheSize(2), // smaller cache size to trigger eviction and unmarshalling of boltdb events
	)
	s.Require().NoError(err)
	s.store = store
}

func (s *BoltDBEventStoreTestSuite) TearDownTest() {
	s.cancel()
	s.Require().NoError(s.store.Close(s.ctx))
}

func (s *BoltDBEventStoreTestSuite) TestStoreAndRetrieveEvents() {
	s.storeEvents(5, watcher.OperationCreate, "TestObject")

	resp := s.getEvents(watcher.TrimHorizonIterator(), 10, watcher.EventFilter{})
	s.assertEventsResponse(resp, 5, watcher.AfterSequenceNumberIterator(5))

	for i, event := range resp.Events {
		s.assertEventEquals(event, uint64(i+1), watcher.OperationCreate, "TestObject", &watchertest.TestObject{Value: i + 1})
	}
}

func (s *BoltDBEventStoreTestSuite) TestFilterEvents() {
	s.Require().NoError(s.serializer.RegisterType("TypeA", reflect.TypeOf("")))
	s.Require().NoError(s.serializer.RegisterType("TypeB", reflect.TypeOf("")))

	s.Require().NoError(s.store.StoreEvent(s.ctx, watcher.StoreEventRequest{Operation: watcher.OperationCreate, ObjectType: "TypeA", Object: "event1"}))
	s.Require().NoError(s.store.StoreEvent(s.ctx, watcher.StoreEventRequest{Operation: watcher.OperationUpdate, ObjectType: "TypeB", Object: "event2"}))
	s.Require().NoError(s.store.StoreEvent(s.ctx, watcher.StoreEventRequest{Operation: watcher.OperationDelete, ObjectType: "TypeA", Object: "event3"}))

	testCases := []struct {
		name           string
		filter         watcher.EventFilter
		expectedEvents int
		expectedSeqNum []uint64
	}{
		{
			name:           "Filter by object type",
			filter:         watcher.EventFilter{ObjectTypes: []string{"TypeA"}},
			expectedEvents: 2,
			expectedSeqNum: []uint64{1, 3},
		},
		{
			name:           "Filter by operation",
			filter:         watcher.EventFilter{Operations: []watcher.Operation{watcher.OperationCreate, watcher.OperationUpdate}},
			expectedEvents: 2,
			expectedSeqNum: []uint64{1, 2},
		},
		{
			name:           "Filter by both object type and operation",
			filter:         watcher.EventFilter{ObjectTypes: []string{"TypeA"}, Operations: []watcher.Operation{watcher.OperationDelete}},
			expectedEvents: 1,
			expectedSeqNum: []uint64{3},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			resp := s.getEvents(watcher.TrimHorizonIterator(), 10, tc.filter)
			s.assertEventsResponse(resp, tc.expectedEvents, watcher.AfterSequenceNumberIterator(3))
			for i, seqNum := range tc.expectedSeqNum {
				s.Equal(seqNum, resp.Events[i].SeqNum)
			}
		})
	}
}

func (s *BoltDBEventStoreTestSuite) TestIteratorBehaviors() {
	s.storeEvents(5, watcher.OperationCreate, "TestObject")
	s.store.options.longPollingTimeout = 0

	testCases := []struct {
		name           string
		iterator       watcher.EventIterator
		expectedEvents int
		expectedFirst  uint64
		expectedNext   watcher.EventIterator
	}{
		{"TrimHorizon", watcher.TrimHorizonIterator(), 5, 1, watcher.AfterSequenceNumberIterator(5)},
		{"Latest", watcher.LatestIterator(), 0, 0, watcher.AfterSequenceNumberIterator(5)},
		{"AtSequenceNumber(0)", watcher.AtSequenceNumberIterator(0), 5, 1, watcher.AfterSequenceNumberIterator(5)},
		{"AtSequenceNumber(1)", watcher.AtSequenceNumberIterator(1), 5, 1, watcher.AfterSequenceNumberIterator(5)},
		{"AtSequenceNumber(3)", watcher.AtSequenceNumberIterator(3), 3, 3, watcher.AfterSequenceNumberIterator(5)},
		{"AtSequenceNumber(5)", watcher.AtSequenceNumberIterator(5), 1, 5, watcher.AfterSequenceNumberIterator(5)},
		{"AtSequenceNumber(100)", watcher.AtSequenceNumberIterator(100), 0, 0, watcher.AtSequenceNumberIterator(100)},
		{"AfterSequenceNumber(0)", watcher.AfterSequenceNumberIterator(0), 5, 1, watcher.AfterSequenceNumberIterator(5)},
		{"AfterSequenceNumber(3)", watcher.AfterSequenceNumberIterator(3), 2, 4, watcher.AfterSequenceNumberIterator(5)},
		{"AfterSequenceNumber(5)", watcher.AfterSequenceNumberIterator(5), 0, 0, watcher.AfterSequenceNumberIterator(5)},
		{"AfterSequenceNumber(100)", watcher.AfterSequenceNumberIterator(100), 0, 0, watcher.AfterSequenceNumberIterator(100)},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			resp := s.getEvents(tc.iterator, 10, watcher.EventFilter{})
			s.assertEventsResponse(resp, tc.expectedEvents, tc.expectedNext)
			if tc.expectedEvents > 0 {
				s.Equal(tc.expectedFirst, resp.Events[0].SeqNum)
			}
		})
	}
}

func (s *BoltDBEventStoreTestSuite) TestIteratorBehaviorsEmptyStore() {
	s.store.options.longPollingTimeout = 0

	testCases := []struct {
		name         string
		iterator     watcher.EventIterator
		expectedNext watcher.EventIterator
	}{
		{"TrimHorizon", watcher.TrimHorizonIterator(), watcher.AfterSequenceNumberIterator(0)},
		{"Latest", watcher.LatestIterator(), watcher.AfterSequenceNumberIterator(0)},
		{"AtSequenceNumber(0)", watcher.AtSequenceNumberIterator(0), watcher.AtSequenceNumberIterator(0)},
		{"AtSequenceNumber(1)", watcher.AtSequenceNumberIterator(1), watcher.AtSequenceNumberIterator(1)},
		{"AfterSequenceNumber(0)", watcher.AfterSequenceNumberIterator(0), watcher.AfterSequenceNumberIterator(0)},
		{"AfterSequenceNumber(1)", watcher.AfterSequenceNumberIterator(1), watcher.AfterSequenceNumberIterator(1)},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			resp := s.getEvents(tc.iterator, 10, watcher.EventFilter{})
			s.assertEventsResponse(resp, 0, tc.expectedNext)
		})
	}
}

func (s *BoltDBEventStoreTestSuite) TestNotifyOnStore() {
	respCh, errCh := s.getEventsAsync(watcher.AfterSequenceNumberIterator(0), 10, watcher.EventFilter{})

	s.assertChannelsEmpty(respCh, errCh)

	s.Require().NoError(s.store.StoreEvent(s.ctx, watcher.StoreEventRequest{
		Operation:  watcher.OperationCreate,
		ObjectType: "TestObject",
		Object:     watchertest.TestObject{Name: "new event"},
	}))

	resp := s.assertResponseReceived(respCh, errCh)
	s.assertEventsResponse(resp, 1, watcher.AfterSequenceNumberIterator(1))
	s.assertEventEquals(resp.Events[0], 1, watcher.OperationCreate, "TestObject", &watchertest.TestObject{Name: "new event"})
}

func (s *BoltDBEventStoreTestSuite) TestLongPolling() {
	longPollingTimeout := 500 * time.Millisecond
	s.store.options.longPollingTimeout = longPollingTimeout

	respCh, errCh := s.getEventsAsync(watcher.AfterSequenceNumberIterator(0), 10, watcher.EventFilter{})

	s.clock.Add(longPollingTimeout - 10*time.Millisecond)
	s.assertChannelsEmpty(respCh, errCh)

	s.clock.Add(20 * time.Millisecond)
	resp := s.assertResponseReceived(respCh, errCh)
	s.assertEventsResponse(resp, 0, watcher.AfterSequenceNumberIterator(0))
}

func (s *BoltDBEventStoreTestSuite) TestCheckpoints() {
	s.Require().NoError(s.store.StoreCheckpoint(s.ctx, "watcher1", 5))
	s.Require().NoError(s.store.StoreCheckpoint(s.ctx, "watcher2", 10))

	checkpoint, err := s.store.GetCheckpoint(s.ctx, "watcher1")
	s.Require().NoError(err)
	s.Equal(uint64(5), checkpoint)

	checkpoint, err = s.store.GetCheckpoint(s.ctx, "watcher2")
	s.Require().NoError(err)
	s.Equal(uint64(10), checkpoint)

	_, err = s.store.GetCheckpoint(s.ctx, "nonexistent")
	s.Require().Error(err)
	s.ErrorIs(err, watcher.ErrCheckpointNotFound)
}

func (s *BoltDBEventStoreTestSuite) TestGarbageCollection() {
	s.storeEvents(5, watcher.OperationCreate, "TestObject")

	s.clock.Add(2 * time.Hour)
	s.Require().NoError(s.store.StoreEvent(s.ctx, watcher.StoreEventRequest{
		Operation:  watcher.OperationCreate,
		ObjectType: "TestObject",
		Object:     watchertest.TestObject{Name: "recent"},
	}))

	s.Require().NoError(s.store.StoreCheckpoint(s.ctx, "watcher1", 3))
	s.Require().NoError(s.store.StoreCheckpoint(s.ctx, "watcher2", 6))

	s.clock.Add(1*time.Hour + 1*time.Millisecond)

	s.Eventually(func() bool {
		resp := s.getEvents(watcher.TrimHorizonIterator(), 10, watcher.EventFilter{})
		return len(resp.Events) == 3
	}, 300*time.Millisecond, 10*time.Millisecond)

	resp := s.getEvents(watcher.TrimHorizonIterator(), 10, watcher.EventFilter{})
	s.assertEventsResponse(resp, 3, watcher.AfterSequenceNumberIterator(6))
	s.assertEventEquals(resp.Events[0], 4, watcher.OperationCreate, "TestObject", &watchertest.TestObject{Value: 4})
	s.assertEventEquals(resp.Events[1], 5, watcher.OperationCreate, "TestObject", &watchertest.TestObject{Value: 5})
	s.assertEventEquals(resp.Events[2], 6, watcher.OperationCreate, "TestObject", &watchertest.TestObject{Name: "recent"})
}

func (s *BoltDBEventStoreTestSuite) TestConcurrentOperations() {
	var wg sync.WaitGroup
	eventCount := 100

	for i := 0; i < eventCount; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			s.Require().NoError(s.store.StoreEvent(s.ctx, watcher.StoreEventRequest{
				Operation:  watcher.OperationCreate,
				ObjectType: "TestObject",
				Object:     watchertest.TestObject{Value: i + 1},
			}))
		}(i)
	}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.getEvents(watcher.TrimHorizonIterator(), eventCount, watcher.EventFilter{})
		}()
	}

	wg.Wait()

	resp := s.getEvents(watcher.TrimHorizonIterator(), eventCount*2, watcher.EventFilter{})
	s.assertEventsResponse(resp, eventCount, watcher.AfterSequenceNumberIterator(uint64(eventCount)))

	for i, event := range resp.Events {
		s.assertEventEquals(event, uint64(i+1), watcher.OperationCreate, "TestObject", nil)
	}
}

func (s *BoltDBEventStoreTestSuite) TestGetLatestEventNum() {
	for i := 1; i <= 5; i++ {
		s.Require().NoError(s.store.StoreEvent(s.ctx, watcher.StoreEventRequest{
			Operation:  watcher.OperationCreate,
			ObjectType: "TestObject",
			Object:     watchertest.TestObject{Value: i},
		}))

		latestNum, err := s.store.GetLatestEventNum(s.ctx)
		s.Require().NoError(err)
		s.Equal(uint64(i), latestNum)
	}

	latestNum, err := s.store.GetLatestEventNum(s.ctx)
	s.Require().NoError(err)
	s.Equal(uint64(5), latestNum)
}

func (s *BoltDBEventStoreTestSuite) TestZeroLimit() {
	s.storeEvents(2, watcher.OperationCreate, "TestObject")

	resp := s.getEvents(watcher.TrimHorizonIterator(), 10, watcher.EventFilter{})
	s.Require().Len(resp.Events, 2)
	s.assertEventEquals(resp.Events[0], uint64(1), watcher.OperationCreate, "TestObject", &watchertest.TestObject{Value: 1})
	s.assertEventEquals(resp.Events[1], uint64(2), watcher.OperationCreate, "TestObject", &watchertest.TestObject{Value: 2})

	s.Require().Equal(watcher.AfterSequenceNumberIterator(2), resp.NextEventIterator)
}

func (s *BoltDBEventStoreTestSuite) TestWithSingleEvent() {
	s.Require().NoError(s.store.StoreEvent(s.ctx, watcher.StoreEventRequest{
		Operation:  watcher.OperationCreate,
		ObjectType: "TestObject",
		Object:     watchertest.TestObject{Value: 1},
	}))
	s.store.options.longPollingTimeout = 0

	testCases := []struct {
		name           string
		iterator       watcher.EventIterator
		expectedEvents int
		expectedNext   watcher.EventIterator
	}{
		{"TrimHorizon", watcher.TrimHorizonIterator(), 1, watcher.AfterSequenceNumberIterator(1)},
		{"Latest", watcher.LatestIterator(), 0, watcher.AfterSequenceNumberIterator(1)},
		{"AtSequenceNumber(1)", watcher.AtSequenceNumberIterator(1), 1, watcher.AfterSequenceNumberIterator(1)},
		{"AtSequenceNumber(2)", watcher.AtSequenceNumberIterator(2), 0, watcher.AtSequenceNumberIterator(2)},
		{"AfterSequenceNumber(0)", watcher.AfterSequenceNumberIterator(0), 1, watcher.AfterSequenceNumberIterator(1)},
		{"AfterSequenceNumber(1)", watcher.AfterSequenceNumberIterator(1), 0, watcher.AfterSequenceNumberIterator(1)},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			resp := s.getEvents(tc.iterator, 10, watcher.EventFilter{})
			s.Require().Len(resp.Events, tc.expectedEvents)
			s.Equal(tc.expectedNext, resp.NextEventIterator)
		})
	}
}

func (s *BoltDBEventStoreTestSuite) TestIteratorProgression() {
	s.storeEvents(5, watcher.OperationCreate, "TestObject")
	s.store.options.longPollingTimeout = 0

	iterator := watcher.TrimHorizonIterator()
	for i := 1; i <= 5; i++ {
		resp := s.getEvents(iterator, 1, watcher.EventFilter{})
		s.Require().Len(resp.Events, 1)
		s.assertEventEquals(resp.Events[0], uint64(i), watcher.OperationCreate, "TestObject", &watchertest.TestObject{Value: i})
		s.Equal(watcher.AfterSequenceNumberIterator(uint64(i)), resp.NextEventIterator)
		iterator = resp.NextEventIterator
	}

	// Verify that we've reached the end
	resp := s.getEvents(iterator, 1, watcher.EventFilter{})
	s.Require().Empty(resp.Events)
	s.Equal(watcher.AfterSequenceNumberIterator(5), resp.NextEventIterator)
}

func (s *BoltDBEventStoreTestSuite) TestLatestIteratorWithNewEvents() {
	s.storeEvents(3, watcher.OperationCreate, "TestObject")

	// Get the latest event
	resp := s.getEvents(watcher.LatestIterator(), 1, watcher.EventFilter{})
	s.Require().Len(resp.Events, 0)
	s.Equal(watcher.AfterSequenceNumberIterator(3), resp.NextEventIterator)

	// Add two more events
	s.storeEvents(2, watcher.OperationUpdate, "TestObject")

	// Get events after the previous latest
	resp = s.getEvents(resp.NextEventIterator, 10, watcher.EventFilter{})
	s.Require().Len(resp.Events, 2)
	s.assertEventEquals(resp.Events[0], 4, watcher.OperationUpdate, "TestObject", nil)
	s.assertEventEquals(resp.Events[1], 5, watcher.OperationUpdate, "TestObject", nil)
	s.Equal(watcher.AfterSequenceNumberIterator(5), resp.NextEventIterator)
}

func (s *BoltDBEventStoreTestSuite) TestCaching() {
	s.storeEvents(1, watcher.OperationCreate, "TestObject")
	event, found := s.store.cache.Get(1)
	s.Require().True(found)
	s.assertEventEquals(event, 1, watcher.OperationCreate, "TestObject", &watchertest.TestObject{Value: 1})

	resp := s.getEvents(watcher.TrimHorizonIterator(), 10, watcher.EventFilter{})
	s.assertEventsResponse(resp, 1, watcher.AfterSequenceNumberIterator(1))
	s.Equal(event, resp.Events[0])
}

// TestNoCaching test we read directly and unmarshal from boltdb
func (s *BoltDBEventStoreTestSuite) TestNoCaching() {
	s.storeEvents(1, watcher.OperationCreate, "TestObject")
	s.store.cache.Purge()
	_, found := s.store.cache.Get(1)
	s.Require().False(found)

	resp := s.getEvents(watcher.TrimHorizonIterator(), 10, watcher.EventFilter{})
	s.assertEventsResponse(resp, 1, watcher.AfterSequenceNumberIterator(1))
	s.assertEventEquals(resp.Events[0], 1, watcher.OperationCreate, "TestObject", &watchertest.TestObject{Value: 1})
}

func (s *BoltDBEventStoreTestSuite) TestConcurrentSubscribers() {
	// Start 3 concurrent subscribers at different positions
	respCh1, errCh1 := s.getEventsAsync(watcher.AfterSequenceNumberIterator(0), 1, watcher.EventFilter{})
	respCh2, errCh2 := s.getEventsAsync(watcher.AfterSequenceNumberIterator(0), 1, watcher.EventFilter{})
	respCh3, errCh3 := s.getEventsAsync(watcher.AfterSequenceNumberIterator(0), 1, watcher.EventFilter{})

	// Verify all are waiting
	s.assertChannelsEmpty(respCh1, errCh1)
	s.assertChannelsEmpty(respCh2, errCh2)
	s.assertChannelsEmpty(respCh3, errCh3)

	// Store an event
	s.Require().NoError(s.store.StoreEvent(s.ctx, watcher.StoreEventRequest{
		Operation:  watcher.OperationCreate,
		ObjectType: "TestObject",
		Object:     watchertest.TestObject{Value: 1},
	}))

	// All subscribers should get the event
	resp1 := s.assertResponseReceived(respCh1, errCh1)
	resp2 := s.assertResponseReceived(respCh2, errCh2)
	resp3 := s.assertResponseReceived(respCh3, errCh3)

	// Verify all got the same event
	s.assertEventsResponse(resp1, 1, watcher.AfterSequenceNumberIterator(1))
	s.assertEventsResponse(resp2, 1, watcher.AfterSequenceNumberIterator(1))
	s.assertEventsResponse(resp3, 1, watcher.AfterSequenceNumberIterator(1))
}

func (s *BoltDBEventStoreTestSuite) TestLongPollingWithMultipleSubscribers() {
	longPollingTimeout := 100 * time.Millisecond
	s.store.options.longPollingTimeout = longPollingTimeout

	// Start multiple subscribers
	subscriberCount := 5
	respChs := make([]<-chan *watcher.GetEventsResponse, subscriberCount)
	errChs := make([]<-chan error, subscriberCount)

	for i := 0; i < subscriberCount; i++ {
		respCh, errCh := s.getEventsAsync(watcher.AfterSequenceNumberIterator(0), 1, watcher.EventFilter{})
		respChs[i] = respCh
		errChs[i] = errCh
	}

	// Verify all subscribers are waiting (no responses yet)
	for i := 0; i < subscriberCount; i++ {
		s.assertChannelsEmpty(respChs[i], errChs[i])
	}

	// Store an event
	s.Require().NoError(s.store.StoreEvent(s.ctx, watcher.StoreEventRequest{
		Operation:  watcher.OperationCreate,
		ObjectType: "TestObject",
		Object:     watchertest.TestObject{Value: 1},
	}))

	// All subscribers should receive the event
	for i := 0; i < subscriberCount; i++ {
		resp := s.assertResponseReceived(respChs[i], errChs[i])
		s.assertEventsResponse(resp, 1, watcher.AfterSequenceNumberIterator(1))
		s.assertEventEquals(resp.Events[0], 1, watcher.OperationCreate, "TestObject", &watchertest.TestObject{Value: 1})
	}
}

// Helper methods

func (s *BoltDBEventStoreTestSuite) storeEvents(count int, operation watcher.Operation, objectType string) {
	for i := 1; i <= count; i++ {
		err := s.store.StoreEvent(s.ctx, watcher.StoreEventRequest{
			Operation:  operation,
			ObjectType: objectType,
			Object:     watchertest.TestObject{Value: i},
		})
		s.Require().NoError(err)
	}
}

func (s *BoltDBEventStoreTestSuite) getEvents(iterator watcher.EventIterator, limit int, filter watcher.EventFilter) *watcher.GetEventsResponse {
	resp, err := s.store.GetEvents(s.ctx, watcher.GetEventsRequest{
		EventIterator: iterator,
		Limit:         limit,
		Filter:        filter,
	})
	s.Require().NoError(err)
	return resp
}

func (s *BoltDBEventStoreTestSuite) getEventsAsync(iterator watcher.EventIterator, limit int, filter watcher.EventFilter) (<-chan *watcher.GetEventsResponse, <-chan error) {
	respCh := make(chan *watcher.GetEventsResponse, 1)
	errCh := make(chan error, 1)
	go func() {
		resp, err := s.store.GetEvents(s.ctx, watcher.GetEventsRequest{
			EventIterator: iterator,
			Limit:         limit,
			Filter:        filter,
		})
		if err != nil {
			errCh <- err
		} else {
			respCh <- resp
		}
	}()
	time.Sleep(50 * time.Millisecond)
	return respCh, errCh
}

func (s *BoltDBEventStoreTestSuite) assertEventsResponse(resp *watcher.GetEventsResponse, expectedCount int, expectedNext watcher.EventIterator) {
	s.Require().NotNil(resp)
	s.Require().Len(resp.Events, expectedCount)
	s.Equal(expectedNext, resp.NextEventIterator)
}

func (s *BoltDBEventStoreTestSuite) assertEventEquals(event watcher.Event, expectedSeqNum uint64, expectedOperation watcher.Operation, expectedObjectType string, expectedObject *watchertest.TestObject) {
	s.Equal(expectedSeqNum, event.SeqNum, "Unexpected sequence number")
	s.Equal(expectedOperation, event.Operation, "Unexpected operation")
	s.Equal(expectedObjectType, event.ObjectType, "Unexpected object type")
	if expectedObject != nil {
		s.Equal(*expectedObject, event.Object, "Unexpected object")
	}
}

func (s *BoltDBEventStoreTestSuite) assertChannelsEmpty(respCh <-chan *watcher.GetEventsResponse, errCh <-chan error) {
	select {
	case <-respCh:
		s.Fail("Received unexpected response")
	case <-errCh:
		s.Fail("Received unexpected error")
	default:
		// This is the expected case
	}
}

func (s *BoltDBEventStoreTestSuite) assertResponseReceived(respCh <-chan *watcher.GetEventsResponse, errCh <-chan error) *watcher.GetEventsResponse {
	select {
	case resp := <-respCh:
		return resp
	case err := <-errCh:
		s.Fail("Received unexpected error", err)
		return nil
	case <-s.ctx.Done():
		s.Fail("Context timeout while waiting for response")
		return nil
	}
}

func TestBoltDBEventStore(t *testing.T) {
	suite.Run(t, new(BoltDBEventStoreTestSuite))
}
