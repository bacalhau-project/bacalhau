package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/objectstore/distributed"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
	spb "go.etcd.io/etcd/api/v3/mvccpb"
	etcd_client "go.etcd.io/etcd/client/v3"
)

type QueuePriority = uint8

type Queue[T any] struct {
	client *etcd_client.Client
	prefix string
}

func NewQueue[T any](store *distributed.DistributedObjectStore, prefix string) *Queue[T] {
	checkedPrefix := lo.Ternary(strings.HasSuffix(prefix, "/"), prefix, prefix+"/")

	return &Queue[T]{
		client: store.GetClient(),
		prefix: checkedPrefix,
	}
}

func (q *Queue[T]) Enqueue(ctx context.Context, object T, priority QueuePriority) error {
	dehydrated, err := json.Marshal(&object)
	if err != nil {
		return err
	}

	fullPrefix := fmt.Sprintf("%s%03d", q.prefix, priority)
	key, err := NewSequentialKV(ctx, q.client, fullPrefix, string(dehydrated))
	if err != nil {
		log.Ctx(ctx).Error().
			Err(err).
			Str("Key", key).
			Int("Priority", int(priority)).
			Msg("failed to add new sequential kv")
	}

	log.Ctx(ctx).Debug().Str("Key", key).Int("Priority", int(priority)).Msg("Enqueued item")

	return err
}

// Dequeue will block if nothing is available in a queue, but will
// eventually return a T that was enqueue using enqueue (in FIFO order)
func (q *Queue[T]) Dequeue(ctx context.Context) (T, error) {
	emptyT := func() T { return *new(T) }
	decodeT := func(b []byte) (T, error) {
		t := emptyT()
		err := json.Unmarshal(b, &t)
		return t, err
	}

	// WithFirstKey will get the item that sorts first, so priority one
	// will be used before priority 2.  `key/000/001 before key/000/002`
	response, err := q.client.Get(ctx, q.prefix, etcd_client.WithFirstKey()...)
	if err != nil {
		return emptyT(), err
	}

	kv, err := ClaimFirstKey(ctx, q.client, response.Kvs)
	if err != nil {
		return emptyT(), err
	} else if kv != nil {
		return decodeT(kv.Value)
	} else if response.More {
		// No kv and no error so if there are more, try those instead
		return q.Dequeue(ctx)
	}

	// Wait for something new to be PUT so we can use it
	ev, err := WaitPrefixEvents(
		ctx,
		q.client,
		q.prefix,
		response.Header.Revision,
		spb.PUT)
	if err != nil {
		return emptyT(), err
	}

	ok, err := DeleteRevKey(ctx, q.client, string(ev.Kv.Key), ev.Kv.ModRevision)
	if err != nil {
		return emptyT(), err
	} else if !ok {
		// No error but unable to delete a specific revision, so let's try
		// again with another dequeue from scratch
		return q.Dequeue(ctx)
	}

	return decodeT(ev.Kv.Value)
}

func (q *Queue[T]) Close() {

}
