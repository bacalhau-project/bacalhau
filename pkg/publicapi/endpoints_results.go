package publicapi

import (
	"encoding/json"
	"net/http"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
)

type resultsRequest struct {
	ClientID string `json:"client_id"`
	JobID    string `json:"job_id"`
}

type resultsResponse struct {
	Results []model.StorageSpec `json:"results"`
}

func (apiServer *APIServer) results(res http.ResponseWriter, req *http.Request) {
	ctx, span := system.GetSpanFromRequest(req, "apiServer/results")
	defer span.End()
	t := system.GetTracer()

	var stateReq stateRequest
	if err := json.NewDecoder(req.Body).Decode(&stateReq); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	getPublisherCtx, getPublisherSpan := t.Start(ctx, "gettingpublisher")
	publisher, err := apiServer.getPublisher(getPublisherCtx, model.PublisherIpfs)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	getPublisherSpan.End()

	composeResultsCtx, composeResultsSpan := t.Start(ctx, "composingresults")
	results, err := publisher.ComposeResultReferences(composeResultsCtx, stateReq.JobID)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	composeResultsSpan.End()

	res.WriteHeader(http.StatusOK)
	err = json.NewEncoder(res).Encode(resultsResponse{
		Results: results,
	})
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
}
