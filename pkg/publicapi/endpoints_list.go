package publicapi

import (
	"context"
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
	Jobs map[string]*model.Job `json:"jobs"`
}

func (apiServer *APIServer) list(res http.ResponseWriter, req *http.Request) {
	ctx, span := system.GetSpanFromRequest(req, "apiServer/list")
	defer span.End()

	var listReq listRequest
	if err := json.NewDecoder(req.Body).Decode(&listReq); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	list, err := apiServer.getJobs(ctx, res)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	rawJobs := map[string]*model.Job{}

	for _, listJob := range list { //nolint:gocritic
		rawJobs[listJob.ID] = listJob
	}

	res.WriteHeader(http.StatusOK)
	err = json.NewEncoder(res).Encode(listResponse{
		Jobs: rawJobs,
	})
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (apiServer *APIServer) getJobs(ctx context.Context, res http.ResponseWriter) ([]*model.Job, error) {
	ctx, span := system.GetTracer().Start(ctx, "pkg/publicapi.list")
	defer span.End()

	list, err := apiServer.Controller.GetJobs(ctx, localdb.JobQuery{})
	if err != nil {
		// Handle error in the calling function, as this function only does one thing.
		return nil, nil
	}
	return list, err
}
