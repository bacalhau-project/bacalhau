package model

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"go.ptx.dk/multierrgroup"

	"github.com/bacalhau-project/bacalhau/dashboard/api/pkg/store"
	"github.com/bacalhau-project/bacalhau/dashboard/api/pkg/types"
	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/localdb"
	"github.com/bacalhau-project/bacalhau/pkg/localdb/postgres"
	bacalhau_model "github.com/bacalhau-project/bacalhau/pkg/model"
	bacalhau_model_beta "github.com/bacalhau-project/bacalhau/pkg/model/v1beta1"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/util"

	libp2p_pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/exp/slices"

	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/pubsub"
	"github.com/bacalhau-project/bacalhau/pkg/pubsub/libp2p"
	"github.com/bacalhau-project/bacalhau/pkg/routing"
	"github.com/bacalhau-project/bacalhau/pkg/routing/inmemory"
)

type ModelOptions struct {
	Libp2pHost       host.Host
	PostgresHost     string
	PostgresPort     int
	PostgresDatabase string
	PostgresUser     string
	PostgresPassword string
	SelectionPolicy  bacalhau_model.JobSelectionPolicy
}

type ModelAPI struct {
	options         ModelOptions
	localDB         localdb.LocalDB
	nodeDB          routing.NodeInfoStore
	store           *store.PostgresStore
	stateResolver   *localdb.StateResolver
	jobEventHandler *jobEventHandler
	jobSelector     bidstrategy.BidStrategy
	cleanupFunc     func(context.Context)
}

func NewModelAPI(options ModelOptions) (*ModelAPI, error) {
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

	nodeDB := inmemory.NewNodeInfoStore(inmemory.NodeInfoStoreParams{
		// compute nodes publish every 30 seconds. We give a graceful period of 2 minutes for them to be considered offline
		TTL: 2 * time.Minute,
	})

	stateResolver := localdb.GetStateResolver(postgresDB)

	// Allow good jobs to be processed immediately but hold bad jobs for moderation.
	jobSelector := bidstrategy.NewWaitingStrategy(
		bidstrategy.FromJobSelectionPolicy(options.SelectionPolicy),
		false,
		true,
	)

	api := &ModelAPI{
		options:         options,
		localDB:         postgresDB,
		nodeDB:          nodeDB,
		store:           dashboardstore,
		stateResolver:   stateResolver,
		jobSelector:     jobSelector,
		jobEventHandler: newJobEventHandler(postgresDB),
	}
	return api, nil
}

func (api *ModelAPI) Start(ctx context.Context) error {
	if api.options.Libp2pHost == nil {
		return fmt.Errorf("libp2p host is required")
	}
	var err error
	ctx, cancel := context.WithCancel(ctx)
	defer func() {
		if err != nil {
			cancel()
		}
	}()

	gossipSub, err := libp2p_pubsub.NewGossipSub(ctx, api.options.Libp2pHost)
	if err != nil {
		return err
	}

	// PubSub to read node info from the network
	log.Debug().Str("Topic", node.NodeInfoTopic).Msg("Subscribing")
	nodeInfoPubSub, err := libp2p.NewPubSub[bacalhau_model.NodeInfo](libp2p.PubSubParams{
		Host:      api.options.Libp2pHost,
		TopicName: node.NodeInfoTopic,
		PubSub:    gossipSub,
	})
	if err != nil {
		return err
	}
	err = nodeInfoPubSub.Subscribe(ctx, pubsub.SubscriberFunc[bacalhau_model.NodeInfo](api.nodeDB.Add))
	if err != nil {
		return err
	}

	// PubSub to read job events from the network
	log.Debug().Str("Topic", node.JobEventsTopic).Msg("Subscribing")
	libp2p2JobEventPubSub, err := libp2p.NewPubSub[pubsub.BufferingEnvelope](libp2p.PubSubParams{
		Host:      api.options.Libp2pHost,
		TopicName: node.JobEventsTopic,
		PubSub:    gossipSub,
	})
	if err != nil {
		return err
	}

	bufferedJobEventPubSub := pubsub.NewBufferingPubSub[bacalhau_model_beta.JobEvent](pubsub.BufferingPubSubParams{
		DelegatePubSub: libp2p2JobEventPubSub,
		MaxBufferAge:   5 * time.Minute, //nolint:gomnd // required, but we don't publish events in the dashboard
	})
	err = bufferedJobEventPubSub.Subscribe(ctx, pubsub.SubscriberFunc[bacalhau_model_beta.JobEvent](api.jobEventHandler.readEvent))
	if err != nil {
		return err
	}

	api.jobEventHandler.startBufferGC(ctx)
	api.cleanupFunc = func(ctx context.Context) {
		cleanupErr := bufferedJobEventPubSub.Close(ctx)
		util.LogDebugIfContextCancelled(ctx, cleanupErr, "job event pubsub")
		cleanupErr = libp2p2JobEventPubSub.Close(ctx)
		util.LogDebugIfContextCancelled(ctx, cleanupErr, "job event pubsub")
		cleanupErr = nodeInfoPubSub.Close(ctx)
		util.LogDebugIfContextCancelled(ctx, cleanupErr, "node info pubsub")
		cancel()
	}
	return nil
}

