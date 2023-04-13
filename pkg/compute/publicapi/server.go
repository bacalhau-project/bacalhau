package publicapi

import (
	"net/http"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi"
)

const APIPrefix = "compute/"
const APIDebugSuffix = "debug"
const APIApproveSuffix = "approve"

type ComputeAPIServerParams struct {
	APIServer          *publicapi.APIServer
	Bidder             compute.Bidder
	Store              store.ExecutionStore
	DebugInfoProviders []model.DebugInfoProvider
}

type ComputeAPIServer struct {
	apiServer          *publicapi.APIServer
	bidder             compute.Bidder
	store              store.ExecutionStore
	debugInfoProviders []model.DebugInfoProvider
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
		{URI: "/" + APIPrefix + APIDebugSuffix, Handler: http.HandlerFunc(s.debug)},
		{URI: "/" + APIPrefix + APIApproveSuffix, Handler: http.HandlerFunc(s.approve)},
	}
	return s.apiServer.RegisterHandlers(handlerConfigs...)
}
