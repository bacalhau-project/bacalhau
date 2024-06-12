package devstack

import (
	"errors"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/node"
)

type ConfigOption = func(cfg *DevStackConfig)

func defaultDevStackConfig(cfg types.BacalhauConfig) (*DevStackConfig, error) {
	computeConfig, err := node.NewComputeConfigWithDefaults(cfg.Node.ComputeStoragePath)
	if err != nil {
		return nil, err
	}

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
		CPUProfilingFile:           "",
		MemoryProfilingFile:        "",

		NumberOfBadRequesterActors: 0,
		NumberOfHybridNodes:        0,
		DisabledFeatures:           node.FeatureConfig{},
		AllowListedLocalPaths:      nil,
		ExecutorPlugins:            false,
	}, nil
}

type DevstackTLSSettings struct {
	Certificate string
	Key         string
}

type DevStackConfig struct {
	ComputeConfig          node.ComputeConfig
	RequesterConfig        node.RequesterConfig
	NodeDependencyInjector node.NodeDependencyInjector
	NodeOverrides          []node.NodeConfig

	// DevStackOptions
	NumberOfHybridNodes        int // Number of nodes to start in the cluster
	NumberOfRequesterOnlyNodes int // Number of nodes to start in the cluster
	NumberOfComputeOnlyNodes   int // Number of nodes to start in the cluster
	NumberOfBadComputeActors   int // Number of compute nodes to be bad actors
	NumberOfBadRequesterActors int // Number of requester nodes to be bad actors
	CPUProfilingFile           string
	MemoryProfilingFile        string
	DisabledFeatures           node.FeatureConfig
	AllowListedLocalPaths      []string // Local paths that are allowed to be mounted into jobs
	ExecutorPlugins            bool     // when true pluggable executors will be used.
	TLS                        DevstackTLSSettings
	AuthSecret                 string
}

func (o *DevStackConfig) MarshalZerologObject(e *zerolog.Event) {
	e.Int("HybridNodes", o.NumberOfHybridNodes).
		Int("RequesterOnlyNodes", o.NumberOfRequesterOnlyNodes).
		Int("ComputeOnlyNodes", o.NumberOfComputeOnlyNodes).
		Int("BadComputeActors", o.NumberOfBadComputeActors).
		Int("BadRequesterActors", o.NumberOfBadRequesterActors).
		Str("CPUProfilingFile", o.CPUProfilingFile).
		Str("MemoryProfilingFile", o.MemoryProfilingFile).
		Str("DisabledFeatures", fmt.Sprintf("%v", o.DisabledFeatures)).
		Strs("AllowListedLocalPaths", o.AllowListedLocalPaths).
		Bool("ExecutorPlugins", o.ExecutorPlugins)
}

func (o *DevStackConfig) Validate() error {
	var errs error
	totalNodeCount := o.NumberOfHybridNodes + o.NumberOfRequesterOnlyNodes + o.NumberOfComputeOnlyNodes

	if totalNodeCount == 0 {
		errs = errors.Join(errs, fmt.Errorf("you cannot create a devstack with zero nodes"))
	}

	totalComputeNodes := o.NumberOfComputeOnlyNodes + o.NumberOfHybridNodes
	if o.NumberOfBadComputeActors > totalComputeNodes {
		errs = errors.Join(errs,
			fmt.Errorf("you cannot have more bad compute actors (%d) than there are nodes (%d)",
				o.NumberOfBadComputeActors, totalComputeNodes))
	}

	totalRequesterNodes := o.NumberOfRequesterOnlyNodes + o.NumberOfHybridNodes
	if o.NumberOfBadRequesterActors > totalRequesterNodes {
		errs = errors.Join(errs,
			fmt.Errorf("you cannot have more bad requester actors (%d) than there are nodes (%d)",
				o.NumberOfBadRequesterActors, totalRequesterNodes))
	}

	return errs
}

func WithAutoNodeApproval() ConfigOption {
	return func(cfg *DevStackConfig) {
		cfg.RequesterConfig.DefaultApprovalState = models.NodeMembership.APPROVED
	}
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

func WithExecutorPlugins(enabled bool) ConfigOption {
	return func(cfg *DevStackConfig) {
		cfg.ExecutorPlugins = enabled
	}
}
func WithAuthSecret(secret string) ConfigOption {
	return func(c *DevStackConfig) {
		c.AuthSecret = secret
	}
}

func WithSelfSignedCertificate(cert string, key string) ConfigOption {
	return func(cfg *DevStackConfig) {
		cfg.TLS = DevstackTLSSettings{
			Certificate: cert,
			Key:         key,
		}
	}
}
