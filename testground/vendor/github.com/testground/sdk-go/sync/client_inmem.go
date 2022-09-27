package sync

import (
	"context"
	"reflect"
	"sync"

	"github.com/testground/sdk-go/runtime"
)

type inmemClient struct {
	sync.Mutex
	*sugarOperations

	states        map[State]int
	barriers      map[State][]*Barrier
	subscriptions map[string][]reflect.Value
	published     map[string][]interface{}
}

// NewInmemClient creates an in-memory sync client for testing.
func NewInmemClient() *inmemClient {
	c := &inmemClient{
		states:        make(map[State]int),
		barriers:      make(map[State][]*Barrier),
		subscriptions: make(map[string][]reflect.Value),
		published:     make(map[string][]interface{}),
	}
	c.sugarOperations = &sugarOperations{c}
	return c
}

// Elemental operations
// ====================

func (i *inmemClient) Publish(_ context.Context, topic *Topic, payload interface{}) (seq int64, err error) {
	i.Lock()
	defer i.Unlock()

	p, ok := i.published[topic.name]
	if !ok {
		p = make([]interface{}, 0, 10)
	}
	p = append(p, payload)
	i.published[topic.name] = p

	for _, ch := range i.subscriptions[topic.name] {
		ch.Send(reflect.ValueOf(payload))
	}

	return int64(len(p)), nil
}

func (i *inmemClient) Subscribe(_ context.Context, topic *Topic, ch interface{}) (*Subscription, error) {
	i.Lock()
	defer i.Unlock()

	s, ok := i.subscriptions[topic.name]
	if !ok {
		s = make([]reflect.Value, 0, 10)
	}
	chV := reflect.ValueOf(ch)
	s = append(s, chV)
	i.subscriptions[topic.name] = s

	// replay any payloads for this topic.
	for _, p := range i.published[topic.name] {
		chV.Send(reflect.ValueOf(p))
	}

	return &Subscription{}, nil
}

func (i *inmemClient) Barrier(_ context.Context, state State, target int) (*Barrier, error) {
	b := &Barrier{
		C:      make(chan error, 1),
		target: int64(target),
	}

	i.Lock()
	defer i.Unlock()

	if i.states[state] == target {
		b.C <- nil
		close(b.C)
		return b, nil
	}

	barriers, ok := i.barriers[state]
	if !ok {
		barriers = make([]*Barrier, 0, 10)
	}
	barriers = append(barriers, b)
	i.barriers[state] = barriers

	return b, nil
}

func (i *inmemClient) SignalEntry(_ context.Context, state State) (after int64, err error) {
	i.Lock()
	defer i.Unlock()

	i.states[state]++

	v := int64(i.states[state])

	var idx int
	for _, b := range i.barriers[state] {
		if v == b.target {
			b.C <- nil
			close(b.C)
			continue
		}
		i.barriers[state][idx] = b
		idx++
	}

	i.barriers[state] = i.barriers[state][:idx]

	return v, nil
}

func (i *inmemClient) SignalEvent(_ context.Context, event *runtime.Event) error {
	return nil
}

func (i *inmemClient) Close() error {
	return nil
}
