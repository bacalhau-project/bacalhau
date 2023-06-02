package model

import (
	"fmt"

	"github.com/ipfs/go-cid"

	bacalhau_model "github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec"
	spec_docker "github.com/bacalhau-project/bacalhau/pkg/model/spec/engine/docker"
	spec_noop "github.com/bacalhau-project/bacalhau/pkg/model/spec/engine/noop"
	spec_wasm "github.com/bacalhau-project/bacalhau/pkg/model/spec/engine/wasm"
	spec_estuary "github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/estuary"
	spec_filecoin "github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/filecoin"
	spec_git "github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/git"
	spec_gitlfs "github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/gitlfs"
	spec_inline "github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/inline"
	spec_ipfs "github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/ipfs"
	spec_local "github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/local"
	spec_s3 "github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/s3"
	spec_url "github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/url"
	bacalhau_model_beta "github.com/bacalhau-project/bacalhau/pkg/model/v1beta1"
)

func parseEventEngineType(engine spec.Engine) (bacalhau_model_beta.Engine, error) {
	switch engine.Schema {
	case cid.Undef:
		return bacalhau_model_beta.Engine(0), nil
	case spec_docker.EngineType:
		return bacalhau_model_beta.EngineDocker, nil
	case spec_wasm.EngineType:
		return bacalhau_model_beta.EngineWasm, nil
	case spec_noop.EngineType:
		return bacalhau_model_beta.EngineNoop, nil
	default:
		return 0, fmt.Errorf("impossible engine type: %s", engine)
	}
}

func convertSliceToMap(slice []string) map[string]string {
	// Initialize a new map of string to string
	m := make(map[string]string)

	// Iterate over the slice in pairs
	for i := 0; i < len(slice); i += 2 {
		// Check if there is a corresponding value for the key
		if i+1 < len(slice) {
			// Use the even index as the key and the odd index as the value
			m[slice[i]] = slice[i+1]
		} else {
			// If there's no corresponding value, assign an empty string
			m[slice[i]] = ""
		}
	}

	// Return the new map
	return m
}

