package model

import (
	"github.com/bacalhau-project/bacalhau/pkg/model/v1beta2"
)

//
// Forwards v1beta2 -> current
//

func ConvertV1beta2StorageSpec(s v1beta2.StorageSpec) StorageSpec {
	return StorageSpec{
		StorageSource: StorageSourceType(s.StorageSource),
		Name:          s.Name,
		CID:           s.CID,
		URL:           s.URL,
		S3:            (*S3StorageSpec)(s.S3),
		Repo:          s.Repo,
		SourcePath:    s.SourcePath,
		ReadWrite:     s.ReadWrite,
		Path:          s.Path,
		Metadata:      s.Metadata,
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
		out[i] = LabelSelectorRequirement{
			Key:      l.Key,
			Operator: l.Operator,
			Values:   l.Values,
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
	return ExecutionState{
		JobID:                e.JobID,
		NodeID:               e.NodeID,
		ComputeReference:     e.ComputeReference,
		State:                ExecutionStateType(e.State),
		AcceptedAskForBid:    e.AcceptedAskForBid,
		Status:               e.Status,
		VerificationProposal: e.VerificationProposal,
		VerificationResult:   VerificationResult(e.VerificationResult),
		PublishedResult:      ConvertV1beta2StorageSpec(e.PublishedResult),
		RunOutput:            (*RunCommandResult)(e.RunOutput),
		Version:              e.Version,
		CreateTime:           e.CreateTime,
		UpdateTime:           e.UpdateTime,
	}
}

func ConvertV1beta2Spec(s v1beta2.Spec) Spec {
	return Spec{
		Engine:    Engine(s.Engine),
		Verifier:  Verifier(s.Verifier),
		Publisher: Publisher(s.Publisher),
		PublisherSpec: PublisherSpec{
			Type:   Publisher(s.PublisherSpec.Type),
			Params: s.PublisherSpec.Params,
		},
		Docker: JobSpecDocker{
			Image:                s.Docker.Image,
			Entrypoint:           s.Docker.Entrypoint,
			Parameters:           s.Docker.Parameters,
			EnvironmentVariables: s.Docker.EnvironmentVariables,
			WorkingDirectory:     s.Docker.WorkingDirectory,
		},
		Wasm: JobSpecWasm{
			EntryModule:          ConvertV1beta2StorageSpec(s.Wasm.EntryModule),
			EntryPoint:           s.Wasm.EntryPoint,
			Parameters:           s.Wasm.Parameters,
			EnvironmentVariables: s.Wasm.EnvironmentVariables,
			ImportModules:        ConvertV1beta2StorageSpecList(s.Wasm.ImportModules...),
		},
		Resources: ResourceUsageConfig(s.Resources),
		Network: NetworkConfig{
			Type:    Network(s.Network.Type),
			Domains: s.Network.Domains,
		},
		Timeout:       s.Timeout,
		Inputs:        ConvertV1beta2StorageSpecList(s.Inputs...),
		Outputs:       ConvertV1beta2StorageSpecList(s.Outputs...),
		Annotations:   s.Annotations,
		NodeSelectors: ConvertV1beta2LabelSelectorRequirementList(s.NodeSelectors...),
		DoNotTrack:    s.DoNotTrack,
		Deal: Deal{
			TargetingMode: TargetingMode(s.Deal.TargetingMode),
			Concurrency:   s.Deal.Concurrency,
			Confidence:    s.Deal.Confidence,
			MinBids:       s.Deal.MinBids,
		},
	}
}

func ConvertV1beta2Job(j v1beta2.Job) Job {
	return Job{
		APIVersion: APIVersionLatest().String(),
		Metadata: Metadata{
			ID:        j.Metadata.ID,
			CreatedAt: j.Metadata.CreatedAt,
			ClientID:  j.Metadata.ClientID,
			Requester: JobRequester{
				RequesterNodeID:    j.Metadata.Requester.RequesterNodeID,
				RequesterPublicKey: PublicKey(j.Metadata.Requester.RequesterPublicKey),
			},
		},
		Spec: ConvertV1beta2Spec(j.Spec),
	}
}

func ConvertV1beta2IncludeTagList(ii ...v1beta2.IncludedTag) []IncludedTag {
	out := make([]IncludedTag, len(ii))
	for i, t := range ii {
		out[i] = IncludedTag(t)
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

func ConvertV1beta2ExcludeTagList(ee ...v1beta2.ExcludedTag) []ExcludedTag {
	out := make([]ExcludedTag, len(ee))
	for i, t := range ee {
		out[i] = ExcludedTag(t)
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

//
// Backwards current -> v1beta2
//

func ConvertJobToV1beta2(j Job) v1beta2.Job {
	return v1beta2.Job{
		APIVersion: j.APIVersion,
		Metadata: v1beta2.Metadata{
			ID:        j.Metadata.ID,
			CreatedAt: j.Metadata.CreatedAt,
			ClientID:  j.Metadata.ClientID,
			Requester: v1beta2.JobRequester{
				RequesterNodeID:    j.Metadata.Requester.RequesterNodeID,
				RequesterPublicKey: v1beta2.PublicKey(j.Metadata.Requester.RequesterPublicKey),
			},
		},
		Spec: v1beta2.Spec{
			Engine:    v1beta2.Engine(j.Spec.Engine),
			Verifier:  v1beta2.Verifier(j.Spec.Verifier),
			Publisher: v1beta2.Publisher(j.Spec.Publisher),
			PublisherSpec: v1beta2.PublisherSpec{
				Type:   v1beta2.Publisher(j.Spec.PublisherSpec.Type),
				Params: j.Spec.PublisherSpec.Params,
			},
			Docker: v1beta2.JobSpecDocker{
				Image:                j.Spec.Docker.Image,
				Entrypoint:           j.Spec.Docker.Entrypoint,
				Parameters:           j.Spec.Docker.Parameters,
				EnvironmentVariables: j.Spec.Docker.EnvironmentVariables,
				WorkingDirectory:     j.Spec.Docker.WorkingDirectory,
			},
			Wasm: v1beta2.JobSpecWasm{
				EntryModule:          ConvertStorageSpecToV1beta2(j.Spec.Wasm.EntryModule),
				EntryPoint:           j.Spec.Wasm.EntryPoint,
				Parameters:           j.Spec.Wasm.Parameters,
				EnvironmentVariables: j.Spec.Wasm.EnvironmentVariables,
				ImportModules:        ConvertStorageSpecListToV1beta2(j.Spec.Wasm.ImportModules...),
			},
			Resources: v1beta2.ResourceUsageConfig(j.Spec.Resources),
			Network: v1beta2.NetworkConfig{
				Type:    v1beta2.Network(j.Spec.Network.Type),
				Domains: j.Spec.Network.Domains,
			},
			Timeout:       j.Spec.Timeout,
			Inputs:        ConvertStorageSpecListToV1beta2(j.Spec.Inputs...),
			Outputs:       ConvertStorageSpecListToV1beta2(j.Spec.Outputs...),
			Annotations:   j.Spec.Annotations,
			NodeSelectors: ConvertLabelSelectorRequirementListToV1beta2(j.Spec.NodeSelectors...),
			DoNotTrack:    j.Spec.DoNotTrack,
			Deal: v1beta2.Deal{
				TargetingMode: v1beta2.TargetingMode(j.Spec.Deal.TargetingMode),
				Concurrency:   j.Spec.Deal.Concurrency,
				Confidence:    j.Spec.Deal.Confidence,
				MinBids:       j.Spec.Deal.MinBids,
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
	return v1beta2.JobState{
		JobID:      s.JobID,
		Executions: ConvertExecutionStateToV1beta2List(s.Executions...),
		State:      v1beta2.JobStateType(s.State),
		Version:    s.Version,
		CreateTime: s.CreateTime,
		UpdateTime: s.UpdateTime,
		TimeoutAt:  s.TimeoutAt,
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
	return v1beta2.ExecutionState{
		JobID:                e.JobID,
		NodeID:               e.NodeID,
		ComputeReference:     e.ComputeReference,
		State:                v1beta2.ExecutionStateType(e.State),
		AcceptedAskForBid:    e.AcceptedAskForBid,
		Status:               e.Status,
		VerificationProposal: e.VerificationProposal,
		VerificationResult:   v1beta2.VerificationResult(e.VerificationResult),
		PublishedResult:      ConvertStorageSpecToV1beta2(e.PublishedResult),
		RunOutput:            (*v1beta2.RunCommandResult)(e.RunOutput),
		Version:              e.Version,
		CreateTime:           e.CreateTime,
		UpdateTime:           e.UpdateTime,
	}
}

func ConvertLabelSelectorRequirementListToV1beta2(ll ...LabelSelectorRequirement) []v1beta2.LabelSelectorRequirement {
	out := make([]v1beta2.LabelSelectorRequirement, len(ll))
	for i, l := range ll {
		out[i] = v1beta2.LabelSelectorRequirement{
			Key:      l.Key,
			Operator: l.Operator,
			Values:   l.Values,
		}
	}
	return out
}

func ConvertStorageSpecToV1beta2(s StorageSpec) v1beta2.StorageSpec {
	return v1beta2.StorageSpec{
		StorageSource: v1beta2.StorageSourceType(s.StorageSource),
		Name:          s.Name,
		CID:           s.CID,
		URL:           s.URL,
		S3:            (*v1beta2.S3StorageSpec)(s.S3),
		Repo:          s.Repo,
		SourcePath:    s.SourcePath,
		ReadWrite:     s.ReadWrite,
		Path:          s.Path,
		Metadata:      s.Metadata,
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
	out := v1beta2.JobHistory{
		Type:             v1beta2.JobHistoryType(h.Type),
		JobID:            h.JobID,
		NodeID:           h.NodeID,
		ComputeReference: h.ComputeReference,
		NewVersion:       h.NewVersion,
		Comment:          h.Comment,
		Time:             h.Time,
	}

	if h.JobState != nil {
		out.JobState = &v1beta2.StateChange[v1beta2.JobStateType]{
			Previous: v1beta2.JobStateType(h.JobState.Previous),
			New:      v1beta2.JobStateType(h.JobState.New),
		}
	}

	if h.ExecutionState != nil {
		out.ExecutionState = &v1beta2.StateChange[v1beta2.ExecutionStateType]{
			Previous: v1beta2.ExecutionStateType(h.ExecutionState.Previous),
			New:      v1beta2.ExecutionStateType(h.ExecutionState.New),
		}
	}

	return out
}

func ConvertPublishedResultToV1beta2(p PublishedResult) v1beta2.PublishedResult {
	return v1beta2.PublishedResult{
		NodeID: p.NodeID,
		Data:   ConvertStorageSpecToV1beta2(p.Data),
	}
}

func ConvertPublishedResultListToV1beta2List(pp ...PublishedResult) []v1beta2.PublishedResult {
	out := make([]v1beta2.PublishedResult, len(pp))
	for i, p := range pp {
		out[i] = ConvertPublishedResultToV1beta2(p)
	}
	return out
}

func ConvertV1beta2PublishedResult(p v1beta2.PublishedResult) PublishedResult {
	return PublishedResult{
		NodeID: p.NodeID,
		Data:   ConvertV1beta2StorageSpec(p.Data),
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
	return JobState{
		JobID:      js.JobID,
		Executions: ConvertV1beta2ExecutionStateList(js.Executions...),
		State:      JobStateType(js.State),
		Version:    js.Version,
		CreateTime: js.CreateTime,
		UpdateTime: js.UpdateTime,
		TimeoutAt:  js.TimeoutAt,
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
	return &JobWithInfo{
		Job:     ConvertV1beta2Job(jwi.Job),
		State:   ConvertV1beta2JobState(jwi.State),
		History: ConvertV1beta2JobHistoryListToJobHistoryList(jwi.History...),
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
	out := JobHistory{
		Type:             JobHistoryType(h.Type),
		JobID:            h.JobID,
		NodeID:           h.NodeID,
		ComputeReference: h.ComputeReference,
		NewVersion:       h.NewVersion,
		Comment:          h.Comment,
		Time:             h.Time,
	}

	if h.JobState != nil {
		out.JobState = &StateChange[JobStateType]{
			Previous: JobStateType(h.JobState.Previous),
			New:      JobStateType(h.JobState.New),
		}
	}

	if h.ExecutionState != nil {
		out.ExecutionState = &StateChange[ExecutionStateType]{
			Previous: ExecutionStateType(h.ExecutionState.Previous),
			New:      ExecutionStateType(h.ExecutionState.New),
		}
	}
	return out
}

func ConvertV1beta2NodeInfo(ni v1beta2.NodeInfo) NodeInfo {
	return NodeInfo{
		BacalhauVersion: BuildVersionInfo(ni.BacalhauVersion),
		PeerInfo:        ni.PeerInfo,
		NodeType:        NodeType(ni.NodeType),
		Labels:          ni.Labels,
		ComputeNodeInfo: ConvertV1beta2ComputeNodeInfo(ni.ComputeNodeInfo),
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
	return &ComputeNodeInfo{
		ExecutionEngines:   ConvertV1beta2EngineList(cni.ExecutionEngines...),
		Verifiers:          ConvertV1beta2VerifierList(cni.Verifiers...),
		Publishers:         ConvertV1beta2PublisherList(cni.Publishers...),
		StorageSources:     ConvertV1beta2StorageSourceTypeList(cni.StorageSources...),
		MaxCapacity:        ResourceUsageData(cni.MaxCapacity),
		AvailableCapacity:  ResourceUsageData(cni.AvailableCapacity),
		MaxJobRequirements: ResourceUsageData(cni.MaxJobRequirements),
		RunningExecutions:  cni.RunningExecutions,
		EnqueuedExecutions: cni.EnqueuedExecutions,
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
