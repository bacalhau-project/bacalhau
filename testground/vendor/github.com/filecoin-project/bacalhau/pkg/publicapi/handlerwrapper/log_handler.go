package handlerwrapper

import (
	"context"
	"encoding/json"

	"github.com/rs/zerolog/log"
)

type JSONLogHandler struct {
}

func NewJSONLogHandler() *JSONLogHandler {
	return &JSONLogHandler{}
}

func (h *JSONLogHandler) Handle(ctx context.Context, ri *HTTPRequestInfo) {
	jsonBytes, err := json.Marshal(ri)
	if err != nil {
		log.Ctx(ctx).Info().Err(err).Msgf("failed to marshal request info %+v", ri)
	}
	if ri.StatusCode >= 400 { //nolint:gomnd
		log.Ctx(ctx).Error().Msg(string(jsonBytes))
	} else {
		log.Ctx(ctx).Info().Msg(string(jsonBytes))
	}
}
