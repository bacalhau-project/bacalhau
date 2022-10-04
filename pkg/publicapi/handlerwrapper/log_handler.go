package handlerwrapper

import (
	"encoding/json"

	"github.com/rs/zerolog/log"
)

type JSONLogHandler struct {
}

func NewJSONLogHandler() *JSONLogHandler {
	return &JSONLogHandler{}
}

func (h *JSONLogHandler) Handle(ri *HTTPRequestInfo) {
	jsonBytes, err := json.Marshal(ri)
	if err != nil {
		log.Info().Err(err).Msgf("failed to marshal request info %+v", ri)
	}
	if ri.StatusCode >= 400 { //nolint:gomnd
		log.Error().Msg(string(jsonBytes))
	} else {
		log.Info().Msg(string(jsonBytes))
	}
}