func parseEventStorageSpecList(storages []spec.Storage) ([]bacalhau_model_beta.StorageSpec, error) {
	out := make([]bacalhau_model_beta.StorageSpec, 0, len(storages))
	for _, storage := range storages {
		s, err := parseEventStorageSpec(storage)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, nil
}

func parseEventStorageSpec(storage spec.Storage) (bacalhau_model_beta.StorageSpec, error) {
	switch storage.Schema {
	// TODO validte this is correct
	// for the case when the storage spec is undefined
	case cid.Undef:
		return bacalhau_model_beta.StorageSpec{}, nil
	case spec_estuary.StorageType:
		es, err := spec_estuary.Decode(storage)
		if err != nil {
			return bacalhau_model_beta.StorageSpec{}, err
		}
		return bacalhau_model_beta.StorageSpec{
			StorageSource: bacalhau_model_beta.StorageSourceEstuary,
			Name:          storage.Name,
			CID:           es.CID.String(),
			URL:           es.URL,
			SourcePath:    "",
			Path:          storage.Mount,
			Metadata:      storage.Metadata.ToMap(),
		}, nil
	case spec_filecoin.StorageType:
		fs, err := spec_filecoin.Decode(storage)
		if err != nil {
			return bacalhau_model_beta.StorageSpec{}, err
		}
		return bacalhau_model_beta.StorageSpec{
			StorageSource: bacalhau_model_beta.StorageSourceFilecoin,
			Name:          storage.Name,
			CID:           fs.CID.String(),
			URL:           "",
			SourcePath:    "",
			Path:          storage.Mount,
			Metadata:      storage.Metadata.ToMap(),
		}, nil
	case spec_inline.StorageType:
		is, err := spec_inline.Decode(storage)
		if err != nil {
			return bacalhau_model_beta.StorageSpec{}, err
		}
		return bacalhau_model_beta.StorageSpec{
			StorageSource: bacalhau_model_beta.StorageSourceInline,
			Name:          storage.Name,
			CID:           "",
			URL:           is.URL,
			SourcePath:    "",
			Path:          storage.Mount,
			Metadata:      storage.Metadata.ToMap(),
		}, nil
	case spec_ipfs.StorageType:
		is, err := spec_ipfs.Decode(storage)
		if err != nil {
			return bacalhau_model_beta.StorageSpec{}, err
		}
		return bacalhau_model_beta.StorageSpec{
			StorageSource: bacalhau_model_beta.StorageSourceIPFS,
			Name:          storage.Name,
			CID:           is.CID.String(),
			URL:           "",
			SourcePath:    "",
			Path:          storage.Mount,
			Metadata:      storage.Metadata.ToMap(),
		}, nil
	case spec_local.StorageType:
		ls, err := spec_local.Decode(storage)
		if err != nil {
			return bacalhau_model_beta.StorageSpec{}, err
		}
		return bacalhau_model_beta.StorageSpec{
			StorageSource: bacalhau_model_beta.StorageSourceLocalDirectory,
			Name:          storage.Name,
			CID:           "",
			URL:           "",
			SourcePath:    ls.Source,
			Path:          storage.Mount,
			Metadata:      storage.Metadata.ToMap(),
		}, nil
	case spec_url.StorageType:
		us, err := spec_url.Decode(storage)
		if err != nil {
			return bacalhau_model_beta.StorageSpec{}, err
		}
		return bacalhau_model_beta.StorageSpec{
			StorageSource: bacalhau_model_beta.StorageSourceLocalDirectory,
			Name:          storage.Name,
			CID:           "",
			URL:           us.URL,
			SourcePath:    "",
			Path:          storage.Mount,
			Metadata:      storage.Metadata.ToMap(),
		}, nil
	case spec_s3.StorageType:
		return bacalhau_model_beta.StorageSpec{}, fmt.Errorf("unsupported storage spec: %s", storage)
	case spec_git.StorageType:
		return bacalhau_model_beta.StorageSpec{}, fmt.Errorf("unsupported storage spec: %s", storage)
	case spec_gitlfs.StorageType:
		return bacalhau_model_beta.StorageSpec{}, fmt.Errorf("unsupported storage spec: %s", storage)
	default:
		return bacalhau_model_beta.StorageSpec{}, fmt.Errorf("impossible storage type: %s", storage)
	}
}

func parseEventNodeSelectors(requirement []bacalhau_model.LabelSelectorRequirement) []bacalhau_model_beta.LabelSelectorRequirement {
	out := make([]bacalhau_model_beta.LabelSelectorRequirement, 0, len(requirement))

	for _, r := range requirement {
		out = append(out, bacalhau_model_beta.LabelSelectorRequirement{
			Key:      r.Key,
			Operator: r.Operator,
			Values:   r.Values,
		})
	}
	return out
}

func ConvertEventToLegacy(event bacalhau_model.JobEvent) (bacalhau_model_beta.JobEvent, error) {
	engineType, err := parseEventEngineType(event.Spec.Engine)
	if err != nil {
		return bacalhau_model_beta.JobEvent{}, err
	}
	inputs, err := parseEventStorageSpecList(event.Spec.Inputs)
	if err != nil {
		return bacalhau_model_beta.JobEvent{}, err
	}
	outputs, err := parseEventStorageSpecList(event.Spec.Outputs)
	if err != nil {
		return bacalhau_model_beta.JobEvent{}, err
	}
	publishRes, err := parseEventStorageSpec(event.PublishedResult)
	if err != nil {
		return bacalhau_model_beta.JobEvent{}, err
	}

	legacyEvent := bacalhau_model_beta.JobEvent{
		APIVersion:   event.APIVersion,
		JobID:        event.JobID,
		ShardIndex:   0,
		ExecutionID:  event.ExecutionID,
		ClientID:     event.ClientID,
		SourceNodeID: event.SourceNodeID,
		TargetNodeID: event.TargetNodeID,
		EventName:    bacalhau_model_beta.JobEventType(event.EventName),
		Spec: bacalhau_model_beta.Spec{
			Engine:    engineType,
			Verifier:  bacalhau_model_beta.Verifier(event.Spec.Verifier),
			Publisher: bacalhau_model_beta.Publisher(event.Spec.Publisher),
			Docker:    bacalhau_model_beta.JobSpecDocker{},
			Wasm:      bacalhau_model_beta.JobSpecWasm{},
			Resources: bacalhau_model_beta.ResourceUsageConfig(event.Spec.Resources),
			Network: bacalhau_model_beta.NetworkConfig{
				Type:    bacalhau_model_beta.Network(event.Spec.Network.Type),
				Domains: event.Spec.Network.Domains,
			},
			Timeout:       event.Spec.Timeout,
			Inputs:        inputs,
			Outputs:       outputs,
			Annotations:   event.Spec.Annotations,
			NodeSelectors: parseEventNodeSelectors(event.Spec.NodeSelectors),
			DoNotTrack:    event.Spec.DoNotTrack,
			Deal:          bacalhau_model_beta.Deal(event.Spec.Deal),
		},
		Deal:                 bacalhau_model_beta.Deal(event.Deal),
		Status:               event.Status,
		VerificationProposal: event.VerificationProposal,
		VerificationResult:   bacalhau_model_beta.VerificationResult(event.VerificationResult),
		PublishedResult:      publishRes,
		EventTime:            event.EventTime,
		SenderPublicKey:      bacalhau_model_beta.PublicKey(event.SenderPublicKey),
		RunOutput:            (*bacalhau_model_beta.RunCommandResult)(event.RunOutput),
	}
	if event.Spec.Engine.Schema == spec_docker.EngineType {
		de, err := spec_docker.Decode(event.Spec.Engine)
		if err != nil {
			return bacalhau_model_beta.JobEvent{}, err
		}
		legacyEvent.Spec.Docker = bacalhau_model_beta.JobSpecDocker{
			Image:                de.Image,
			Entrypoint:           de.Entrypoint,
			EnvironmentVariables: de.EnvironmentVariables,
			WorkingDirectory:     de.WorkingDirectory,
		}
	}
	if event.Spec.Engine.Schema == spec_wasm.EngineType {
		we, err := spec_wasm.Decode(event.Spec.Engine)
		if err != nil {
			return bacalhau_model_beta.JobEvent{}, err
		}
		importModules, err := parseEventStorageSpecList(we.ImportModules)
		if err != nil {
			return bacalhau_model_beta.JobEvent{}, err
		}
		entryModule, err := parseEventStorageSpec(*we.EntryModule)
		if err != nil {
			return bacalhau_model_beta.JobEvent{}, err
		}
		legacyEvent.Spec.Wasm = bacalhau_model_beta.JobSpecWasm{
			EntryModule:          entryModule,
			EntryPoint:           we.EntryPoint,
			Parameters:           we.Parameters,
			EnvironmentVariables: convertSliceToMap(we.EnvironmentVariables),
			ImportModules:        importModules,
		}

	}
	return legacyEvent, nil
}
