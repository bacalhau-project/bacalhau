package types

import (
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/lib/validate"
)

type Orchestrator struct {
	Enabled          bool             `yaml:"Enabled,omitempty"`
	Listen           string           `yaml:"Listen,omitempty"`
	Advertise        string           `yaml:"Advertise,omitempty"`
	TLS              TLS              `yaml:"TLS,omitempty"`
	Cluster          Cluster          `yaml:"Cluster,omitempty"`
	NodeManager      NodeManager      `yaml:"NodeManager,omitempty"`
	Scheduler        Scheduler        `yaml:"Scheduler,omitempty"`
	EvaluationBroker EvaluationBroker `yaml:"EvaluationBroker,omitempty"`
}

func (c Orchestrator) Validate() error {
	if err := validateAddress(c.Listen); err != nil {
		return fmt.Errorf("orchestrator listen address is invalid: %w", err)
	}
	if err := validateAddress(c.Advertise); err != nil {
		return fmt.Errorf("orchestrator advertise address is invalid: %w", err)
	}

	if err := validateFields(c); err != nil {
		return err
	}
	return nil
}

type Cluster struct {
	Listen    string   `yaml:"Listen,omitempty"`
	Advertise string   `yaml:"Advertise,omitempty"`
	TLS       TLS      `yaml:"TLS,omitempty"`
	Peers     []string `yaml:"Peers,omitempty"`
}

func (c Cluster) Validate() error {
	// TODO what are valid schemas for the Listen address of a cluster?
	// field isn't required
	if c.Listen != "" {
		if err := validateURL(c.Listen, "http", "https"); err != nil {
			return fmt.Errorf("orchestrator cluster listen address is invalid: %w", err)
		}
	}
	// field isn't required
	if c.Advertise != "" {
		// TODO what are valid schemas for the Advertise address of a cluster?
		if err := validateURL(c.Advertise, "nats", ""); err != nil {
			return fmt.Errorf("orchestraor cluster advertise address is invalid: %w", err)
		}
	}

	for i, p := range c.Peers {
		// TODO what are valid peer schemas?
		if err := validateURL(p, "nats", ""); err != nil {
			return fmt.Errorf("peer address %q at index %d is invalid: %w", p, i, err)
		}
	}
	if err := validateFields(c); err != nil {
		return err
	}
	return nil
}

type NodeManager struct {
	DisconnectTimeout Duration `yaml:"DisconnectTimeout,omitempty"`
	ManualApproval    bool     `yaml:"ManualApproval,omitempty"`
}

func (c NodeManager) Validate() error {
	if c.DisconnectTimeout == 0 {
		return fmt.Errorf("node manager disconnect timeout cannot be zero")
	}
	return nil
}

type Scheduler struct {
	WorkerCount          int      `yaml:"WorkerCount,omitempty"`
	HousekeepingInterval Duration `yaml:"HousekeepingInterval,omitempty"`
	HousekeepingTimeout  Duration `yaml:"HousekeepingTimeout,omitempty"`
}

func (c Scheduler) Validate() error {
	if err := validate.IsGreaterThanZero(c.WorkerCount, "scheduler worker count cannot be zero"); err != nil {
		return err
	}
	if err := validate.IsGreaterThanZero(
		c.HousekeepingInterval,
		"scheduler house keeping interval cannot be zero",
	); err != nil {
		return err
	}
	// TODO is it acceptable for the housekeepingtimeout to be zero? assuming zero means no timeout.

	return nil
}

type EvaluationBroker struct {
	VisibilityTimeout Duration `yaml:"VisibilityTimeout,omitempty"`
	MaxRetryCount     int      `yaml:"MaxRetryCount,omitempty"`
}
