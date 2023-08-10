package publicapi

import (
	"net/http"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi"
)

const APIPrefix = "compute/"
const APIDebugSuffix = "debug"
const APIApproveSuffix = "approve"

type ComputeAPIServerParams struct {
	APIServer          *publicapi.APIServer
	Bidder             compute.Bidder
	Store              store.ExecutionStore
	DebugInfoProviders []models.DebugInfoProvider
}

type ComputeAPIServer struct {
	apiServer          *publicapi.APIServer
	bidder             compute.Bidder
	store              store.ExecutionStore
	debugInfoProviders []models.DebugInfoProvider
}

func NewComputeAPIServer(params ComputeAPIServerParams) *ComputeAPIServer {
	return &ComputeAPIServer{
		apiServer:          params.APIServer,
		bidder:             params.Bidder,
		store:              params.Store,
		debugInfoProviders: params.DebugInfoProviders,
	}
}

func (s *ComputeAPIServer) RegisterAllHandlers() error {
	handlerConfigs := []publicapi.HandlerConfig{
		{Path: "/" + APIPrefix + APIDebugSuffix, Handler: http.HandlerFunc(s.debug)},
		{Path: "/" + APIPrefix + APIApproveSuffix, Handler: http.HandlerFunc(s.approve)},
	}
	// register URIs at root prefix for backward compatibility before migrating to API versioning
	// we should remove these eventually, or have throttling limits shared across versions
	err := s.apiServer.RegisterHandlers(publicapi.LegacyAPIPrefix, handlerConfigs...)
	if err != nil {
		return err
	}
	return s.apiServer.RegisterHandlers(publicapi.V1APIPrefix, handlerConfigs...)
}
