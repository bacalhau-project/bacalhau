package devstack

import (
	"errors"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/node"
)

type ConfigOption = func(cfg *DevStackConfig)

func defaultDevStackConfig() (*DevStackConfig, error) {
	bacalhauConfig, err := config.NewTestConfig()
	if err != nil {
		return nil, err
	}

	return &DevStackConfig{
		BacalhauConfig:         bacalhauConfig,
		NodeDependencyInjector: node.NodeDependencyInjector{},
		NodeOverrides:          nil,

		NumberOfRequesterOnlyNodes: 1,
		NumberOfComputeOnlyNodes:   3,
		NumberOfBadComputeActors:   0,
		CPUProfilingFile:           "",
		MemoryProfilingFile:        "",

		NumberOfHybridNodes: 0,
	}, nil
}

type DevstackTLSSettings struct {
	Certificate string
	Key         string
}

type DevStackConfig struct {
	BacalhauConfig         types.Bacalhau
	NodeDependencyInjector node.NodeDependencyInjector
	NodeOverrides          []node.NodeConfig

	// DevStackOptions
	NumberOfHybridNodes        int // Number of nodes to start in the cluster
	NumberOfRequesterOnlyNodes int // Number of nodes to start in the cluster
	NumberOfComputeOnlyNodes   int // Number of nodes to start in the cluster
	NumberOfBadComputeActors   int // Number of compute nodes to be bad actors
	CPUProfilingFile           string
	MemoryProfilingFile        string
}

func (o *DevStackConfig) MarshalZerologObject(e *zerolog.Event) {
	e.Int("HybridNodes", o.NumberOfHybridNodes).
		Int("RequesterOnlyNodes", o.NumberOfRequesterOnlyNodes).
		Int("ComputeOnlyNodes", o.NumberOfComputeOnlyNodes).
		Int("BadComputeActors", o.NumberOfBadComputeActors).
		Str("CPUProfilingFile", o.CPUProfilingFile).
		Str("MemoryProfilingFile", o.MemoryProfilingFile)
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

	return errs
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
		cfg.BacalhauConfig.Engines.Disabled = disable.Engines
		cfg.BacalhauConfig.Publishers.Disabled = disable.Publishers
		cfg.BacalhauConfig.InputSources.Disabled = disable.Storages
	}
}

func WithAllowListedLocalPaths(paths []string) ConfigOption {
	return func(cfg *DevStackConfig) {
		cfg.BacalhauConfig.Compute.AllowListedLocalPaths = paths
	}
}

func WithAuthSecret(secret string) ConfigOption {
	return func(c *DevStackConfig) {
		c.BacalhauConfig.Orchestrator.Auth.Token = secret
		c.BacalhauConfig.Compute.Auth.Token = secret
	}
}

func WithSelfSignedCertificate(cert string, key string) ConfigOption {
	return func(cfg *DevStackConfig) {
		cfg.BacalhauConfig.API.TLS.CertFile = cert
		cfg.BacalhauConfig.API.TLS.KeyFile = key
	}
}
