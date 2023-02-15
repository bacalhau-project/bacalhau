package publicapi

import (
	"encoding/json"
	"net/http"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/version"
)

type VersionRequest struct {
	ClientID string `json:"client_id" example:"ac13188e93c97a9c2e7cf8e86c7313156a73436036f30da1ececc2ce79f9ea51"`
}

type VersionResponse struct {
	VersionInfo *model.BuildVersionInfo `json:"build_version_info"`
}

// version godoc
//
//	@ID				apiServer/version
//	@Summary		Returns the build version running on the server.
//	@Description	See https://github.com/filecoin-project/bacalhau/releases for a complete list of `gitversion` tags.
//	@Tags			Misc
//	@Accept			json
//	@Produce		json
//	@Param			VersionRequest	body		VersionRequest	true	"Request must specify a `client_id`. To retrieve your `client_id`, you can do the following: (1) submit a dummy job to Bacalhau (or use one you created before), (2) run `bacalhau describe <job-id>` and fetch the `ClientID` field."
//	@Success		200				{object}	VersionResponse
//	@Failure		400				{object}	string
//	@Failure		500				{object}	string
//	@Router			/version [post]
//
//nolint:lll
func (apiServer *APIServer) version(res http.ResponseWriter, req *http.Request) {
	var versionReq VersionRequest
	err := json.NewDecoder(req.Body).Decode(&versionReq)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	res.WriteHeader(http.StatusOK)
	err = json.NewEncoder(res).Encode(VersionResponse{
		VersionInfo: version.Get(),
	})
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
}
