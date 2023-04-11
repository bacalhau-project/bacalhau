package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/gorilla/mux"

	bacmodel "github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/model/v1beta1"

	"github.com/bacalhau-project/bacalhau/dashboard/api/pkg/model"
	"github.com/bacalhau-project/bacalhau/dashboard/api/pkg/types"
	"github.com/bacalhau-project/bacalhau/pkg/localdb"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

type ServerOptions struct {
	Host        string
	Port        int
	SwarmPort   int
	PeerConnect string
	JWTSecret   string
}

type DashboardAPIServer struct {
	Options ServerOptions
	API     *model.ModelAPI
}

func NewServer(
	options ServerOptions,
	api *model.ModelAPI,
) (*DashboardAPIServer, error) {
	if options.Host == "" {
		return nil, fmt.Errorf("host is required")
	}
	if options.Port == 0 {
		return nil, fmt.Errorf("port is required")
	}
	if options.JWTSecret == "" {
		return nil, fmt.Errorf("jwt secret is required")
	}
	return &DashboardAPIServer{
		Options: options,
		API:     api,
	}, nil
}

func (apiServer *DashboardAPIServer) URL() *url.URL {
	url, err := url.Parse(fmt.Sprintf("http://%s:%d/", apiServer.Options.Host, apiServer.Options.Port))
	if err != nil {
		panic(err)
	}
	return url
}

func (apiServer *DashboardAPIServer) ListenAndServe(ctx context.Context, cm *system.CleanupManager) error {
	authnHandler := func(handler httpErrorFunc) httpErrorFunc {
		return requiresLogin(apiServer.API, apiServer.Options.JWTSecret, handler)
	}

	router := mux.NewRouter()
	subrouter := router.PathPrefix("/api/v1").Subrouter()
	subrouter.HandleFunc("/nodes", handleError(returnsJSON(expectsNothing(apiServer.API.GetNodes)))).Methods("GET")
	subrouter.HandleFunc("/run", apiServer.run).Methods("POST")
	subrouter.HandleFunc("/stablediffusion", apiServer.stablediffusion).Methods("POST")
	subrouter.HandleFunc("/jobs", handleError(returnsJSON(expectsJSON(apiServer.API.GetJobs)))).Methods("POST")
	subrouter.HandleFunc("/jobs/count", handleError(returnsJSON(expectsJSON(apiServer.jobsCount)))).Methods("POST")
	subrouter.HandleFunc("/jobs/shouldrun", handleError(returnsJSON(expectsJSON(apiServer.API.ShouldExecuteJob)))).Methods("POST")

	jobrouter := subrouter.PathPrefix("/job/{id}").Subrouter()
	jobrouter.HandleFunc("/", handleError(returnsJSON(apiServer.job))).Methods("GET")
	jobrouter.HandleFunc("/info", handleError(returnsJSON(apiServer.jobInfo))).Methods("GET")
	jobrouter.HandleFunc("/inputs", handleError(returnsJSON(apiServer.jobProducingInputs))).Methods("GET")
	jobrouter.HandleFunc("/outputs", handleError(returnsJSON(apiServer.jobOperatingOnOutputs))).Methods("GET")
	jobrouter.HandleFunc("/datacap", handleError(authnHandler(returnsJSON(apiServer.moderateJobDatacap)))).Methods("POST")
	jobrouter.HandleFunc("/exec", handleError(authnHandler(returnsJSON(apiServer.moderateJobRequest)))).Methods("POST")

	cidrouter := subrouter.PathPrefix("/cid/{cid}").Subrouter()
	cidrouter.HandleFunc("/jobs", handleError(returnsJSON(apiServer.findJobsOperatingOnCID))).Methods("GET")

	statrouter := subrouter.PathPrefix("/summary").Subrouter()
	statrouter.HandleFunc("/annotations", handleError(returnsJSON(expectsNothing(apiServer.API.GetAnnotationSummary)))).Methods("GET")
	statrouter.HandleFunc("/jobmonths", handleError(returnsJSON(expectsNothing(apiServer.API.GetJobMonthSummary)))).Methods("GET")
	statrouter.HandleFunc("/jobexecutors", handleError(returnsJSON(expectsNothing(apiServer.API.GetJobExecutorSummary)))).Methods("GET")
	statrouter.HandleFunc("/totaljobs", handleError(returnsJSON(expectsNothing(apiServer.API.GetTotalJobsCount)))).Methods("GET")
	statrouter.HandleFunc("/totaljobevents", handleError(returnsJSON(expectsNothing(apiServer.API.GetTotalEventCount)))).Methods("GET")
	statrouter.HandleFunc("/totalusers", handleError(returnsJSON(expectsNothing(apiServer.API.GetTotalUserCount)))).Methods("GET")
	statrouter.HandleFunc("/totalexecutors", handleError(returnsJSON(expectsNothing(apiServer.API.GetTotalExecutorCount)))).Methods("GET")

	subrouter.HandleFunc("/admin/login", handleError(returnsJSON(expectsJSON(apiServer.adminlogin)))).Methods("POST")
	subrouter.HandleFunc("/admin/status", handleError(authnHandler(returnsJSON(apiServer.adminstatus)))).Methods("GET")

	srv := &http.Server{
		Addr:              fmt.Sprintf("%s:%d", apiServer.Options.Host, apiServer.Options.Port),
		WriteTimeout:      time.Minute * 15,
		ReadTimeout:       time.Minute * 15,
		ReadHeaderTimeout: time.Minute * 15,
		IdleTimeout:       time.Minute * 60,
		Handler:           router,
	}
	cm.RegisterCallbackWithContext(srv.Shutdown)
	return srv.ListenAndServe()
}

