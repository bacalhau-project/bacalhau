package publicapi

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/filecoin-project/bacalhau/pkg/bacerrors"
	"github.com/filecoin-project/bacalhau/pkg/localdb"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publicapi/handlerwrapper"
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
	Jobs []*model.Job `json:"jobs"`
}

func (apiServer *APIServer) list(res http.ResponseWriter, req *http.Request) {
	ctx, span := system.GetSpanFromRequest(req, "pkg/publicapi.list")
	defer span.End()

	var listReq listRequest
	if err := json.NewDecoder(req.Body).Decode(&listReq); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
	res.Header().Set(handlerwrapper.HTTPHeaderClientID, listReq.ClientID)
	res.Header().Set(handlerwrapper.HTTPHeaderJobID, listReq.JobID)

	jobList, err := apiServer.getJobsList(ctx, listReq)
	if err != nil {
		_, ok := err.(*bacerrors.JobNotFound)
		if ok {
			http.Error(res, bacerrors.ErrorToErrorResponse(err), http.StatusBadRequest)
			return
		}
	}
	if len(jobList) > 0 {
		// get JobStates
		err = apiServer.getJobStates(ctx, jobList)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("error getting job states")
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	res.WriteHeader(http.StatusOK)
	err = json.NewEncoder(res).Encode(listResponse{
		Jobs: jobList,
	})
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (apiServer *APIServer) getJobsList(ctx context.Context, listReq listRequest) ([]*model.Job, error) {
	ctx, span := system.GetTracer().Start(ctx, "pkg/publicapi.list")
	defer span.End()

	list, err := apiServer.localdb.GetJobs(ctx, localdb.JobQuery{
		ClientID:    listReq.ClientID,
		ID:          listReq.JobID,
		Limit:       listReq.MaxJobs,
		ReturnAll:   listReq.ReturnAll,
		SortBy:      listReq.SortBy,
		SortReverse: listReq.SortReverse,
	})
	if err != nil {
		return nil, err
	}
	return list, nil
}

func (apiServer *APIServer) getJobStates(ctx context.Context, jobList []*model.Job) error {
	ctx, span := system.GetTracer().Start(ctx, "pkg/publicapi.getJobStates")
	defer span.End()

	var err error
	for k := range jobList {
		jobList[k].State, err = apiServer.localdb.GetJobState(ctx, jobList[k].ID)
		if err != nil {
			log.Ctx(ctx).Error().Msgf("error getting job state: %s", err)
			return err
		}
	}
	return nil
}
