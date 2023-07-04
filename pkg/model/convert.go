package model

import (
	"github.com/bacalhau-project/bacalhau/pkg/model/v1beta2"
)

//
// Forwards v1beta2 -> current
//

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

func ConvertV1beta2ExecutionStateList(ee ...v1beta2.ExecutionState) []ExecutionState {
	out := make([]ExecutionState, len(ee))
	for i, e := range ee {
		out[i] = ConvertV1beta2ExecutionState(e)
	}
	return out
}

func ConvertV1beta2ExecutionState(e v1beta2.ExecutionState) ExecutionState {
	return ExecutionState{ //nolint:govet
		e.JobID,
		e.NodeID,
		e.ComputeReference,
		ExecutionStateType(e.State),
		e.AcceptedAskForBid,
		e.Status,
		e.VerificationProposal,
		VerificationResult(e.VerificationResult),
		ConvertV1beta2StorageSpec(e.PublishedResult),
		(*RunCommandResult)(e.RunOutput),
		e.Version,
		e.CreateTime,
		e.UpdateTime,
	}
}

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

func ConvertV1beta2Job(j v1beta2.Job) Job {
	return Job{ //nolint:govet
		APIVersionLatest().String(),
		Metadata{ //nolint:govet
			j.Metadata.ID,
			j.Metadata.CreatedAt,
			j.Metadata.ClientID,
			JobRequester{
				j.Metadata.Requester.RequesterNodeID,
				PublicKey(j.Metadata.Requester.RequesterPublicKey),
			},
		},
		ConvertV1beta2Spec(j.Spec),
	}
}

func ConvertV1beta2IncludeTagList(ii ...v1beta2.IncludedTag) []IncludedTag {
	out := make([]IncludedTag, len(ii))
	for i, t := range ii {
		out[i] = IncludedTag(t)
	}
	return out
}

func ConvertV1beta2ExcludeTagList(ee ...v1beta2.ExcludedTag) []ExcludedTag {
	out := make([]ExcludedTag, len(ee))
	for i, t := range ee {
		out[i] = ExcludedTag(t)
	}
	return out
}

func ConvertV1beta2JobCreatePayload(p v1beta2.JobCreatePayload) JobCreatePayload {
	out := JobCreatePayload{
		ClientID:   p.ClientID,
		APIVersion: p.APIVersion,
	}
	if p.Spec != nil {
		spec := ConvertV1beta2Spec(*p.Spec)
		out.Spec = &spec
	}
	return out
}

func ConvertV1beta2PublishedResult(p v1beta2.PublishedResult) PublishedResult {
	return PublishedResult{ //nolint:govet
		p.NodeID,
		ConvertV1beta2StorageSpec(p.Data),
	}
}

func ConvertV1beta2PublishedResultList(pp ...v1beta2.PublishedResult) []PublishedResult {
	out := make([]PublishedResult, len(pp))
	for i, p := range pp {
		out[i] = ConvertV1beta2PublishedResult(p)
	}
	return out
}

func ConvertV1beta2JobState(js v1beta2.JobState) JobState {
	return JobState{ //nolint:govet
		js.JobID,
		ConvertV1beta2ExecutionStateList(js.Executions...),
		JobStateType(js.State),
		js.Version,
		js.CreateTime,
		js.UpdateTime,
		js.TimeoutAt,
	}
}

func ConvertV1beta2JobWithInfoList(jwis ...*v1beta2.JobWithInfo) []*JobWithInfo {
	out := make([]*JobWithInfo, len(jwis))
	for i, jwi := range jwis {
		out[i] = ConvertV1beta2JobWithInfo(jwi)
	}
	return out
}

func ConvertV1beta2JobWithInfo(jwi *v1beta2.JobWithInfo) *JobWithInfo {
	return &JobWithInfo{ //nolint:govet
		ConvertV1beta2Job(jwi.Job),
		ConvertV1beta2JobState(jwi.State),
		ConvertV1beta2JobHistoryListToJobHistoryList(jwi.History...),
	}
}

