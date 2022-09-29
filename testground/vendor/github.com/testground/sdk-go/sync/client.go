package sync

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/testground/sdk-go/runtime"
	tgsync "github.com/testground/sync-service"
	"go.uber.org/zap"
	"nhooyr.io/websocket"
)

const (
	EnvServiceHost = "SYNC_SERVICE_HOST"
	EnvServicePort = "SYNC_SERVICE_PORT"
)

// ErrNoRunParameters is returned by the generic client when an unbound context
// is passed in. See WithRunParams to bind RunParams to the context.
var ErrNoRunParameters = fmt.Errorf("no run parameters provided")

type DefaultClient struct {
	*sugarOperations

	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	log       *zap.SugaredLogger
	extractor func(ctx context.Context) (rp *runtime.RunParams)

	nextMu     sync.Mutex
	next       int
	handlersMu sync.Mutex
	handlers   map[string]chan *tgsync.Response
	socket     *websocket.Conn
}

// NewBoundClient returns a new sync DefaultClient that is bound to the provided
// RunEnv. All operations will be automatically scoped to the keyspace of that
// run.
//
// The context passed in here will govern the lifecycle of the client.
// Cancelling it will cancel all ongoing operations. However, for a clean
// closure, the user should call Close().
//
// For test plans, a suitable context to pass here is the background context.
func NewBoundClient(ctx context.Context, runenv *runtime.RunEnv) (*DefaultClient, error) {
	log := runenv.SLogger()

	return newClient(ctx, log, func(ctx context.Context) *runtime.RunParams {
		return &runenv.RunParams
	})
}

// MustBoundClient creates a new bound client by calling NewBoundClient, and
// panicking if it errors.
func MustBoundClient(ctx context.Context, runenv *runtime.RunEnv) *DefaultClient {
	c, err := NewBoundClient(ctx, runenv)
	if err != nil {
		panic(err)
	}
	return c
}

// NewGenericClient returns a new sync DefaultClient that is bound to no RunEnv.
// It is intended to be used by testground services like the sidecar.
//
// All operations expect to find the RunParams of the run to scope its actions
// inside the supplied context.Context. Call WithRunParams to bind the
// appropriate RunParams.
//
// The context passed in here will govern the lifecycle of the client.
// Cancelling it will cancel all ongoing operations. However, for a clean
// closure, the user should call Close().
//
// A suitable context to pass here is the background context of the main
// process.
func NewGenericClient(ctx context.Context, log *zap.SugaredLogger) (*DefaultClient, error) {
	return newClient(ctx, log, GetRunParams)
}

// MustGenericClient creates a new generic client by calling NewGenericClient,
// and panicking if it errors.
func MustGenericClient(ctx context.Context, log *zap.SugaredLogger) *DefaultClient {
	c, err := NewGenericClient(ctx, log)
	if err != nil {
		panic(err)
	}
	return c
}

// newClient creates a new sync client.
func newClient(ctx context.Context, log *zap.SugaredLogger, extractor func(ctx context.Context) *runtime.RunParams) (*DefaultClient, error) {
	ctx, cancel := context.WithCancel(ctx)
	c := &DefaultClient{
		ctx:       ctx,
		cancel:    cancel,
		log:       log,
		extractor: extractor,
		handlers:  map[string]chan *tgsync.Response{},
	}

	c.sugarOperations = &sugarOperations{c}

	addr, err := socketAddress()
	if err != nil {
		return nil, err
	}

	c.socket, _, err = websocket.Dial(ctx, addr, nil)
	if err != nil {
		return nil, err
	}

	c.wg.Add(1)
	go c.responsesWorker()

	return c, nil
}

// Close closes this client, cancels ongoing operations, and releases resources.
func (c *DefaultClient) Close() error {
	err := c.socket.Close(websocket.StatusNormalClosure, "")
	if err != nil {
		return err
	}

	c.cancel()
	c.wg.Wait()
	return nil
}

func socketAddress() (string, error) {
	var (
		port = os.Getenv(EnvServicePort)
		host = os.Getenv(EnvServiceHost)
	)

	if port == "" {
		port = "5050"
	}

	if host == "" {
		host = "testground-sync-service"
	}

	return fmt.Sprintf("ws://%s:%s", host, port), nil
}
