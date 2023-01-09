package publicapi

import (
	"encoding/json"
	"net/http"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/version"
)

type versionRequest struct {
	ClientID string `json:"client_id" example:"ac13188e93c97a9c2e7cf8e86c7313156a73436036f30da1ececc2ce79f9ea51"`
}

type versionResponse struct {
	VersionInfo *model.BuildVersionInfo `json:"build_version_info"`
}

// version godoc
//
//	@ID				version
//	@Summary		Returns the build version running on the server.
//	@Description	See https://github.com/filecoin-project/bacalhau/releases for a complete list of `gitversion` tags.
//	@Tags			Utils
//	@Accept			json
//	@Produce		json
//	@Param			versionRequest	body		versionRequest	true	"Request must specify a `client_id`. To retrieve your `client_id`, you can do the following: (1) submit a dummy job to Bacalhau (or use one you created before), (2) run `bacalhau describe <job-id>` and fetch the `ClientID` field."
//	@Success		200				{object}	versionResponse
//	@Failure		400				{object}	string
//	@Failure		500				{object}	string
//	@Router			/version [post]
//
//nolint:lll
func (apiServer *APIServer) version(res http.ResponseWriter, req *http.Request) {
	ctx, span := system.GetSpanFromRequest(req, "apiServer/version")
	defer span.End()

	t := system.GetTracer()

	_, unMarshallSpan := t.Start(ctx, "unmarshallingversionrequest")
	var versionReq versionRequest
	err := json.NewDecoder(req.Body).Decode(&versionReq)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
	unMarshallSpan.End()

	_, respondingSpan := t.Start(ctx, "encodingversionresponse")
	res.WriteHeader(http.StatusOK)
	err = json.NewEncoder(res).Encode(versionResponse{
		VersionInfo: version.Get(),
	})
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	respondingSpan.End()
}