func ConvertV1beta2JobHistoryListToJobHistoryList(hh ...v1beta2.JobHistory) []JobHistory {
	out := make([]JobHistory, len(hh))
	for i, h := range hh {
		out[i] = ConvertV1beta2JobHistoryToJobHistory(h)
	}
	return out
}

func ConvertV1beta2JobHistoryToJobHistory(h v1beta2.JobHistory) JobHistory {
	out := JobHistory{ //nolint:govet
		JobHistoryType(h.Type),
		h.JobID,
		h.NodeID,
		h.ComputeReference,
		nil,
		nil,
		h.NewVersion,
		h.Comment,
		h.Time,
	}

	if h.JobState != nil {
		out.JobState = &StateChange[JobStateType]{ //nolint:govet
			JobStateType(h.JobState.Previous),
			JobStateType(h.JobState.New),
		}
	}

	if h.ExecutionState != nil {
		out.ExecutionState = &StateChange[ExecutionStateType]{ //nolint:govet
			ExecutionStateType(h.ExecutionState.Previous),
			ExecutionStateType(h.ExecutionState.New),
		}
	}
	return out
}

func ConvertV1beta2NodeInfo(ni v1beta2.NodeInfo) NodeInfo {
	return NodeInfo{ //nolint:govet
		BuildVersionInfo(ni.BacalhauVersion),
		ni.PeerInfo,
		NodeType(ni.NodeType),
		ni.Labels,
		ConvertV1beta2ComputeNodeInfo(ni.ComputeNodeInfo),
	}
}

func ConvertV1beta2NodeInfoList(nis ...v1beta2.NodeInfo) []NodeInfo {
	out := make([]NodeInfo, len(nis))
	for i, ni := range nis {
		out[i] = ConvertV1beta2NodeInfo(ni)
	}
	return out
}

func ConvertV1beta2ComputeNodeInfo(cni *v1beta2.ComputeNodeInfo) *ComputeNodeInfo {
	return &ComputeNodeInfo{ //nolint:govet
		ConvertV1beta2EngineList(cni.ExecutionEngines...),
		ConvertV1beta2VerifierList(cni.Verifiers...),
		ConvertV1beta2PublisherList(cni.Publishers...),
		ConvertV1beta2StorageSourceTypeList(cni.StorageSources...),
		ResourceUsageData(cni.MaxCapacity),
		ResourceUsageData(cni.AvailableCapacity),
		ResourceUsageData(cni.MaxJobRequirements),
		cni.RunningExecutions,
		cni.EnqueuedExecutions,
	}
}

func ConvertV1beta2StorageSourceType(s v1beta2.StorageSourceType) StorageSourceType {
	return StorageSourceType(s)
}

func ConvertV1beta2StorageSourceTypeList(ss ...v1beta2.StorageSourceType) []StorageSourceType {
	out := make([]StorageSourceType, len(ss))
	for i, s := range ss {
		out[i] = ConvertV1beta2StorageSourceType(s)
	}
	return out
}

func ConvertV1beta2Publisher(p v1beta2.Publisher) Publisher {
	return Publisher(p)
}

func ConvertV1beta2PublisherList(pp ...v1beta2.Publisher) []Publisher {
	out := make([]Publisher, len(pp))
	for i, p := range pp {
		out[i] = ConvertV1beta2Publisher(p)
	}
	return out
}

func ConvertV1beta2Verifier(v v1beta2.Verifier) Verifier {
	return Verifier(v)
}

func ConvertV1beta2VerifierList(vv ...v1beta2.Verifier) []Verifier {
	out := make([]Verifier, len(vv))
	for i, v := range vv {
		out[i] = ConvertV1beta2Verifier(v)
	}
	return out
}

func ConvertV1beta2Engine(e v1beta2.Engine) Engine {
	return Engine(e)
}

