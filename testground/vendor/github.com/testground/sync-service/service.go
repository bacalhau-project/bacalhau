package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

type DefaultService struct {
	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc

	barriersMu sync.Mutex
	barriers   map[string]*barrier
	subsMu     sync.Mutex
	subs       map[string]*pubsub
}

func NewDefaultService(ctx context.Context, log *zap.SugaredLogger) (*DefaultService, error) {
	ctx, cancel := context.WithCancel(ctx)

	s := &DefaultService{
		ctx:      ctx,
		cancel:   cancel,
		barriers: map[string]*barrier{},
		subs:     map[string]*pubsub{},
	}

	s.wg.Add(2)
	go s.subscriptionGC()
	go s.barriersGC()

	return s, nil
}

func (s *DefaultService) Publish(ctx context.Context, topic string, payload interface{}) (int, error) {
	log := log.With("topic", topic)
	log.Debugw("publishing item on topic", "payload", payload)

	bytes, err := json.Marshal(payload)
	if err != nil {
		return -1, fmt.Errorf("failed while serializing payload: %w", err)
	}

	log.Debugw("serialized json payload", "json", string(bytes))

	s.subsMu.Lock()
	defer s.subsMu.Unlock()

	ps := s.createSubIfNew(topic)
	seq := ps.publish(string(bytes))

	log.Debugw("successfully published item; sequence number obtained", "seq", seq)
	return seq, nil
}

func (s *DefaultService) Subscribe(ctx context.Context, topic string) (*subscription, error) {
	log := log.With("topic", topic)
	log.Debugw("subscribing to topic")

	s.subsMu.Lock()
	defer s.subsMu.Unlock()

	log.Debugw("creating topic if new")
	ps := s.createSubIfNew(topic)

	log.Debugw("creating subscription")
	sub := ps.subscribe(ctx)
	return sub, nil
}

func (s *DefaultService) createSubIfNew(topic string) *pubsub {
	if _, ok := s.subs[topic]; !ok {
		ctx, cancel := context.WithCancel(s.ctx)
		s.subs[topic] = &pubsub{
			ctx:     ctx,
			cancel:  cancel,
			lastmod: time.Now(),
			msgs:    []string{},
			subs:    []*subscription{},
		}
		go s.subs[topic].worker()
	}

	return s.subs[topic]
}

func (s *DefaultService) Barrier(ctx context.Context, state string, target int) error {
	log := log.With("state", state, "target", target)
	log.Debugw("subscribing to topic")

	if target == 0 {
		log.Warnw("requested a barrier with target zero; satisfying immediately")
		return nil
	}

	s.barriersMu.Lock()
	log.Debugw("creating state if new")
	barrier := s.createBarrierIfNew(state)
	s.barriersMu.Unlock()

	log.Debugw("waiting for barrier")
	err := barrier.wait(ctx, target)

	if err != nil {
		log.Debugw("barrier errored", "err", err)
	} else {
		log.Debugw("barrier satisfied")
	}

	return err
}

func (s *DefaultService) SignalEntry(ctx context.Context, state string) (int, error) {
	log.Debugw("signalling entry to state", "key", state)

	s.barriersMu.Lock()
	barrier := s.createBarrierIfNew(state)
	s.barriersMu.Unlock()

	seq := barrier.inc()
	log.Debugw("new value of state", "key", state, "value", seq)

	return seq, nil
}

func (s *DefaultService) createBarrierIfNew(state string) *barrier {
	if _, ok := s.barriers[state]; !ok {
		s.barriers[state] = &barrier{
			count: 0,
			zcs:   []*zeroCounter{},
		}
	}

	return s.barriers[state]
}

func (s *DefaultService) Close() error {
	s.cancel()
	s.wg.Wait()
	return nil
}

func (s *DefaultService) subscriptionGC() {
	tick := time.NewTicker(10 * time.Minute)
	defer tick.Stop()
	defer s.wg.Done()

	for {
		select {
		case <-tick.C:
			log.Info("subscription gc running")
			s.subsMu.Lock()

			for topic, sub := range s.subs {
				now := time.Now()
				lastmod := sub.lastmod

				// We delete a PubSub topic if all the subscriptions are done
				// and it's been 10 minutes since it was last modified, or if
				// it's been 30 minutes without modifications.
				//
				// The first condition allows for cases where all the test instances
				// are done, while the second one allows for cases where some instances
				// may hang and we need to free the memory.
				cancel := (sub.isDone() && lastmod.Add(10*time.Minute).Before(now)) ||
					(lastmod.Add(30 * time.Minute).Before(now))

				if cancel {
					log.Debugw("subscription will be deleted", "topic", topic)
					sub.close()
					delete(s.subs, topic)
					log.Debugw("subscription deleted", "topic", topic)
				}
			}

			s.subsMu.Unlock()
			log.Info("subscription gc finished running")
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *DefaultService) barriersGC() {
	tick := time.NewTicker(10 * time.Minute)
	defer tick.Stop()
	defer s.wg.Done()

	for {
		select {
		case <-tick.C:
			log.Info("barrier gc running")
			s.barriersMu.Lock()

			for state, barrier := range s.barriers {
				if barrier.isDone() {
					delete(s.barriers, state)
					log.Debugw("barrier deleted", "state", state)
				}
			}

			s.barriersMu.Unlock()
			log.Info("barrier gc finished running")
		case <-s.ctx.Done():
			return
		}
	}
}
