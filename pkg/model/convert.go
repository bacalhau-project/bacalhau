package model

import "github.com/bacalhau-project/bacalhau/pkg/model/v1beta2"

func ConvertV1beta2Spec(s v1beta2.Spec) Spec {
	return Spec{ //nolint:govet
		Engine(s.Engine),
		Verifier(s.Verifier),
		Publisher(s.Publisher),
		PublisherSpec{ //nolint:govet
			Publisher(s.PublisherSpec.Type),
			s.PublisherSpec.Params,
		},
		JobSpecDocker{ //nolint:govet
			s.Docker.Image,
			s.Docker.Entrypoint,
			s.Docker.Parameters,
			s.Docker.EnvironmentVariables,
			s.Docker.WorkingDirectory,
		},
		JobSpecWasm{ //nolint:govet
			ConvertV1beta2StorageSpec(s.Wasm.EntryModule),
			s.Wasm.EntryPoint,
			s.Wasm.Parameters,
			s.Wasm.EnvironmentVariables,
			ConvertV1beta2StorageSpecList(s.Wasm.ImportModules...),
		},
		ResourceUsageConfig(s.Resources),
		NetworkConfig{ //nolint:govet
			Network(s.Network.Type),
			s.Network.Domains,
		},
		s.Timeout,
		ConvertV1beta2StorageSpecList(s.Inputs...),
		ConvertV1beta2StorageSpecList(s.Outputs...),
		s.Annotations,
		ConvertV1beta2LabelSelectorRequirementList(s.NodeSelectors...),
		s.DoNotTrack,
		Deal{ //nolint:govet
			TargetingMode(s.Deal.TargetingMode),
			s.Deal.Concurrency,
			s.Deal.Confidence,
			s.Deal.MinBids,
		},
	}
}
func ConvertV1beta2StorageSpec(s v1beta2.StorageSpec) StorageSpec {
	//nolint: govet
	return StorageSpec{ //nolint:govet
		StorageSourceType(s.StorageSource),
		s.Name,
		s.CID,
		s.URL,
		(*S3StorageSpec)(s.S3),
		s.Repo,
		s.SourcePath,
		s.ReadWrite,
		s.Path,
		s.Metadata,
	}
}

func ConvertV1beta2StorageSpecList(ss ...v1beta2.StorageSpec) []StorageSpec {
	out := make([]StorageSpec, len(ss))
	for i, s := range ss {
		out[i] = ConvertV1beta2StorageSpec(s)
	}
	return out
}

func ConvertV1beta2LabelSelectorRequirementList(ll ...v1beta2.LabelSelectorRequirement) []LabelSelectorRequirement {
	out := make([]LabelSelectorRequirement, len(ll))
	for i, l := range ll {
		out[i] = LabelSelectorRequirement{ //nolint:govet
			l.Key,
			l.Operator,
			l.Values,
		}
	}
	return out
}
