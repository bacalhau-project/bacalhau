package publicapi

import (
	"net/http"

	"github.com/filecoin-project/bacalhau/pkg/localdb"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/requester"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/gorilla/websocket"
	sync "github.com/lukemarsden/golang-mutex-tracer"
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

	nodeWebsockets      []*websocket.Conn
	nodeWebsocketsMutex sync.RWMutex
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
		{URI: "/" + APIPrefix + "list", Handler: http.HandlerFunc(s.List)},
		{URI: "/" + APIPrefix + "states", Handler: http.HandlerFunc(s.States)},
		{URI: "/" + APIPrefix + "results", Handler: http.HandlerFunc(s.Results)},
		{URI: "/" + APIPrefix + "events", Handler: http.HandlerFunc(s.Events)},
		{URI: "/" + APIPrefix + "local_events", Handler: http.HandlerFunc(s.LocalEvents)},
		{URI: "/" + APIPrefix + "submit", Handler: http.HandlerFunc(s.Submit)},
		{URI: "/" + APIPrefix + "websocket", Handler: http.HandlerFunc(s.websocket), Raw: true},
		{URI: "/" + APIPrefix + "node/websocket", Handler: http.HandlerFunc(s.websocketNode), Raw: true},
		{URI: "/" + APIPrefix + "debug", Handler: http.HandlerFunc(s.debug)},
	}
	return s.apiServer.RegisterHandlers(handlerConfigs...)
}
