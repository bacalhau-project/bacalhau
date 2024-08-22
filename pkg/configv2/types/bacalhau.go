package types

import (
	"fmt"
	"slices"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/util/idgen"
)

type Bacalhau struct {
	API                 API                 `yaml:"API,omitempty"`
	NameProvider        string              `yaml:"NameProvider,omitempty"`
	DataDir             string              `yaml:"DataDir,omitempty"`
	StrictVersionMatch  bool                `yaml:"StrictVersionMatch,omitempty"`
	Orchestrator        Orchestrator        `yaml:"Orchestrator,omitempty"`
	Compute             Compute             `yaml:"Compute,omitempty"`
	WebUI               WebUI               `yaml:"WebUI,omitempty"`
	InputSources        InputSourcesConfig  `yaml:"InputSources,omitempty"`
	Publishers          PublishersConfig    `yaml:"Publishers,omitempty"`
	Executors           ExecutorsConfig     `yaml:"Executors,omitempty"`
	ResultDownloaders   ResultDownloaders   `yaml:"ResultDownloaders,omitempty"`
	JobDefaults         JobDefaults         `yaml:"JobDefaults,omitempty"`
	JobAdmissionControl JobAdmissionControl `yaml:"JobAdmissionControl,omitempty"`
	Logging             Logging             `yaml:"Logging,omitempty"`
	UpdateConfig        UpdateConfig        `yaml:"UpdateConfig,omitempty"`
	FeatureFlags        FeatureFlags        `yaml:"FeatureFlags,omitempty"`
}

// Validate returns an error if the config is invalid
// It uses the helper method validateFields to validate fields that implement Validatable
func (c Bacalhau) Validate() error {
	// check non-struct fields
	if c.NameProvider == "" {
		return fmt.Errorf("NameProvider cannot be empty")
	}
	if !slices.ContainsFunc(idgen.NameProviders, func(s string) bool {
		return strings.ToLower(s) == strings.ToLower(c.NameProvider)
	}) {
		return fmt.Errorf("NameProvider type %q unknow. must be one of: %v", c.NameProvider, idgen.NameProviders)
	}

	if c.DataDir == "" {
		return fmt.Errorf("DataDir cannot be empty")
	}

	if !c.Orchestrator.Enabled && !c.Compute.Enabled {
		return fmt.Errorf("compute and orchestraor services cannot both be disabled")
	}

	// Validate struct fields using the helper method
	if err := validateFields(c); err != nil {
		return err
	}

	return nil
}

type API struct {
	Address string     `yaml:"Address,omitempty"`
	Host    string     `yaml:"-"`
	Port    int        `yaml:"-"`
	TLS     TLS        `yaml:"TLS,omitempty"`
	Auth    AuthConfig `yaml:"Auth,omitempty"`
}

func (c API) Validate() error {
	if err := validateURL(c.Address, "http", "https"); err != nil {
		return fmt.Errorf("API address invalid: %w", err)
	}
	return nil
}

type TLS struct {
	CertFile string `yaml:"Certificate,omitempty"`
	KeyFile  string `yaml:"Key,omitempty"`
	CAFile   string `yaml:"CAFile,omitempty"`
}

func (c TLS) Validate() error {
	// TODO consider validating a key is present when the CAFile is, and visa versa.
	if err := validateFileIffExists(c.CertFile); err != nil {
		return fmt.Errorf("TLS CertFile invalid: %w", err)
	}
	if err := validateFileIffExists(c.KeyFile); err != nil {
		return fmt.Errorf("TLS KeyFile invalid: %w", err)
	}
	if err := validateFileIffExists(c.CAFile); err != nil {
		return fmt.Errorf("TLS CAFile invalid: %w", err)
	}
	return nil
}

type WebUI struct {
	Enabled bool   `yaml:"Enabled,omitempty"`
	Listen  string `yaml:"Listen,omitempty"`
}

func (c WebUI) Validate() error {
	if c.Enabled {
		if err := validateURL(c.Listen); err != nil {
			return fmt.Errorf("WebUI address invalid: %w", err)
		}
	}
	return nil
}

type Logging struct {
	Level                string   `yaml:"Level,omitempty"`
	Mode                 string   `yaml:"Mode,omitempty"`
	LogDebugInfoInterval Duration `yaml:"LogDebugInfoInterval,omitempty"`
}

func (c Logging) Validate() error {
	validLogLevels := []string{"trace", "debug", "info", "warn", "error", "fatal"}
	if !slices.ContainsFunc(validLogLevels, func(s string) bool {
		return strings.ToLower(c.Level) == s
	}) {
		return fmt.Errorf("logging level %q invalid. must be one of: %v", c.Level, validLogLevels)
	}
	return nil

}

type FeatureFlags struct {
	ExecTranslation bool `yaml:"ExecTranslation,omitempty"`
}

type UpdateConfig struct {
	Interval Duration `yaml:"Interval,omitempty"`
}

type JobAdmissionControl struct {
	RejectStatelessJobs bool   `yaml:"RejectStatelessJobs,omitempty"`
	AcceptNetworkedJobs bool   `yaml:"AcceptNetworkedJobs,omitempty"`
	ProbeHTTP           string `yaml:"ProbeHTTP,omitempty"`
	ProbeExec           string `yaml:"ProbeExec,omitempty"`
}

type JobDefaults struct {
	Batch   JobDefaultsConfig `yaml:"Batch,omitempty"`
	Daemon  JobDefaultsConfig `yaml:"Daemon,omitempty"`
	Service JobDefaultsConfig `yaml:"Service,omitempty"`
	Ops     JobDefaultsConfig `yaml:"Ops,omitempty"`
}

type JobDefaultsConfig struct {
	Priority uint              `yaml:"Priority,omitempty"`
	Task     TaskDefaultConfig `yaml:"Task,omitempty"`
}

type TaskDefaultConfig struct {
	Resources Resource               `yaml:"Resources,omitempty"`
	Publisher DefaultPublisherConfig `yaml:"Publisher,omitempty"`
	Timeouts  TaskTimeoutConfig      `yaml:"Timeouts,omitempty"`
}

type DefaultPublisherConfig struct {
	Type string `yaml:"Type,omitempty"`
}

func (c DefaultPublisherConfig) Validate() error {
	if c.Type == "" {
		return fmt.Errorf("default publisher type cannot be empty")
	}
	isValidType := false
	for _, expected := range models.PublisherNames {
		if strings.ToLower(c.Type) == strings.ToLower(expected) {
			isValidType = true
		}
	}
	if !isValidType {
		return fmt.Errorf("default publisher type %q unknow. must be one of: %v", c.Type, models.PublisherNames)
	}
	return nil
}

type TaskTimeoutConfig struct {
	ExecutionTimeout Duration `yaml:"ExecutionTimeout,omitempty"`
}
