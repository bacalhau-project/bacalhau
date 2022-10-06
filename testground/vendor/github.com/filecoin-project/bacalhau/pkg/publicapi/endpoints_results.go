package publicapi

import (
	"encoding/json"
	"net/http"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publicapi/handlerwrapper"
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
	ctx, span := system.GetSpanFromRequest(req, "pkg/publicapi.results")
	defer span.End()

	var stateReq stateRequest
	if err := json.NewDecoder(req.Body).Decode(&stateReq); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
	res.Header().Set(handlerwrapper.HTTPHeaderClientID, stateReq.ClientID)
	res.Header().Set(handlerwrapper.HTTPHeaderJobID, stateReq.JobID)

	ctx = system.AddJobIDToBaggage(ctx, stateReq.JobID)
	system.AddJobIDFromBaggageToSpan(ctx, span)

	publisher, err := apiServer.Publishers.GetPublisher(ctx, model.PublisherIpfs)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	results, err := publisher.ComposeResultReferences(ctx, stateReq.JobID)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	res.WriteHeader(http.StatusOK)
	err = json.NewEncoder(res).Encode(resultsResponse{
		Results: results,
	})
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
}
