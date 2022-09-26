package publicapi

import (
	"encoding/json"
	"net/http"

	"github.com/filecoin-project/bacalhau/pkg/localdb"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
)

type listRequest struct {
	JobID       string `json:"id"`
	ClientID    string `json:"client_id"`
	MaxJobs     int    `json:"max_jobs"`
	ReturnAll   bool   `json:"return_all"`
	SortBy      string `json:"sort_by"`
	SortReverse bool   `json:"sort_reverse"`
}

type listResponse struct {
	JobsWithInfo []model.JobWithInfo `json:"jobs"`
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

	// get Jobs
	getJobsCtx, getJobsSpan := t.Start(ctx, "gettingjobs")
	list, err := apiServer.Controller.GetJobs(getJobsCtx, localdb.JobQuery{
		ClientID:    listReq.ClientID,
		ID:          listReq.JobID,
		Limit:       listReq.MaxJobs,
		ReturnAll:   listReq.ReturnAll,
		SortBy:      listReq.SortBy,
		SortReverse: listReq.SortReverse,
	})
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	getJobsSpan.End()

	// get JobStates
	getJobStateCtx, getJobStateSpan := t.Start(ctx, "gettingjobstates")
	jobsWithInfo := []model.JobWithInfo{}
	for i := range list {
		jobState, err := apiServer.Controller.GetJobState(getJobStateCtx, list[i].ID)
		if err != nil {
			log.Error().Msgf("error getting job state: %s", err)
		}
		jobsWithInfo = append(jobsWithInfo, model.JobWithInfo{
			Job:      list[i],
			JobState: jobState,
		})
	}
	getJobStateSpan.End()

	_, marshallSpan := t.Start(ctx, "marshallingresponse")
	res.WriteHeader(http.StatusOK)
	err = json.NewEncoder(res).Encode(listResponse{
		JobsWithInfo: jobsWithInfo,
	})
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	marshallSpan.End()
}