type PromptParam struct {
	Prompt string `json:"prompt"`
}

// TODO: factor commonality from following two funcs
func (apiServer *DashboardAPIServer) run(res http.ResponseWriter, req *http.Request) {
	// any crazy mofo on the planet can build this into their web apps
	res.Header().Set("Access-Control-Allow-Origin", "*")

	spec := bacmodel.Spec{}
	err := json.NewDecoder(req.Body).Decode(&spec)
	if err != nil {
		_, _ = res.Write([]byte(fmt.Sprintf(`{"error": "%s"}`, strings.Trim(err.Error(), "\n"))))
		return
	}

	cid, err := runGenericJob(spec)
	if err != nil {
		log.Ctx(req.Context()).Error().Err(err).Send()
		_, _ = res.Write([]byte(fmt.Sprintf(`{"error": "%s"}`, strings.Trim(err.Error(), "\n"))))
	} else {
		log.Ctx(req.Context()).Info().Str("CID", cid).Send()
		_, _ = res.Write([]byte(fmt.Sprintf(`{"cid": "%s"}`, strings.Trim(cid, "\n"))))
	}
}

func (apiServer *DashboardAPIServer) stablediffusion(res http.ResponseWriter, req *http.Request) {
	// any crazy mofo on the planet can build this into their web apps
	res.Header().Set("Access-Control-Allow-Origin", "*")

	promptParam := PromptParam{}
	err := json.NewDecoder(req.Body).Decode(&promptParam)
	if err != nil {
		_, _ = res.Write([]byte(fmt.Sprintf(`{"error": "%s"}`, strings.Trim(err.Error(), "\n"))))
		return
	}
	prompt := promptParam.Prompt

	// user can pass ?testing=1 to bypass GPU and just return the prompt
	testing := len(req.URL.Query()["testing"]) > 0

	log.Ctx(req.Context()).Info().Msgf("--> testing=%t", testing)

	cid, err := runStableDiffusion(prompt, testing)
	if err != nil {
		log.Ctx(req.Context()).Error().Err(err).Send()
		_, _ = res.Write([]byte(fmt.Sprintf(`{"error": "%s"}`, strings.Trim(err.Error(), "\n"))))
	} else {
		log.Ctx(req.Context()).Info().Str("CID", cid).Send()
		_, _ = res.Write([]byte(fmt.Sprintf(`{"cid": "%s"}`, strings.Trim(cid, "\n"))))
	}
}

