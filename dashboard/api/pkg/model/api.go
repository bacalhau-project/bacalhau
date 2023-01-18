package model

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/filecoin-project/bacalhau/dashboard/api/pkg/store"
	"github.com/filecoin-project/bacalhau/dashboard/api/pkg/types"
	jobutils "github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/localdb"
	"github.com/filecoin-project/bacalhau/pkg/localdb/postgres"
	bacalhau_model "github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/requester"
	"github.com/filecoin-project/bacalhau/pkg/requester/nodestore"
	"golang.org/x/crypto/bcrypt"
)

type ModelOptions struct {
	UpstreamHost     string
	UpstreamPort     int
	PostgresHost     string
	PostgresPort     int
	PostgresDatabase string
	PostgresUser     string
	PostgresPassword string
}

type ModelAPI struct {
	options          ModelOptions
	localDB          localdb.LocalDB
	nodeDB           requester.NodeInfoStore
	store            *store.PostgresStore
	stateResolver    *jobutils.StateResolver
	jobEventHandler  *jobEventHandler
	nodeEventHandler *nodeEventHandler
}

func NewModelAPI(options ModelOptions) (*ModelAPI, error) {
	if options.UpstreamHost == "" {
		return nil, fmt.Errorf("upstream host is required")
	}
	if options.UpstreamPort == 0 {
		return nil, fmt.Errorf("upstream port is required")
	}
	if options.PostgresHost == "" {
		return nil, fmt.Errorf("postgres host is required")
	}
	if options.PostgresPort == 0 {
		return nil, fmt.Errorf("postgres port is required")
	}
	if options.PostgresDatabase == "" {
		return nil, fmt.Errorf("postgres database is required")
	}
	if options.PostgresUser == "" {
		return nil, fmt.Errorf("postgres user is required")
	}
	if options.PostgresPassword == "" {
		return nil, fmt.Errorf("postgres password is required")
	}
	postgresDB, err := postgres.NewPostgresDatastore(
		options.PostgresHost,
		options.PostgresPort,
		options.PostgresDatabase,
		options.PostgresUser,
		options.PostgresPassword,
		true,
	)
	if err != nil {
		return nil, err
	}
	dashboardstore, err := store.NewPostgresStore(
		options.PostgresHost,
		options.PostgresPort,
		options.PostgresDatabase,
		options.PostgresUser,
		options.PostgresPassword,
		true,
	)
	if err != nil {
		return nil, err
	}

	nodeDB := nodestore.NewInMemoryNodeInfoStore(nodestore.InMemoryNodeInfoStoreParams{
		// compute nodes publish every 30 seconds. We give a graceful period of 2 minutes for them to be considered offline
		TTL: 2 * time.Minute,
	})

	jobEventHandler, err := newJobEventHandler(
		options.UpstreamHost,
		options.UpstreamPort,
		postgresDB,
	)
	if err != nil {
		return nil, err
	}

	nodeEventHandler, err := newNodeEventHandler(
		options.UpstreamHost,
		options.UpstreamPort,
		nodeDB,
	)
	if err != nil {
		return nil, err
	}

	stateResolver := localdb.GetStateResolver(postgresDB)

	api := &ModelAPI{
		options:          options,
		localDB:          postgresDB,
		nodeDB:           nodeDB,
		store:            dashboardstore,
		stateResolver:    stateResolver,
		jobEventHandler:  jobEventHandler,
		nodeEventHandler: nodeEventHandler,
	}
	return api, nil
}

func (api *ModelAPI) Start(ctx context.Context) {
	api.jobEventHandler.start(ctx)
	api.nodeEventHandler.start(ctx)
}

func (api *ModelAPI) GetNodes(ctx context.Context) (map[string]bacalhau_model.NodeInfo, error) {
	nodesList, err := api.nodeDB.List(ctx)
	if err != nil {
		return nil, err
	}
	nodesMap := make(map[string]bacalhau_model.NodeInfo, len(nodesList))
	for _, node := range nodesList {
		if node.NodeType == bacalhau_model.NodeTypeCompute {
			nodesMap[node.PeerInfo.ID.String()] = node
		}
	}
	return nodesMap, nil
}

func (api *ModelAPI) GetJobs(ctx context.Context, query localdb.JobQuery) ([]*bacalhau_model.Job, error) {
	return api.localDB.GetJobs(context.Background(), query)
}

func (api *ModelAPI) GetJobsCount(ctx context.Context, query localdb.JobQuery) (int, error) {
	return api.localDB.GetJobsCount(context.Background(), query)
}

func (api *ModelAPI) GetJob(ctx context.Context, id string) (*bacalhau_model.Job, error) {
	return api.localDB.GetJob(ctx, id)
}

