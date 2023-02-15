package publicapi

import (
	"net/http"

	sync "github.com/bacalhau-project/golang-mutex-tracer"
	"github.com/filecoin-project/bacalhau/pkg/localdb"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/requester"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/gorilla/websocket"
)

const APIPrefix = "requester/"

type RequesterAPIServerParams struct {
	APIServer          *publicapi.APIServer
	Requester          requester.Endpoint
	DebugInfoProviders []model.DebugInfoProvider
	LocalDB            localdb.LocalDB
	StorageProviders   storage.StorageProvider
}

type RequesterAPIServer struct {
	apiServer          *publicapi.APIServer
	requester          requester.Endpoint
	debugInfoProviders []model.DebugInfoProvider
	localDB            localdb.LocalDB
	storageProviders   storage.StorageProvider
	// jobId or "" (for all events) -> connections for that subscription
	websockets      map[string][]*websocket.Conn
	websocketsMutex sync.RWMutex
}

func NewRequesterAPIServer(params RequesterAPIServerParams) *RequesterAPIServer {
	return &RequesterAPIServer{
		apiServer:          params.APIServer,
		requester:          params.Requester,
		debugInfoProviders: params.DebugInfoProviders,
		localDB:            params.LocalDB,
		storageProviders:   params.StorageProviders,
		websockets:         make(map[string][]*websocket.Conn),
	}
}

func (s *RequesterAPIServer) RegisterAllHandlers() error {
	handlerConfigs := []publicapi.HandlerConfig{
		{URI: "/" + APIPrefix + "list", Handler: http.HandlerFunc(s.list)},
		{URI: "/" + APIPrefix + "states", Handler: http.HandlerFunc(s.states)},
		{URI: "/" + APIPrefix + "results", Handler: http.HandlerFunc(s.results)},
		{URI: "/" + APIPrefix + "events", Handler: http.HandlerFunc(s.events)},
		{URI: "/" + APIPrefix + "local_events", Handler: http.HandlerFunc(s.localEvents)},
		{URI: "/" + APIPrefix + "submit", Handler: http.HandlerFunc(s.submit)},
		{URI: "/" + APIPrefix + "websocket/events", Handler: http.HandlerFunc(s.websocketJobEvents), Raw: true},
		{URI: "/" + APIPrefix + "debug", Handler: http.HandlerFunc(s.debug)},
	}
	return s.apiServer.RegisterHandlers(handlerConfigs...)
}
