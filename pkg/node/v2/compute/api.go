package compute

import (
	"net/url"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/resource"
	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/semantic"
	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/compute/logstream"
	"github.com/bacalhau-project/bacalhau/pkg/compute/sensors"
	v2 "github.com/bacalhau-project/bacalhau/pkg/config/types/v2"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	modelsutils "github.com/bacalhau-project/bacalhau/pkg/models/utils"
	nats_transport "github.com/bacalhau-project/bacalhau/pkg/nats/transport"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi"
	compute_endpoint "github.com/bacalhau-project/bacalhau/pkg/publicapi/endpoint/compute"
	"github.com/bacalhau-project/bacalhau/pkg/publisher"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
)

type EndpointProvider interface {
	Compute() *compute_endpoint.Endpoint
	Bidding() compute.BaseEndpoint
}

func NewComputeEndpointProvider(
	name string,
	transport *nats_transport.NATSTransport,
	server *publicapi.Server,
	cfg v2.SelectionPolicy,
	storages storage.StorageProvider,
	engines executor.ExecutorProvider,
	publishers publisher.PublisherProvider,
	executor ExecutorProvider,
	capacity CapacityProvider,
) (*ComputeEndpointProvider, error) {

	bidder := compute.NewBidder(compute.BidderParams{
		NodeID:           name,
		Callback:         transport.CallbackProxy(),
		Store:            executor.Store(),
		Executor:         executor.Executor(),
		UsageCalculator:  capacity.Calculator(),
		GetApproveURL:    func() *url.URL { return server.GetURI().JoinPath("/api/v1/compute/approve") },
		SemanticStrategy: setupSemanticBidStrategies(cfg, storages, engines, publishers),
		ResourceStrategy: setupResourceBidStrategies(capacity),
	})

	logserver := logstream.NewServer(logstream.ServerParams{
		ExecutionStore: executor.Store(),
		Executors:      engines,
		// NB(forrest) this is taken from a default value
		Buffer: 10,
	})

	baseEndpoint := compute.NewBaseEndpoint(compute.BaseEndpointParams{
		ID:              name,
		UsageCalculator: capacity.Calculator(),
		Executor:        executor.Executor(),
		ExecutionStore:  executor.Store(),
		Bidder:          bidder,
		LogServer:       logserver,
	})

	debugInfoProviders := []models.DebugInfoProvider{
		executor.DebugProvider(),
		sensors.NewCompletedJobs(executor.Store()),
	}

	// register compute public http apis
	computeEndpoint := compute_endpoint.NewEndpoint(compute_endpoint.EndpointParams{
		Router:             server.Router,
		Bidder:             bidder,
		Store:              executor.Store(),
		DebugInfoProviders: debugInfoProviders,
	})

	return &ComputeEndpointProvider{
		compute: computeEndpoint,
		bidding: baseEndpoint,
	}, nil

}

func setupResourceBidStrategies(provider CapacityProvider) []bidstrategy.ResourceBidStrategy {
	return []bidstrategy.ResourceBidStrategy{
		resource.NewMaxCapacityStrategy(resource.MaxCapacityStrategyParams{
			MaxJobRequirements: provider.Capacity(),
		}),
	}
}

func setupSemanticBidStrategies(
	cfg v2.SelectionPolicy,
	storages storage.StorageProvider,
	executors executor.ExecutorProvider,
	publishers publisher.PublisherProvider,
) []bidstrategy.SemanticBidStrategy {
	localityPolicy := semantic.Anywhere
	if cfg.Local {
		localityPolicy = semantic.Local
	}
	return []bidstrategy.SemanticBidStrategy{
		semantic.NewProviderInstalledStrategy(
			publishers,
			func(j *models.Job) string { return j.Task().Publisher.Type },
		),
		semantic.NewProviderInstalledStrategy(
			executors,
			func(j *models.Job) string { return j.Task().Engine.Type },
		),
		semantic.NewProviderInstalledArrayStrategy(
			storages,
			func(j *models.Job) []string {
				return modelsutils.AllInputSourcesTypes(j)
			},
		),
		semantic.NewInputLocalityStrategy(semantic.InputLocalityStrategyParams{
			Locality: localityPolicy,
			Storages: storages,
		}),
		semantic.NewNetworkingStrategy(cfg.Networked),
	}
	// NB(forrest): we have stated we are discarding these configurations and thus policies:
	// https://www.notion.so/expanso/Rethinking-Configuration-435fbe87419148b4bbc5119d413786eb?pvs=4#106588c3e3c94c8191e1983084e00f0f
	/*
		semantic.NewTimeoutStrategy(semantic.TimeoutStrategyParams{
			MaxJobExecutionTimeout:                config.MaxJobExecutionTimeout,
			MinJobExecutionTimeout:                config.MinJobExecutionTimeout,
			JobExecutionTimeoutClientIDBypassList: config.JobExecutionTimeoutClientIDBypassList,
		}),
		semantic.NewStatelessJobStrategy(semantic.StatelessJobStrategyParams{
			RejectStatelessJobs: config.JobSelectionPolicy.RejectStatelessJobs,
		}),
		semantic.NewExternalCommandStrategy(semantic.ExternalCommandStrategyParams{
			Command: config.JobSelectionPolicy.ProbeExec,
		}),
		semantic.NewExternalHTTPStrategy(semantic.ExternalHTTPStrategyParams{
			URL: config.JobSelectionPolicy.ProbeHTTP,
		}),

	*/
}

type ComputeEndpointProvider struct {
	compute *compute_endpoint.Endpoint
	bidding compute.BaseEndpoint
}

func (c *ComputeEndpointProvider) Compute() *compute_endpoint.Endpoint {
	return c.compute
}

func (c *ComputeEndpointProvider) Bidding() compute.BaseEndpoint {
	return c.bidding
}