func ConvertV1beta2EngineList(ee ...v1beta2.Engine) []Engine {
	out := make([]Engine, len(ee))
	for i, e := range ee {
		out[i] = ConvertV1beta2Engine(e)
	}
	return out
}

//
// Backwards current -> v1beta2
//

func ConvertJobToV1beta2(j Job) v1beta2.Job {
	return v1beta2.Job{ //nolint:govet
		j.APIVersion,
		v1beta2.Metadata{ //nolint:govet
			j.Metadata.ID,
			j.Metadata.CreatedAt,
			j.Metadata.ClientID,
			v1beta2.JobRequester{ //nolint:govet
				j.Metadata.Requester.RequesterNodeID,
				v1beta2.PublicKey(j.Metadata.Requester.RequesterPublicKey),
			},
		},
		v1beta2.Spec{ //nolint:govet
			v1beta2.Engine(j.Spec.Engine),
			v1beta2.Verifier(j.Spec.Verifier),
			v1beta2.Publisher(j.Spec.Publisher),
			v1beta2.PublisherSpec{ //nolint:govet
				v1beta2.Publisher(j.Spec.PublisherSpec.Type),
				j.Spec.PublisherSpec.Params,
			},
			v1beta2.JobSpecDocker{ //nolint:govet
				j.Spec.Docker.Image,
				j.Spec.Docker.Entrypoint,
				j.Spec.Docker.Parameters,
				j.Spec.Docker.EnvironmentVariables,
				j.Spec.Docker.WorkingDirectory,
			},
			v1beta2.JobSpecWasm{ //nolint:govet
				ConvertStorageSpecToV1beta2(j.Spec.Wasm.EntryModule),
				j.Spec.Wasm.EntryPoint,
				j.Spec.Wasm.Parameters,
				j.Spec.Wasm.EnvironmentVariables,
				ConvertStorageSpecListToV1beta2(j.Spec.Wasm.ImportModules...),
			},
			v1beta2.ResourceUsageConfig(j.Spec.Resources),
			v1beta2.NetworkConfig{ //nolint:govet
				v1beta2.Network(j.Spec.Network.Type),
				j.Spec.Network.Domains,
			},
			j.Spec.Timeout,
			ConvertStorageSpecListToV1beta2(j.Spec.Inputs...),
			ConvertStorageSpecListToV1beta2(j.Spec.Outputs...),
			j.Spec.Annotations,
			ConvertLabelSelectorRequirementListToV1beta2(j.Spec.NodeSelectors...),
			j.Spec.DoNotTrack,
			v1beta2.Deal{ //nolint:govet
				v1beta2.TargetingMode(j.Spec.Deal.TargetingMode),
				j.Spec.Deal.Concurrency,
				j.Spec.Deal.Confidence,
				j.Spec.Deal.MinBids,
			},
		},
	}
}

func ConvertJobListToV1beta2List(jj ...Job) []v1beta2.Job {
	out := make([]v1beta2.Job, len(jj))
	for i, j := range jj {
		out[i] = ConvertJobToV1beta2(j)
	}
	return out
}

func ConvertJobStateToV1beta2(s JobState) v1beta2.JobState {
	return v1beta2.JobState{ //nolint:govet
		s.JobID,
		ConvertExecutionStateToV1beta2List(s.Executions...),
		v1beta2.JobStateType(s.State),
		s.Version,
		s.CreateTime,
		s.UpdateTime,
		s.TimeoutAt,
	}
}

func ConvertExecutionStateToV1beta2List(ee ...ExecutionState) []v1beta2.ExecutionState {
	out := make([]v1beta2.ExecutionState, len(ee))
	for i, e := range ee {
		out[i] = ConvertExecutionStateToV1beta2(e)
	}
	return out
}

