package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	bacmodel "github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/rs/zerolog/log"

	"github.com/filecoin-project/bacalhau/dashboard/api/pkg/model"
	"github.com/filecoin-project/bacalhau/dashboard/api/pkg/types"
	"github.com/filecoin-project/bacalhau/pkg/localdb"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/gorilla/mux"
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

func (apiServer *DashboardAPIServer) ListenAndServe(ctx context.Context, cm *system.CleanupManager) error {
	router := mux.NewRouter()
	subrouter := router.PathPrefix("/api/v1").Subrouter()
	subrouter.HandleFunc("/nodes", apiServer.nodes).Methods("GET")
	subrouter.HandleFunc("/run", apiServer.run).Methods("POST")
	subrouter.HandleFunc("/stablediffusion", apiServer.stablediffusion).Methods("POST")
	subrouter.HandleFunc("/jobs", apiServer.jobs).Methods("POST")
	subrouter.HandleFunc("/jobs/count", apiServer.jobsCount).Methods("POST")
	subrouter.HandleFunc("/job/{id}", apiServer.job).Methods("GET")
	subrouter.HandleFunc("/job/{id}/info", apiServer.jobInfo).Methods("GET")
	subrouter.HandleFunc("/summary/annotations", apiServer.annotations).Methods("GET")
	subrouter.HandleFunc("/summary/jobmonths", apiServer.jobmonths).Methods("GET")
	subrouter.HandleFunc("/summary/jobexecutors", apiServer.jobexecutors).Methods("GET")
	subrouter.HandleFunc("/summary/totaljobs", apiServer.totaljobs).Methods("GET")
	subrouter.HandleFunc("/summary/totaljobevents", apiServer.totaljobevents).Methods("GET")
	subrouter.HandleFunc("/summary/totalusers", apiServer.totalusers).Methods("GET")
	subrouter.HandleFunc("/summary/totalexecutors", apiServer.totalexecutors).Methods("GET")

	subrouter.HandleFunc("/admin/login", apiServer.adminlogin).Methods("POST")
	subrouter.HandleFunc("/admin/status", apiServer.adminstatus).Methods("GET")
	subrouter.HandleFunc("/admin/moderate", apiServer.adminmoderate).Methods("POST")

	srv := &http.Server{
		Addr:              fmt.Sprintf("%s:%d", apiServer.Options.Host, apiServer.Options.Port),
		WriteTimeout:      time.Minute * 15,
		ReadTimeout:       time.Minute * 15,
		ReadHeaderTimeout: time.Minute * 15,
		IdleTimeout:       time.Minute * 60,
		Handler:           router,
	}
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

func (apiServer *DashboardAPIServer) annotations(res http.ResponseWriter, req *http.Request) {
	data, err := apiServer.API.GetAnnotationSummary(context.Background())
	if err != nil {
		log.Ctx(req.Context()).Error().Msgf("error for annotations route: %s", err.Error())
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(res).Encode(data)
	if err != nil {
		log.Ctx(req.Context()).Error().Msgf("error for annotations route: %s", err.Error())
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (apiServer *DashboardAPIServer) jobmonths(res http.ResponseWriter, req *http.Request) {
	data, err := apiServer.API.GetJobMonthSummary(context.Background())
	if err != nil {
		log.Ctx(req.Context()).Error().Msgf("error for job months route: %s", err.Error())
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(res).Encode(data)
	if err != nil {
		log.Ctx(req.Context()).Error().Msgf("error for job months route: %s", err.Error())
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (apiServer *DashboardAPIServer) jobexecutors(res http.ResponseWriter, req *http.Request) {
	data, err := apiServer.API.GetJobExecutorSummary(context.Background())
	if err != nil {
		log.Ctx(req.Context()).Error().Msgf("error for job executors route: %s", err.Error())
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(res).Encode(data)
	if err != nil {
		log.Ctx(req.Context()).Error().Msgf("error for job executors route: %s", err.Error())
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (apiServer *DashboardAPIServer) totaljobs(res http.ResponseWriter, req *http.Request) {
	data, err := apiServer.API.GetTotalJobsCount(context.Background())
	if err != nil {
		log.Ctx(req.Context()).Error().Msgf("error for job totals route: %s", err.Error())
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(res).Encode(data)
	if err != nil {
		log.Ctx(req.Context()).Error().Msgf("error for job totals route: %s", err.Error())
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (apiServer *DashboardAPIServer) totaljobevents(res http.ResponseWriter, req *http.Request) {
	data, err := apiServer.API.GetTotalEventCount(context.Background())
	if err != nil {
		log.Ctx(req.Context()).Error().Msgf("error for job event totals route: %s", err.Error())
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(res).Encode(data)
	if err != nil {
		log.Ctx(req.Context()).Error().Msgf("error for job event totals route: %s", err.Error())
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (apiServer *DashboardAPIServer) totalusers(res http.ResponseWriter, req *http.Request) {
	data, err := apiServer.API.GetTotalUserCount(context.Background())
	if err != nil {
		log.Ctx(req.Context()).Error().Msgf("error for job user totals route: %s", err.Error())
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(res).Encode(data)
	if err != nil {
		log.Ctx(req.Context()).Error().Msgf("error for job user totals route: %s", err.Error())
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (apiServer *DashboardAPIServer) totalexecutors(res http.ResponseWriter, req *http.Request) {
	data, err := apiServer.API.GetTotalExecutorCount(context.Background())
	if err != nil {
		log.Ctx(req.Context()).Error().Msgf("error for job executors totals route: %s", err.Error())
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(res).Encode(data)
	if err != nil {
		log.Ctx(req.Context()).Error().Msgf("error for job executors totals route: %s", err.Error())
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (apiServer *DashboardAPIServer) nodes(res http.ResponseWriter, req *http.Request) {
	nodes, err := apiServer.API.GetNodes(context.Background())
	if err == nil {
		err = json.NewEncoder(res).Encode(nodes)
	}
	if err != nil {
		log.Ctx(req.Context()).Error().Msgf("error for nodes route: %s", err.Error())
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (apiServer *DashboardAPIServer) jobs(res http.ResponseWriter, req *http.Request) {
	query, err := GetRequestBody[localdb.JobQuery](res, req)
	if err != nil {
		log.Ctx(req.Context()).Error().Msgf("error for jobs route: %s", err.Error())
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	results, err := apiServer.API.GetJobs(context.Background(), *query)
	if err != nil {
		log.Ctx(req.Context()).Error().Msgf("error for jobs route: %s", err.Error())
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	err = json.NewEncoder(res).Encode(results)
	if err != nil {
		log.Ctx(req.Context()).Error().Msgf("error for jobs route: %s", err.Error())
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
}

type jobsCountResponse struct {
	Count int `json:"count"`
}

func (apiServer *DashboardAPIServer) jobsCount(res http.ResponseWriter, req *http.Request) {
	query, err := GetRequestBody[localdb.JobQuery](res, req)
	if err != nil {
		log.Ctx(req.Context()).Error().Msgf("error for jobs route: %s", err.Error())
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	count, err := apiServer.API.GetJobsCount(context.Background(), *query)
	if err != nil {
		log.Ctx(req.Context()).Error().Msgf("error for jobsCount route: %s", err.Error())
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	err = json.NewEncoder(res).Encode(jobsCountResponse{
		Count: count,
	})
	if err != nil {
		log.Ctx(req.Context()).Error().Msgf("error for jobsCount route: %s", err.Error())
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (apiServer *DashboardAPIServer) job(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id := vars["id"]

	data, err := apiServer.API.GetJob(context.Background(), id)
	if err != nil {
		log.Ctx(req.Context()).Error().Msgf("error for job route: %s", err.Error())
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	err = json.NewEncoder(res).Encode(data)
	if err != nil {
		log.Ctx(req.Context()).Error().Msgf("error for job route: %s", err.Error())
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (apiServer *DashboardAPIServer) jobInfo(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id := vars["id"]

	data, err := apiServer.API.GetJobInfo(context.Background(), id)
	if err != nil {
		log.Ctx(req.Context()).Error().Msgf("error for jobInfo route: %s", err.Error())
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	err = json.NewEncoder(res).Encode(data)
	if err != nil {
		log.Ctx(req.Context()).Error().Msgf("error for jobInfo route: %s", err.Error())
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
}

type loginResponse struct {
	Token string `json:"token"`
}

func (apiServer *DashboardAPIServer) adminlogin(res http.ResponseWriter, req *http.Request) {
	// decode the request body into a LoginRequest struct
	var loginRequest types.LoginRequest
	err := json.NewDecoder(req.Body).Decode(&loginRequest)
	if err != nil {
		log.Ctx(req.Context()).Error().Msgf("error for login route: %s", err.Error())
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	user, err := apiServer.API.Login(context.Background(), loginRequest)
	if err != nil {
		log.Ctx(req.Context()).Error().Msgf("error for login route: %s", err.Error())
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	token, err := generateJWT(apiServer.Options.JWTSecret, user.Username)
	if err != nil {
		log.Ctx(req.Context()).Error().Msgf("error for login route: %s", err.Error())
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(res).Encode(loginResponse{
		Token: token,
	})
	if err != nil {
		log.Ctx(req.Context()).Error().Msgf("error for login route: %s", err.Error())
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (apiServer *DashboardAPIServer) adminstatus(res http.ResponseWriter, req *http.Request) {
	user, err := getUserFromRequest(apiServer.API, req, apiServer.Options.JWTSecret)
	if err != nil {
		log.Ctx(req.Context()).Error().Msgf("error for adminstatus route: %s", err.Error())
		http.Error(res, fmt.Sprintf("error for adminstatus route: %s", err.Error()), http.StatusUnauthorized)
		return
	}
	err = json.NewEncoder(res).Encode(user)
	if err != nil {
		log.Ctx(req.Context()).Error().Msgf("error for status route: %s", err.Error())
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (apiServer *DashboardAPIServer) adminmoderate(res http.ResponseWriter, req *http.Request) {
	user, err := getUserFromRequest(apiServer.API, req, apiServer.Options.JWTSecret)
	if err != nil || user == nil {
		log.Ctx(req.Context()).Error().Msgf("access denied: %s", err.Error())
		http.Error(res, fmt.Sprintf("access denied: %s", err.Error()), http.StatusUnauthorized)
		return
	}
	data, err := GetRequestBody[types.JobModeration](res, req)
	if err != nil {
		log.Ctx(req.Context()).Error().Msgf("error for adminmoderate route: %s", err.Error())
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	err = apiServer.API.CreateJobModeration(context.Background(), *data)
	if err != nil {
		log.Ctx(req.Context()).Error().Msgf("error for adminmoderate route: %s", err.Error())
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	err = json.NewEncoder(res).Encode(struct {
		Success bool `json:"success"`
	}{
		Success: true,
	})
	if err != nil {
		log.Ctx(req.Context()).Error().Msgf("error for adminmoderate route: %s", err.Error())
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
}
