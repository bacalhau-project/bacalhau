package model

import (
	"fmt"
	"reflect"

	"github.com/bacalhau-project/bacalhau/pkg/model/specs/engine/wasm"
)

type WasmJobCreatePayload struct {
	ClientID string
	WasmJob  *WasmJob
}

func (w WasmJobCreatePayload) GetClientID() string {
	return w.ClientID
}

type WasmJob struct {
	APIVersion APIVersion `json:"APIVersion,omitempty"`

	WasmSpec      wasm.WasmEngineSpec `json:"WasmSpec"`
	PublisherSpec PublisherSpec       `json:"PublisherSpec"`
	VerifierSpec  Verifier            `json:"VerifierSpec,omitempty"`

	ResourceConfig ResourceUsageConfig `json:"ResourceConfig"`
	NetworkConfig  NetworkConfig       `json:"NetworkConfig"`

	Inputs  []StorageSpec `json:"Inputs,omitempty"`
	Outputs []StorageSpec `json:"Outputs,omitempty"`

	DealSpec Deal `json:"DealSpec"`

	NodeSelectors []LabelSelectorRequirement `json:"NodeSelectors,omitempty"`

	Timeout     float64  `json:"Timeout,omitempty"`
	Annotations []string `json:"Annotations,omitempty"`
}

func (w *WasmJob) ToSpec() (*Spec, error) {
	engine, err := w.WasmSpec.AsSpec()
	if err != nil {
		return nil, err
	}
	return &Spec{
		Engine:        engine,
		Verifier:      w.VerifierSpec,
		Publisher:     w.PublisherSpec.Type,
		PublisherSpec: w.PublisherSpec,
		Resources:     w.ResourceConfig,
		Network:       w.NetworkConfig,
		Timeout:       w.Timeout,
		Inputs:        w.Inputs,
		Outputs:       w.Outputs,
		Annotations:   w.Annotations,
		NodeSelectors: w.NodeSelectors,
		// TODO does this even belong in the spec? Looks unused aside from testing.
		DoNotTrack: false,
		Deal:       w.DealSpec,
	}, nil
}

func (w *WasmJob) Validate() error {
	if reflect.DeepEqual(wasm.WasmEngineSpec{}, w.WasmSpec) {
		return fmt.Errorf("wasm engine spec is empty")
	}

	if reflect.DeepEqual(Deal{}, w.DealSpec) {
		return fmt.Errorf("job deal is empty")
	}

	if w.DealSpec.Concurrency <= 0 {
		return fmt.Errorf("concurrency must be >= 1")
	}

	if w.DealSpec.Confidence < 0 {
		return fmt.Errorf("confidence must be >= 0")
	}

	if !IsValidVerifier(w.VerifierSpec) {
		return fmt.Errorf("invalid verifier type: %s", w.VerifierSpec.String())
	}

	if !IsValidPublisher(w.PublisherSpec.Type) {
		return fmt.Errorf("invalid publisher type: %s", w.PublisherSpec.Type.String())
	}

	if err := w.NetworkConfig.IsValid(); err != nil {
		return err
	}

	if w.DealSpec.Confidence > w.DealSpec.Concurrency {
		return fmt.Errorf("the deal confidence cannot be higher than the concurrency")
	}

	for _, inputVolume := range w.Inputs {
		if !IsValidStorageSourceType(inputVolume.StorageSource) {
			return fmt.Errorf("invalid input volume type: %s", inputVolume.StorageSource.String())
		}
	}
	return nil
}
