package devstack

import (
	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/rs/zerolog"

	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/routing"
)

type ConfigOption = func(cfg *DevStackConfig)

func defaultDevStackConfig() (*DevStackConfig, error) {
	computeConfig, err := node.NewComputeConfigWithDefaults()
	requesterConfig, err := node.NewRequesterConfigWithDefaults()
	if err != nil {
		return nil, err
	}
	return &DevStackConfig{
		ComputeConfig:          computeConfig,
		RequesterConfig:        requesterConfig,
		NodeDependencyInjector: node.NodeDependencyInjector{},
		NodeOverrides:          nil,

		NumberOfRequesterOnlyNodes: 1,
		NumberOfComputeOnlyNodes:   3,
		NumberOfBadComputeActors:   0,
		Peer:                       "",
		PublicIPFSMode:             false,
		CPUProfilingFile:           "",
		MemoryProfilingFile:        "",
		NodeInfoPublisherInterval:  node.TestNodeInfoPublishConfig,

		NumberOfBadRequesterActors: 0,
		NumberOfHybridNodes:        0,
		DisabledFeatures:           node.FeatureConfig{},
		AllowListedLocalPaths:      nil,
		ExecutorPlugins:            false,
	}, nil
}

type DevStackConfig struct {
	ComputeConfig          node.ComputeConfig
	RequesterConfig        node.RequesterConfig
	NodeDependencyInjector node.NodeDependencyInjector
	NodeOverrides          []node.NodeConfig

	// DevStackOptions
	NumberOfHybridNodes        int    // Number of nodes to start in the cluster
	NumberOfRequesterOnlyNodes int    // Number of nodes to start in the cluster
	NumberOfComputeOnlyNodes   int    // Number of nodes to start in the cluster
	NumberOfBadComputeActors   int    // Number of compute nodes to be bad actors
	NumberOfBadRequesterActors int    // Number of requester nodes to be bad actors
	Peer                       string // Connect node 0 to another network node
	PublicIPFSMode             bool   // Use public IPFS nodes
	CPUProfilingFile           string
	MemoryProfilingFile        string
	DisabledFeatures           node.FeatureConfig
	AllowListedLocalPaths      []string // Local paths that are allowed to be mounted into jobs
	NodeInfoPublisherInterval  routing.NodeInfoPublisherIntervalConfig
	ExecutorPlugins            bool // when true pluggable executors will be used.
}

func (o *DevStackConfig) MarshalZerologObject(e *zerolog.Event) {
	e.Int("HybridNodes", o.NumberOfHybridNodes).
		Int("RequesterOnlyNodes", o.NumberOfRequesterOnlyNodes).
		Int("ComputeOnlyNodes", o.NumberOfComputeOnlyNodes).
		Int("BadComputeActors", o.NumberOfBadComputeActors).
		Int("BadRequesterActors", o.NumberOfBadRequesterActors).
		Str("Peer", o.Peer).
		Str("CPUProfilingFile", o.CPUProfilingFile).
		Str("MemoryProfilingFile", o.MemoryProfilingFile).
		Str("DisabledFeatures", fmt.Sprintf("%v", o.DisabledFeatures)).
		Strs("AllowListedLocalPaths", o.AllowListedLocalPaths).
		Str("NodeInfoPublisherInterval", fmt.Sprintf("%v", o.NodeInfoPublisherInterval)).
		Bool("PublicIPFSMode", o.PublicIPFSMode).
		Bool("ExecutorPlugins", o.ExecutorPlugins)
}

