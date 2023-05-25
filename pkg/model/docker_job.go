package model

import (
	"fmt"
	"reflect"

	"github.com/bacalhau-project/bacalhau/pkg/model/spec/engine/docker"
)

type DockerJobCreatePayload struct {
	ClientID  string
	DockerJob *DockerJob
}

func (d DockerJobCreatePayload) GetClientID() string {
	return d.ClientID
}

type DockerJob struct {
	APIVersion APIVersion `json:"APIVersion,omitempty"`

	DockerSpec    docker.DockerEngineSpec `json:"DockerSpec"`
	PublisherSpec PublisherSpec           `json:"PublisherSpec"`
	VerifierSpec  Verifier                `json:"VerifierSpec,omitempty"`

	ResourceConfig ResourceUsageConfig `json:"ResourceConfig"`
	NetworkConfig  NetworkConfig       `json:"NetworkConfig"`

	Inputs  []StorageSpec `json:"Inputs,omitempty"`
	Outputs []StorageSpec `json:"Outputs,omitempty"`

	DealSpec Deal `json:"DealSpec"`

	NodeSelectors []LabelSelectorRequirement `json:"NodeSelectors,omitempty"`

	Timeout     float64  `json:"Timeout,omitempty"`
	Annotations []string `json:"Annotations,omitempty"`
}

func (d *DockerJob) ToSpec() (*Spec, error) {
	engine, err := d.DockerSpec.AsSpec()
	if err != nil {
		return nil, err
	}
	return &Spec{
		Engine:        engine,
		Verifier:      d.VerifierSpec,
		Publisher:     d.PublisherSpec.Type,
		PublisherSpec: d.PublisherSpec,
		Resources:     d.ResourceConfig,
		Network:       d.NetworkConfig,
		Timeout:       d.Timeout,
		Inputs:        d.Inputs,
		Outputs:       d.Outputs,
		Annotations:   d.Annotations,
		NodeSelectors: d.NodeSelectors,
		// TODO does this even belong in the spec? Looks unused aside from testing.
		DoNotTrack: false,
		Deal:       d.DealSpec,
	}, nil
}

func (d *DockerJob) Validate() error {
	if reflect.DeepEqual(docker.DockerEngineSpec{}, d.DockerSpec) {
		return fmt.Errorf("docker engine spec is empty")
	}

	if reflect.DeepEqual(Deal{}, d.DealSpec) {
		return fmt.Errorf("job deal is empty")
	}

	if d.DealSpec.Concurrency <= 0 {
		return fmt.Errorf("concurrency must be >= 1")
	}

	if d.DealSpec.Confidence < 0 {
		return fmt.Errorf("confidence must be >= 0")
	}

	if !IsValidVerifier(d.VerifierSpec) {
		return fmt.Errorf("invalid verifier type: %s", d.VerifierSpec.String())
	}

	if !IsValidPublisher(d.PublisherSpec.Type) {
		return fmt.Errorf("invalid publisher type: %s", d.PublisherSpec.Type.String())
	}

	if err := d.NetworkConfig.IsValid(); err != nil {
		return err
	}

	if d.DealSpec.Confidence > d.DealSpec.Concurrency {
		return fmt.Errorf("the deal confidence cannot be higher than the concurrency")
	}

	for _, inputVolume := range d.Inputs {
		if !IsValidStorageSourceType(inputVolume.StorageSource) {
			return fmt.Errorf("invalid input volume type: %s", inputVolume.StorageSource.String())
		}
	}
	return nil
}