func (api *ModelAPI) Stop(ctx context.Context) error {
	if api.cleanupFunc != nil {
		api.cleanupFunc(ctx)
	}
	return nil
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

func (api *ModelAPI) GetJobsProducingJobInput(ctx context.Context, id string) ([]*types.JobRelation, error) {
	return api.store.GetJobsProducingJobInput(ctx, id)
}

func (api *ModelAPI) GetJobsOperatingOnJobOutput(ctx context.Context, id string) ([]*types.JobRelation, error) {
	return api.store.GetJobsOperatingOnJobOutput(ctx, id)
}

func (api *ModelAPI) GetJobsOperatingOnCID(ctx context.Context, cid string) ([]*types.JobDataIO, error) {
	return api.store.GetJobsOperatingOnCID(ctx, cid)
}

func (api *ModelAPI) GetJobs(ctx context.Context, query localdb.JobQuery) ([]*bacalhau_model_beta.Job, error) {
	return api.localDB.GetJobs(ctx, query)
}

func (api *ModelAPI) GetJobsCount(ctx context.Context, query localdb.JobQuery) (int, error) {
	return api.localDB.GetJobsCount(ctx, query)
}

func (api *ModelAPI) GetJob(ctx context.Context, id string) (*bacalhau_model_beta.Job, error) {
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

	var wg multierrgroup.Group
	wg.Go(func() (err error) {
		info.Events, err = api.localDB.GetJobEvents(ctx, loadedID)
		return
	})
	wg.Go(func() (err error) {
		info.State, err = api.stateResolver.GetJobState(ctx, loadedID)
		return
	})
	wg.Go(func() (err error) {
		info.Results, err = api.stateResolver.GetResults(ctx, loadedID)
		return
	})
	wg.Go(func() (err error) {
		info.Moderations, err = api.store.GetJobModerations(ctx, loadedID)
		return
	})
	wg.Go(func() (err error) {
		info.Requests, err = api.store.GetModerationRequestsForJob(ctx, loadedID)
		return
	})

	err = wg.Wait()
	return info, err
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

func (api *ModelAPI) AddEvent(event bacalhau_model_beta.JobEvent) error {
	return api.jobEventHandler.readEvent(context.Background(), event)
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

// Response returned to signal that a job will be moderated.
var waitResponse = bidstrategy.BidStrategyResponse{
	ShouldWait: true,
	Reason:     "Awaiting human approval",
}

func (api *ModelAPI) ShouldExecuteJob(
	ctx context.Context,
	probe *bidstrategy.JobSelectionPolicyProbeData,
) (*bidstrategy.BidStrategyResponse, error) {
	// Do we have an approval for the job? If so, return it.
	resp, err := api.store.GetJobModerations(ctx, probe.JobID)
	idx := slices.IndexFunc(resp, func(moderation types.JobModerationSummary) bool {
		return moderation.Request.Type == types.ModerationTypeExecution
	})
	if err != nil {
		return nil, err
	} else if idx >= 0 {
		return api.shouldExecuteFromJobModeration(resp[idx].Moderation), nil
	}

	// No approval found. Is there an approval request?
	request, err := api.store.GetModerationRequestByJob(ctx, probe.JobID, types.ModerationTypeExecution)
	if err == nil && request != nil {
		// There is an open request – we are just waiting for it to be handled.
		return &waitResponse, err
	}

	// No request. Firstly let's run our selection strategy.
	bidResponse, err := api.jobSelector.ShouldBid(ctx, bidstrategy.BidStrategyRequest{
		NodeID: probe.NodeID,
		Job: bacalhau_model.Job{
			Metadata: bacalhau_model.Metadata{ID: probe.JobID},
			Spec:     probe.Spec,
		},
		Callback: probe.Callback,
	})
	if !bidResponse.ShouldWait {
		// We can respond immediately.
		return &bidResponse, err
	}

	// Our own strategy says this must be moderated.
	// TODO: we should probably reject requests for jobs that are already running.
	var callback *types.URL
	if probe.Callback != nil {
		callback = &types.URL{URL: *probe.Callback}
	}
	_, err = api.store.CreateJobModerationRequest(ctx, probe.JobID, types.ModerationTypeExecution, callback)
	return &waitResponse, err
}

func (api *ModelAPI) shouldExecuteFromJobModeration(resp *types.JobModeration) *bidstrategy.BidStrategyResponse {
	return &bidstrategy.BidStrategyResponse{
		ShouldBid:  resp.Status,
		ShouldWait: false,
		Reason:     resp.Notes,
	}
}

func (api *ModelAPI) ModerateJob(
	ctx context.Context,
	requestID int64,
	reason string,
	approved bool,
	user *types.User,
) error {
	moderation := types.JobModeration{
		RequestID:     requestID,
		UserAccountID: user.ID,
		Status:        approved,
		Notes:         reason,
	}
	err := api.store.CreateJobModeration(ctx, moderation)
	if err != nil {
		return err
	}

	request, err := api.store.GetModerationRequest(ctx, requestID)
	if err != nil {
		return err
	}

	if request.Callback.IsAbs() {
		log.Ctx(ctx).Debug().Stringer("Callback", &request.Callback.URL).Msg("Returning moderation response")

		req := bidstrategy.ModerateJobRequest{
			ClientID: system.GetClientID(),
			JobID:    request.JobID,
			Response: *api.shouldExecuteFromJobModeration(&moderation),
		}

		port, err := strconv.ParseUint(request.Callback.Port(), 10, 16)
		if err != nil {
			return err
		}

		client := publicapi.NewAPIClient(request.Callback.Hostname(), uint16(port))
		return errors.Wrap(client.PostSigned(ctx, request.Callback.RequestURI(), req, nil), "response from callback")
	}

	return nil
}

func (api *ModelAPI) ModerateJobWithoutRequest(
	ctx context.Context,
	jobID, reason string,
	approved bool,
	moderationType types.ModerationType,
	user *types.User,
) error {
	// Do we have a moderation request for this already?
	request, err := api.store.GetModerationRequestByJob(ctx, jobID, moderationType)
	if err != nil {
		return err
	} else if request == nil {
		// No request. Create one.
		request, err = api.store.CreateJobModerationRequest(ctx, jobID, moderationType, nil)
		if err != nil {
			return err
		}
	}

	return api.ModerateJob(ctx, request.ID, reason, approved, user)
}
