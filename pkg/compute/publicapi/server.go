package publicapi

import (
	"net/http"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi"
)

const APIPrefix = "compute/"
const APIDebugSuffix = "debug"

type ComputeAPIServerParams struct {
	APIServer          *publicapi.APIServer
	DebugInfoProviders []model.DebugInfoProvider
}

type ComputeAPIServer struct {
	apiServer          *publicapi.APIServer
	debugInfoProviders []model.DebugInfoProvider
}

func NewComputeAPIServer(params ComputeAPIServerParams) *ComputeAPIServer {
	return &ComputeAPIServer{
		apiServer:          params.APIServer,
		debugInfoProviders: params.DebugInfoProviders,
	}
}

func (s *ComputeAPIServer) RegisterAllHandlers() error {
	handlerConfigs := []publicapi.HandlerConfig{
		{URI: "/" + APIPrefix + APIDebugSuffix, Handler: http.HandlerFunc(s.debug)},
	}
	return s.apiServer.RegisterHandlers(handlerConfigs...)
}
