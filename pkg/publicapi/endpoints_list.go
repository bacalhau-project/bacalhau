package publicapi

import (
	"encoding/json"
	"net/http"

	"github.com/filecoin-project/bacalhau/pkg/localdb"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
)

type listRequest struct {
	ClientID string `json:"client_id"`
}

type listResponse struct {
	Jobs map[string]model.Job `json:"jobs"`
}

func (apiServer *APIServer) list(res http.ResponseWriter, req *http.Request) {
	ctx, span := system.GetSpanFromRequest(req, "apiServer/list")
	defer span.End()
	t := system.GetTracer()

	_, unMarshallSpan := t.Start(ctx, "unmarshallinglistrequest")
	var listReq listRequest
	if err := json.NewDecoder(req.Body).Decode(&listReq); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
	unMarshallSpan.End()

	getJobsCtx, getJobsSpan := t.Start(ctx, "gettingjobs")
	list, err := apiServer.Controller.GetJobs(getJobsCtx, localdb.JobQuery{})
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	getJobsSpan.End()

	rawJobs := map[string]model.Job{}

	for _, listJob := range list { //nolint:gocritic
		rawJobs[listJob.ID] = listJob
	}

	_, marshallSpan := t.Start(ctx, "marshallingresponse")
	res.WriteHeader(http.StatusOK)
	err = json.NewEncoder(res).Encode(listResponse{
		Jobs: rawJobs,
	})
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	marshallSpan.End()
}
