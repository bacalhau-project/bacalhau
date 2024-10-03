package types

import (
	"encoding/json"

	"github.com/imdario/mergo"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
)

// NB: Developers, after making changes (comments included) to this struct or any of its children, run go generate.

//go:generate go run gen/generate.go ./
//go:generate go fmt ./generated_constants.go ./generated_descriptions.go
type Bacalhau struct {
	API API `yaml:"API,omitempty" json:"API,omitempty"`
	// NameProvider specifies the method used to generate names for the node. One of: hostname, aws, gcp, uuid, puuid.
	NameProvider string `yaml:"NameProvider,omitempty" json:"NameProvider,omitempty"`
	// DataDir specifies a location on disk where the bacalhau node will maintain state.
	DataDir string `yaml:"DataDir,omitempty" json:"DataDir,omitempty"`
	// StrictVersionMatch indicates whether to enforce strict version matching.
	StrictVersionMatch bool         `yaml:"StrictVersionMatch,omitempty" json:"StrictVersionMatch,omitempty"`
	Orchestrator       Orchestrator `yaml:"Orchestrator,omitempty" json:"Orchestrator,omitempty"`
	Compute            Compute      `yaml:"Compute,omitempty" json:"Compute,omitempty"`
	// Labels are key-value pairs used to describe and categorize the nodes.
	Labels              map[string]string   `yaml:"Labels,omitempty" json:"Labels,omitempty"`
	WebUI               WebUI               `yaml:"WebUI,omitempty" json:"WebUI,omitempty"`
	InputSources        InputSourcesConfig  `yaml:"InputSources,omitempty" json:"InputSources,omitempty"`
	Publishers          PublishersConfig    `yaml:"Publishers,omitempty" json:"Publishers,omitempty"`
	Engines             EngineConfig        `yaml:"Engines,omitempty" json:"Engines,omitempty"`
	ResultDownloaders   ResultDownloaders   `yaml:"ResultDownloaders,omitempty" json:"ResultDownloaders,omitempty"`
	JobDefaults         JobDefaults         `yaml:"JobDefaults,omitempty" json:"JobDefaults,omitempty"`
	JobAdmissionControl JobAdmissionControl `yaml:"JobAdmissionControl,omitempty" json:"JobAdmissionControl,omitempty"`
	Logging             Logging             `yaml:"Logging,omitempty" json:"Logging,omitempty"`
	UpdateConfig        UpdateConfig        `yaml:"UpdateConfig,omitempty" json:"UpdateConfig,omitempty"`
	FeatureFlags        FeatureFlags        `yaml:"FeatureFlags,omitempty" json:"FeatureFlags,omitempty"`
	DisableAnalytics    bool                `yaml:"DisableAnalytics,omitempty" json:"DisableAnalytics,omitempty"`
}

// Copy returns a deep copy of the Bacalhau configuration.
func (b Bacalhau) Copy() (Bacalhau, error) {
	// Serialize the struct to JSON
	data, err := json.Marshal(b)
	if err != nil {
		return Bacalhau{}, bacerrors.Wrap(err, "error marshaling types.Bacalhau while copy")
	}

	// Deserialize the JSON into a new struct
	var cpy Bacalhau
	err = json.Unmarshal(data, &cpy)
	if err != nil {
		return Bacalhau{}, bacerrors.Wrap(err, "error unmarshaling types.Bacalhau while copy")
	}

	return cpy, nil
}

// MergeNew combines the current Bacalhau configuration with another one,
// returning a new instance with the merged configuration.
func (b Bacalhau) MergeNew(other Bacalhau) (Bacalhau, error) {
	// Create a copy of the current config
	merged, err := b.Copy()
	if err != nil {
		return Bacalhau{}, err
	}

	// MergeNew the other config into the copy
	if err = mergo.Merge(&merged, other, mergo.WithOverride); err != nil {
		return Bacalhau{}, err
	}

	return merged, nil
}