func (o *DevStackConfig) Validate() error {
	errs := new(multierror.Error)
	totalNodeCount := o.NumberOfHybridNodes + o.NumberOfRequesterOnlyNodes + o.NumberOfComputeOnlyNodes

	if totalNodeCount == 0 {
		errs = multierror.Append(errs, fmt.Errorf("you cannot create a devstack with zero nodes"))
	}

	totalComputeNodes := o.NumberOfComputeOnlyNodes + o.NumberOfHybridNodes
	if o.NumberOfBadComputeActors > totalComputeNodes {
		errs = multierror.Append(errs,
			fmt.Errorf("you cannot have more bad compute actors (%d) than there are nodes (%d)",
				o.NumberOfBadComputeActors, totalComputeNodes))
	}

	totalRequesterNodes := o.NumberOfRequesterOnlyNodes + o.NumberOfHybridNodes
	if o.NumberOfBadRequesterActors > totalRequesterNodes {
		errs = multierror.Append(errs,
			fmt.Errorf("you cannot have more bad requester actors (%d) than there are nodes (%d)",
				o.NumberOfBadRequesterActors, totalRequesterNodes))
	}

	return errs.ErrorOrNil()
}

func WithNodeOverrides(overrides ...node.NodeConfig) ConfigOption {
	return func(cfg *DevStackConfig) {
		cfg.NodeOverrides = overrides
	}
}

func WithDependencyInjector(injector node.NodeDependencyInjector) ConfigOption {
	return func(cfg *DevStackConfig) {
		cfg.NodeDependencyInjector = injector
	}
}

func WithComputeConfig(computeCfg node.ComputeConfig) ConfigOption {
	return func(cfg *DevStackConfig) {
		cfg.ComputeConfig = computeCfg
	}
}

func WithRequesterConfig(requesterConfig node.RequesterConfig) ConfigOption {
	return func(cfg *DevStackConfig) {
		cfg.RequesterConfig = requesterConfig
	}
}

func WithNumberOfHybridNodes(count int) ConfigOption {
	return func(cfg *DevStackConfig) {
		cfg.NumberOfHybridNodes = count
	}
}

func WithPublicIPFSMode(enabled bool) ConfigOption {
	return func(cfg *DevStackConfig) {
		cfg.PublicIPFSMode = enabled
	}
}

func WithNumberOfRequesterOnlyNodes(count int) ConfigOption {
	return func(cfg *DevStackConfig) {
		cfg.NumberOfRequesterOnlyNodes = count
	}
}

func WithNumberOfComputeOnlyNodes(count int) ConfigOption {
	return func(cfg *DevStackConfig) {
		cfg.NumberOfComputeOnlyNodes = count
	}
}

func WithNumberOfBadComputeActors(count int) ConfigOption {
	return func(cfg *DevStackConfig) {
		cfg.NumberOfBadComputeActors = count
	}
}

func WithNumberOfBadRequesterActors(count int) ConfigOption {
	return func(cfg *DevStackConfig) {
		cfg.NumberOfBadRequesterActors = count
	}
}

func WithPeer(p string) ConfigOption {
	return func(cfg *DevStackConfig) {
		cfg.Peer = p
	}
}

func WithCPUProfilingFile(path string) ConfigOption {
	return func(cfg *DevStackConfig) {
		cfg.CPUProfilingFile = path
	}
}

func WithMemoryProfilingFile(path string) ConfigOption {
	return func(cfg *DevStackConfig) {
		cfg.MemoryProfilingFile = path
	}
}

func WithDisabledFeatures(disable node.FeatureConfig) ConfigOption {
	return func(cfg *DevStackConfig) {
		cfg.DisabledFeatures = disable
	}
}

func WithAllowListedLocalPaths(paths []string) ConfigOption {
	return func(cfg *DevStackConfig) {
		cfg.AllowListedLocalPaths = paths
	}
}

func WithNodeInfoPublisherInterval(interval routing.NodeInfoPublisherIntervalConfig) ConfigOption {
	return func(cfg *DevStackConfig) {
		cfg.NodeInfoPublisherInterval = interval
	}
}

func WithExecutorPlugins(enabled bool) ConfigOption {
	return func(cfg *DevStackConfig) {
		cfg.ExecutorPlugins = enabled
	}
}
