package distributed

import (
	"context"
	"encoding/json"
	"fmt"

	"sync"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/objectstore/index"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
	client "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/server/v3/embed"
)

const (
	DefaultStartWithTime = 10 * time.Second
)

type DistributedObjectStore struct {
	ctx            context.Context
	callbacks      *index.CallbackHooks
	cli            *client.Client
	cm             *system.CleanupManager
	dataDir        string
	db             *embed.Etcd
	wg             sync.WaitGroup
	startedChannel chan struct{}
	closeChannel   chan struct{}
	closed         bool
}

func New(options ...Option) (*DistributedObjectStore, error) {
	store := &DistributedObjectStore{
		ctx:            context.Background(),
		cm:             system.NewCleanupManager(),
		callbacks:      index.NewCallbackHooks(),
		startedChannel: make(chan struct{}),
		closeChannel:   make(chan struct{}),
		closed:         true, // cannot claim to be open until DB exists
	}

	for _, opt := range options {
		opt(store)
	}

	store.wg.Add(1)
	go store.startLocalInstance(store.ctx)

	log.Ctx(store.ctx).Debug().Msg("waiting for etcd to start")
	<-store.startedChannel
	log.Ctx(store.ctx).Debug().Msg("etcd has started")

	var err error
	store.cli, err = client.New(client.Config{
		Endpoints:            store.db.Server.Cfg.ClientURLs.StringSlice(),
		DialTimeout:          5 * time.Second,
		DialKeepAliveTimeout: 5 * time.Second,
	})
	if err != nil {
		panic("Errk")
	}

	return store, nil
}

func (d *DistributedObjectStore) CallbackHooks() *index.CallbackHooks {
	return d.callbacks
}

func (d *DistributedObjectStore) Delete(ctx context.Context, prefix string, key string, object any) error {
	return nil
}

func (d *DistributedObjectStore) GetBatch(ctx context.Context, prefix string, keys []string, objects any) (bool, error) {
	return false, nil
}

func (d *DistributedObjectStore) Get(ctx context.Context, prefix string, key string, object any) (bool, error) {
	p := prefixKey(prefix, key)
	response, err := d.cli.Get(ctx, p)
	if err != nil {
		// switch err {
		// case context.Canceled:
		// 	log.Fatalf("ctx is canceled by another routine: %v", err)
		// case context.DeadlineExceeded:
		// 	log.Fatalf("ctx is attached with a deadline is exceeded: %v", err)
		// case rpctypes.ErrEmptyKey:
		// 	log.Fatalf("client-side error: %v", err)
		// default:
		// 	log.Fatalf("bad cluster endpoints, which are not etcd servers: %v", err)
		// }
		return false, err
	}

	// Not found!
	if response.Count == 0 {
		return false, nil
	}

	bytes := response.Kvs[0].Value
	err = json.Unmarshal(bytes, &object)
	return true, err
}

func (d *DistributedObjectStore) Put(ctx context.Context, prefix string, key string, object any) error {
	p := prefixKey(prefix, key)

	// Decompose the object to a byte array for storage
	bytes, err := json.Marshal(&object)
	if err != nil {
		return err
	}

	_, err = d.cli.Put(ctx, p, string(bytes))
	if err != nil {
		return err
	}

	return nil
}

func (d *DistributedObjectStore) Close(ctx context.Context) error {
	// Tell the embedded DB we want to close...
	d.closeChannel <- struct{}{}
	d.wg.Wait() // .. and wait for it to do so

	if d.cli != nil {
		d.cli.Close()
	}

	// Cleanup...
	d.cm.Cleanup(ctx)

	return nil
}

func (d *DistributedObjectStore) startLocalInstance(ctx context.Context) {
	cfg := embed.NewConfig()
	cfg.Dir = d.dataDir
	cfg.LogLevel = "error"

	log.Ctx(ctx).Debug().Str("Dir", cfg.Dir).Msg("etcd data directory configured")

	err := cfg.Validate()
	if err != nil {
		// TODO: Shouldn't get here, we need to validate earlier if the Dir is not
		// valid
		return
	}

	e, err := embed.StartEtcd(cfg)
	if err != nil {
		// TODO: How do we report this without panicking
		log.Ctx(ctx).Error().Err(err).Msg("failed to start embedded database")
		return
	}

	d.db = e
	defer func() {
		<-d.closeChannel

		// Make sure we do this _AFTER_ we've been told to close, not
		// whilst we're waiting.
		d.db.Close()
		d.wg.Done()
	}()

	select {
	case <-d.db.Server.ReadyNotify():
		log.Ctx(ctx).Info().Msg("embedded etcd server ready")
	case <-time.After(DefaultStartWithTime):
		log.Ctx(ctx).Error().Msg("timeout waiting for etcd start")
		d.db.Server.Stop() // trigger a shutdown
		d.closed = true
		return
	}

	// Let the store know we've started
	d.startedChannel <- struct{}{}
	d.closed = false

	log.Ctx(ctx).Info().Msg("waiting for close message")
}

func prefixKey(prefix, key string) string {
	return fmt.Sprintf("%s/%s", prefix, key)
}