func ConvertExecutionStateToV1beta2(e ExecutionState) v1beta2.ExecutionState {
	return v1beta2.ExecutionState{ //nolint:govet
		e.JobID,
		e.NodeID,
		e.ComputeReference,
		v1beta2.ExecutionStateType(e.State),
		e.AcceptedAskForBid,
		e.Status,
		e.VerificationProposal,
		v1beta2.VerificationResult(e.VerificationResult),
		ConvertStorageSpecToV1beta2(e.PublishedResult),
		(*v1beta2.RunCommandResult)(e.RunOutput),
		e.Version,
		e.CreateTime,
		e.UpdateTime,
	}
}

func ConvertLabelSelectorRequirementListToV1beta2(ll ...LabelSelectorRequirement) []v1beta2.LabelSelectorRequirement {
	out := make([]v1beta2.LabelSelectorRequirement, len(ll))
	for i, l := range ll {
		out[i] = v1beta2.LabelSelectorRequirement{ //nolint:govet
			l.Key,
			l.Operator,
			l.Values,
		}
	}
	return out
}

func ConvertStorageSpecToV1beta2(s StorageSpec) v1beta2.StorageSpec {
	return v1beta2.StorageSpec{ //nolint:govet
		v1beta2.StorageSourceType(s.StorageSource),
		s.Name,
		s.CID,
		s.URL,
		(*v1beta2.S3StorageSpec)(s.S3),
		s.Repo,
		s.SourcePath,
		s.ReadWrite,
		s.Path,
		s.Metadata,
	}
}

func ConvertStorageSpecListToV1beta2(ss ...StorageSpec) []v1beta2.StorageSpec {
	out := make([]v1beta2.StorageSpec, len(ss))
	for i, s := range ss {
		out[i] = ConvertStorageSpecToV1beta2(s)
	}
	return out
}

func ConvertJobHistoryListToV1beta2List(hh ...JobHistory) []v1beta2.JobHistory {
	out := make([]v1beta2.JobHistory, len(hh))
	for i, h := range hh {
		out[i] = ConvertJobHistoryToV1beta2(h)
	}
	return out
}

func ConvertJobHistoryToV1beta2(h JobHistory) v1beta2.JobHistory {
	out := v1beta2.JobHistory{ //nolint:govet
		v1beta2.JobHistoryType(h.Type),
		h.JobID,
		h.NodeID,
		h.ComputeReference,
		nil,
		nil,
		h.NewVersion,
		h.Comment,
		h.Time,
	}

	if h.JobState != nil {
		out.JobState = &v1beta2.StateChange[v1beta2.JobStateType]{ //nolint:govet
			v1beta2.JobStateType(h.JobState.Previous),
			v1beta2.JobStateType(h.JobState.New),
		}
	}

	if h.ExecutionState != nil {
		out.ExecutionState = &v1beta2.StateChange[v1beta2.ExecutionStateType]{ //nolint:govet
			v1beta2.ExecutionStateType(h.ExecutionState.Previous),
			v1beta2.ExecutionStateType(h.ExecutionState.New),
		}
	}

	return out
}

func ConvertPublishedResultToV1beta2(p PublishedResult) v1beta2.PublishedResult {
	return v1beta2.PublishedResult{ //nolint:govet
		p.NodeID,
		ConvertStorageSpecToV1beta2(p.Data),
	}
}

func ConvertPublishedResultListToV1beta2List(pp ...PublishedResult) []v1beta2.PublishedResult {
	out := make([]v1beta2.PublishedResult, len(pp))
	for i, p := range pp {
		out[i] = ConvertPublishedResultToV1beta2(p)
	}
	return out
}

func ConvertIncludeTagListToV1beta2(ii ...IncludedTag) []v1beta2.IncludedTag {
	out := make([]v1beta2.IncludedTag, len(ii))
	for i, t := range ii {
		out[i] = v1beta2.IncludedTag(t)
	}
	return out
}

func ConvertExcludeTagListToV1beta2(ee ...ExcludedTag) []v1beta2.ExcludedTag {
	out := make([]v1beta2.ExcludedTag, len(ee))
	for i, t := range ee {
		out[i] = v1beta2.ExcludedTag(t)
	}
	return out
}
