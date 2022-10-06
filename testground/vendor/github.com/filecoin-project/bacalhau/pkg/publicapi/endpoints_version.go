package publicapi

import (
	"encoding/json"
	"net/http"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/version"
)

type versionRequest struct {
	ClientID string `json:"client_id"`
}

type versionResponse struct {
	VersionInfo *model.BuildVersionInfo `json:"build_version_info"`
}

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