func (api *ModelAPI) GetJobInfo(ctx context.Context, id string) (*types.JobInfo, error) {
	info := &types.JobInfo{}

	job, err := api.localDB.GetJob(ctx, id)
	if err != nil {
		return nil, err
	}
	info.Job = *job

	// they might have asked for a short job ID so if we found a job
	// let's use that for subsequent queries
	loadedID := job.Metadata.ID

	errorChan := make(chan error, 1)
	doneChan := make(chan bool, 1)
	var wg sync.WaitGroup
	//nolint:gomnd
	wg.Add(4)
	go func() {
		events, err := api.localDB.GetJobEvents(ctx, loadedID)
		if err != nil {
			errorChan <- err
		}
		info.Events = events
		wg.Done()
	}()
	go func() {
		state, err := api.stateResolver.GetJobState(ctx, loadedID)
		if err != nil {
			errorChan <- err
		}
		info.State = state
		wg.Done()
	}()
	go func() {
		results, err := api.stateResolver.GetResults(ctx, loadedID)
		if err != nil {
			errorChan <- err
		}
		info.Results = results
		wg.Done()
	}()
	go func() {
		results, err := api.GetModerationSummary(ctx, loadedID)
		if err != nil {
			errorChan <- err
		}
		info.Moderation = *results
		wg.Done()
	}()
	go func() {
		wg.Wait()
		doneChan <- true
	}()
	select {
	case <-doneChan:
		return info, nil
	case err := <-errorChan:
		return nil, err
	}
}

func (api *ModelAPI) GetAnnotationSummary(
	ctx context.Context,
) ([]*types.AnnotationSummary, error) {
	return api.store.GetAnnotationSummary(ctx)
}

func (api *ModelAPI) GetJobMonthSummary(
	ctx context.Context,
) ([]*types.JobMonthSummary, error) {
	return api.store.GetJobMonthSummary(ctx)
}

func (api *ModelAPI) GetJobExecutorSummary(
	ctx context.Context,
) ([]*types.JobExecutorSummary, error) {
	return api.store.GetJobExecutorSummary(ctx)
}

func (api *ModelAPI) GetTotalJobsCount(
	ctx context.Context,
) (*types.Counter, error) {
	return api.store.GetTotalJobsCount(ctx)
}

func (api *ModelAPI) GetTotalEventCount(
	ctx context.Context,
) (*types.Counter, error) {
	return api.store.GetTotalEventCount(ctx)
}

func (api *ModelAPI) GetTotalUserCount(
	ctx context.Context,
) (*types.Counter, error) {
	return api.store.GetTotalUserCount(ctx)
}

func (api *ModelAPI) GetTotalExecutorCount(
	ctx context.Context,
) (*types.Counter, error) {
	return api.store.GetTotalExecutorCount(ctx)
}

func (api *ModelAPI) AddEvent(event bacalhau_model.JobEvent) {
	api.jobEventHandler.readEvent(context.Background(), event)
}

func (api *ModelAPI) AddUser(
	ctx context.Context,
	username string,
	password string,
) (*types.User, error) {
	hashedPassword, err := hashPassword(password)
	if err != nil {
		return nil, err
	}
	err = api.store.AddUser(ctx, username, hashedPassword)
	if err != nil {
		return nil, err
	}
	return api.store.LoadUser(ctx, username)
}

func (api *ModelAPI) GetUser(
	ctx context.Context,
	username string,
) (*types.User, error) {
	return api.store.LoadUser(ctx, username)
}

func (api *ModelAPI) UpdateUserPassword(
	ctx context.Context,
	username string,
	password string,
) (*types.User, error) {
	user, err := api.store.LoadUser(ctx, username)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}
	hashedPassword, err := hashPassword(password)
	if err != nil {
		return nil, err
	}
	err = api.store.UpdateUserPassword(ctx, username, hashedPassword)
	if err != nil {
		return nil, err
	}
	return api.store.LoadUser(ctx, username)
}

func (api *ModelAPI) Login(
	ctx context.Context,
	req types.LoginRequest,
) (*types.User, error) {
	user, err := api.store.LoadUser(ctx, req.Username)
	if err != nil || user == nil {
		return nil, fmt.Errorf("incorrect details")
	}
	err = bcrypt.CompareHashAndPassword([]byte(user.HashedPassword), []byte(req.Password))
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (api *ModelAPI) GetModerationSummary(
	ctx context.Context,
	jobID string,
) (*types.JobModerationSummary, error) {
	moderation, err := api.store.GetJobModeration(ctx, jobID)
	if err != nil {
		return nil, err
	}
	if moderation == nil {
		return &types.JobModerationSummary{}, nil
	}
	user, err := api.store.LoadUserByID(ctx, moderation.UserAccountID)
	if err != nil {
		return nil, err
	}
	user.HashedPassword = ""
	return &types.JobModerationSummary{
		Moderation: moderation,
		User:       user,
	}, nil
}

func (api *ModelAPI) CreateJobModeration(
	ctx context.Context,
	moderation types.JobModeration,
) error {
	return api.store.CreateJobModeration(ctx, moderation)
}
