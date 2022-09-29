package publicapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/localdb"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/transport"

	sync "github.com/lukemarsden/golang-mutex-tracer"

	"github.com/didip/tollbooth"
	"github.com/didip/tollbooth/limiter"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publisher"
	"github.com/filecoin-project/bacalhau/pkg/requesternode"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// APIServer configures a node's public REST API.
type APIServer struct {
<<<<<<< HEAD
	Controller  *controller.Controller
	Publishers  map[model.Publisher]publisher.Publisher
	Host        string
	Port        int
	componentMu sync.Mutex
||||||| 5d1cca3e
	Controller  *controller.Controller
	Publishers  map[model.PublisherType]publisher.Publisher
	Host        string
	Port        int
	componentMu sync.Mutex
=======
	localdb          localdb.LocalDB
	transport        transport.Transport
	Requester        *requesternode.RequesterNode
	Publishers       map[model.PublisherType]publisher.Publisher
	StorageProviders map[model.StorageSourceType]storage.StorageProvider
	Host             string
	Port             int
	componentMu      sync.Mutex
>>>>>>> main
}

func init() { //nolint:gochecknoinits
	sync.SetGlobalOpts(sync.Opts{
		Threshold: 10 * time.Millisecond,
		Enabled:   true,
		Id:        "<UNKNOWN>",
	})
}

const ServerReadHeaderTimeout = 10 * time.Second

// NewServer returns a new API server for a requester node.
func NewServer(
	ctx context.Context,
	host string,
	port int,
<<<<<<< HEAD
	c *controller.Controller,
	publishers map[model.Publisher]publisher.Publisher,
||||||| 5d1cca3e
	c *controller.Controller,
	publishers map[model.PublisherType]publisher.Publisher,
=======
	localdb localdb.LocalDB,
	transport transport.Transport,
	requester *requesternode.RequesterNode,
	publishers map[model.PublisherType]publisher.Publisher,
	storageProviders map[model.StorageSourceType]storage.StorageProvider,
>>>>>>> main
) *APIServer {
	a := &APIServer{
		localdb:          localdb,
		transport:        transport,
		Requester:        requester,
		Publishers:       publishers,
		StorageProviders: storageProviders,
		Host:             host,
		Port:             port,
	}
	a.componentMu.EnableTracerWithOpts(sync.Opts{
		Threshold: 10 * time.Millisecond,
		Id:        "APIServer.componentMu",
	})
	return a
}

// GetURI returns the HTTP URI that the server is listening on.
func (apiServer *APIServer) GetURI() string {
	return fmt.Sprintf("http://%s:%d", apiServer.Host, apiServer.Port)
}

// ListenAndServe listens for and serves HTTP requests against the API server.
func (apiServer *APIServer) ListenAndServe(ctx context.Context, cm *system.CleanupManager) error {
	hostID := apiServer.Requester.ID

	throttle := func(h http.Handler) http.Handler {
		return tollbooth.LimitHandler(
			tollbooth.NewLimiter(
				1000, //nolint:gomnd
				&limiter.ExpirableOptions{DefaultExpirationTTL: time.Hour}),
			h,
		)
	}

	// TODO: #677 Significant issue, when client returns error to any of these commands, it still submits to server
	sm := http.NewServeMux()
	sm.Handle("/list", throttle(instrument("list", apiServer.list)))
	sm.Handle("/states", throttle(instrument("states", apiServer.states)))
	sm.Handle("/results", throttle(instrument("results", apiServer.results)))
	sm.Handle("/events", throttle(instrument("events", apiServer.events)))
	sm.Handle("/local_events", throttle(instrument("local_events", apiServer.localEvents)))
	sm.Handle("/id", throttle(instrument("id", apiServer.id)))
	sm.Handle("/peers", throttle(instrument("peers", apiServer.peers)))
	sm.Handle("/submit", throttle(instrument("submit", apiServer.submit)))
	sm.Handle("/version", throttle(instrument("version", apiServer.version)))
	sm.Handle("/healthz", throttle(instrument("healthz", apiServer.healthz)))
	sm.Handle("/logz", throttle(instrument("logz", apiServer.logz)))
	sm.Handle("/varz", throttle(instrument("varz", apiServer.varz)))
	sm.Handle("/livez", throttle(instrument("livez", apiServer.livez)))
	sm.Handle("/readyz", throttle(instrument("readyz", apiServer.readyz)))
	sm.Handle("/metrics", promhttp.Handler())

	srv := http.Server{
		Handler:           sm,
		Addr:              fmt.Sprintf("%s:%d", apiServer.Host, apiServer.Port),
		ReadHeaderTimeout: ServerReadHeaderTimeout,
	}

	log.Debug().Msgf(
		"API server listening for host %s on %s...", hostID, srv.Addr)

	// Cleanup resources when system is done:
	cm.RegisterCallback(func() error {
		return srv.Shutdown(ctx)
	})

	err := srv.ListenAndServe()
	if err == http.ErrServerClosed {
		log.Debug().Msgf(
			"API server closed for host %s on %s.", hostID, srv.Addr)
		return nil // expected error if the server is shut down
	}

	return err
}

func (apiServer *APIServer) getPublisher(ctx context.Context, typ model.Publisher) (publisher.Publisher, error) {
	ctx, span := system.GetTracer().Start(ctx, "pkg/publicapi/getPublisher")
	defer span.End()

	apiServer.componentMu.Lock()
	defer apiServer.componentMu.Unlock()

	if _, ok := apiServer.Publishers[typ]; !ok {
		return nil, fmt.Errorf("no matching verifier found on this server: %s", typ.String())
	}

	v := apiServer.Publishers[typ]
	installed, err := v.IsInstalled(ctx)
	if err != nil {
		return nil, err
	}
	if !installed {
		return nil, fmt.Errorf("verifier is not installed: %s", typ.String())
	}

	return v, nil
}

func verifySubmitRequest(req *submitRequest) error {
	if req.Data.ClientID == "" {
		return errors.New("job deal must contain a client ID")
	}
	if req.ClientSignature == "" {
		return errors.New("client's signature is required")
	}
	if req.ClientPublicKey == "" {
		return errors.New("client's public key is required")
	}

	// Check that the client's public key matches the client ID:
	ok, err := system.PublicKeyMatchesID(req.ClientPublicKey, req.Data.ClientID)
	if err != nil {
		return fmt.Errorf("error verifying client ID: %w", err)
	}
	if !ok {
		return errors.New("client's public key does not match client ID")
	}

	// Check that the signature is valid:
	jsonData, err := json.Marshal(req.Data)
	if err != nil {
		return fmt.Errorf("error marshaling job data: %w", err)
	}

	err = system.Verify(jsonData, req.ClientSignature, req.ClientPublicKey)
	if err != nil {
		return fmt.Errorf("client's signature is invalid: %w", err)
	}

	return nil
}

func instrument(name string, fn http.HandlerFunc) http.Handler {
	return otelhttp.NewHandler(fn, fmt.Sprintf("pkg/publicapi/%s", name))
}