type jobsCountResponse struct {
	Count int `json:"count"`
}

func (apiServer *DashboardAPIServer) jobsCount(ctx context.Context, query localdb.JobQuery) (*jobsCountResponse, error) {
	count, err := apiServer.API.GetJobsCount(ctx, query)
	return &jobsCountResponse{Count: count}, err
}

func (apiServer *DashboardAPIServer) job(ctx context.Context, req *http.Request) (*v1beta1.Job, error) {
	vars := mux.Vars(req)
	id := vars["id"]

	return apiServer.API.GetJob(ctx, id)
}

func (apiServer *DashboardAPIServer) jobProducingInputs(ctx context.Context, req *http.Request) ([]*types.JobRelation, error) {
	vars := mux.Vars(req)
	id := vars["id"]

	return apiServer.API.GetJobsProducingJobInput(ctx, id)
}

func (apiServer *DashboardAPIServer) jobOperatingOnOutputs(ctx context.Context, req *http.Request) ([]*types.JobRelation, error) {
	vars := mux.Vars(req)
	id := vars["id"]

	return apiServer.API.GetJobsOperatingOnJobOutput(ctx, id)
}

func (apiServer *DashboardAPIServer) findJobsOperatingOnCID(ctx context.Context, req *http.Request) ([]*types.JobDataIO, error) {
	vars := mux.Vars(req)
	cid := vars["cid"]

	return apiServer.API.GetJobsOperatingOnCID(ctx, cid)
}

func (apiServer *DashboardAPIServer) jobInfo(ctx context.Context, req *http.Request) (*types.JobInfo, error) {
	vars := mux.Vars(req)
	id := vars["id"]

	return apiServer.API.GetJobInfo(ctx, id)
}

type loginResponse struct {
	Token string `json:"token"`
}

func (apiServer *DashboardAPIServer) adminlogin(ctx context.Context, loginRequest types.LoginRequest) (*loginResponse, error) {
	// decode the request body into a LoginRequest struct
	user, err := apiServer.API.Login(ctx, loginRequest)
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).Str("user", loginRequest.Username).Msg("User authentication failed")
		return nil, err
	}
	token, err := generateJWT(apiServer.Options.JWTSecret, user.Username)
	return &loginResponse{Token: token}, err
}

func (apiServer *DashboardAPIServer) adminstatus(ctx context.Context, req *http.Request) (*types.User, error) {
	return ctx.Value(userContextKey{}).(*types.User), nil
}

type moderateRequest struct {
	Reason   string `json:"reason"`
	Approved bool   `json:"approved"`
}

type moderateResult struct {
	Success bool `json:"success"`
}

func (apiServer *DashboardAPIServer) moderateJobDatacap(ctx context.Context, req *http.Request) (*moderateResult, error) {
	user := ctx.Value(userContextKey{}).(*types.User)
	jobID := mux.Vars(req)["id"]

	data, err := GetRequestBody[moderateRequest](req)
	if err != nil {
		return nil, err
	}

	err = apiServer.API.ModerateJobWithoutRequest(ctx, jobID, data.Reason, data.Approved, types.ModerationTypeDatacap, user)
	return &moderateResult{Success: err == nil}, err
}

func (apiServer *DashboardAPIServer) moderateJobRequest(ctx context.Context, req *http.Request) (*moderateResult, error) {
	user := ctx.Value(userContextKey{}).(*types.User)
	jobID := mux.Vars(req)["id"]

	data, err := GetRequestBody[moderateRequest](req)
	if err != nil {
		return nil, err
	}

	err = apiServer.API.ModerateJobWithoutRequest(ctx, jobID, data.Reason, data.Approved, types.ModerationTypeExecution, user)
	return &moderateResult{Success: err == nil}, err
}
