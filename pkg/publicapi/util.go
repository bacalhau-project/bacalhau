package publicapi

import (
	"context"
	"net/http"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/rs/zerolog/log"
)

func HTTPError(ctx context.Context, res http.ResponseWriter, err error, statusCode int) {
	log.Ctx(ctx).Error().Err(err).Send()
	http.Error(res, bacerrors.ErrorToErrorResponse(err), statusCode)
}
