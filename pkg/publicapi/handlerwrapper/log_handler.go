package handlerwrapper

import (
	"context"
	"net/http"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

type JSONLogHandler struct {
}

func NewJSONLogHandler() *JSONLogHandler {
	return &JSONLogHandler{}
}

func (h *JSONLogHandler) Handle(ctx context.Context, ri *HTTPRequestInfo) {
	jsonBytes, err := model.JSONMarshalWithMax(ri)
	if err != nil {
		log.Ctx(ctx).Info().Err(err).Msgf("failed to marshal request info %+v", ri)
	}
	if ri.StatusCode >= http.StatusBadRequest {
		log.Ctx(ctx).Error().RawJSON("Request", jsonBytes).Send()
	} else {
		// TODO: #830 Same as #829 in pkg/eventhandler/chained_handlers.go
		if system.GetEnvironment() == system.EnvironmentTest ||
			system.GetEnvironment() == system.EnvironmentDev {
			return
		}
		log.Ctx(ctx).Info().RawJSON("Request", jsonBytes).Send()
	}
}
