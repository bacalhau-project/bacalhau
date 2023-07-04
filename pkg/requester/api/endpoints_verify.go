package api

import (
	"net/http"

	"github.com/bacalhau-project/bacalhau/pkg/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/verifier/external"
)

func (s *RequesterAPIServer) verify(res http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	verification, err := publicapi.UnmarshalSigned[external.ExternalVerificationResponse](ctx, req.Body)
	if err != nil {
		publicapi.HTTPError(ctx, res, err, http.StatusBadRequest)
		return
	}

	err = s.requester.VerifyExecutions(ctx, verification)
	if err != nil {
		publicapi.HTTPError(ctx, res, err, http.StatusBadRequest)
		return
	}
}
