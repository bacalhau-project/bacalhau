package v1beta1

import (
	"github.com/filecoin-project/bacalhau/pkg/model/v1alpha1"
)

func ConvertV1alpha1StorageSpec(data v1alpha1.StorageSpec) StorageSpec {
	return StorageSpec{
		StorageSource: StorageSourceType(data.StorageSource),
		Name:          data.Name,
		CID:           data.CID,
		URL:           data.URL,
		Path:          data.Path,
		Metadata:      data.Metadata,
	}
}

func ConvertV1alpha1StorageSpecs(data []v1alpha1.StorageSpec) []StorageSpec {
	if data == nil {
		return nil
	}
	ret := []StorageSpec{}
	for _, spec := range data {
		ret = append(ret, ConvertV1alpha1StorageSpec(spec))
	}
	return ret
}

func ConvertV1alpha1Spec(
	data v1alpha1.Spec,
	executionPlan v1alpha1.JobExecutionPlan,
	deal v1alpha1.Deal,
) Spec {
	return Spec{
		Engine:    Engine(data.Engine),
		Verifier:  Verifier(data.Verifier),
		Publisher: Publisher(data.Publisher),
		Docker:    JobSpecDocker(data.Docker),
		Language: JobSpecLanguage{
			Language:         data.Language.Language,
			LanguageVersion:  data.Language.LanguageVersion,
			Deterministic:    data.Language.Deterministic,
			Context:          ConvertV1alpha1StorageSpec(data.Language.Context),
			Command:          data.Language.Command,
			ProgramPath:      data.Language.ProgramPath,
			RequirementsPath: data.Language.RequirementsPath,
		},
		Wasm: JobSpecWasm{
			EntryPoint:           data.Wasm.EntryPoint,
			Parameters:           data.Wasm.Parameters,
			EnvironmentVariables: data.Wasm.EnvironmentVariables,
			ImportModules:        ConvertV1alpha1StorageSpecs(data.Wasm.ImportModules),
		},
		Resources:     ResourceUsageConfig(data.Resources),
		Timeout:       data.Timeout,
		Inputs:        ConvertV1alpha1StorageSpecs(data.Inputs),
		Outputs:       ConvertV1alpha1StorageSpecs(data.Outputs),
		Contexts:      ConvertV1alpha1StorageSpecs(data.Contexts),
		Annotations:   data.Annotations,
		Sharding:      JobShardingConfig(data.Sharding),
		DoNotTrack:    data.DoNotTrack,
		ExecutionPlan: JobExecutionPlan(executionPlan),
		Deal:          Deal(deal),
	}
}

func ConvertV1alpha1RunCommandResult(data *v1alpha1.RunCommandResult) *RunCommandResult {
	var runOutput *RunCommandResult
	if data != nil {
		converted := RunCommandResult(*data)
		runOutput = &converted
	}
	return runOutput
}

func ConvertV1alpha1ShardState(data v1alpha1.JobShardState) JobShardState {
	return JobShardState{
		NodeID:               data.NodeID,
		ShardIndex:           data.ShardIndex,
		State:                JobStateType(data.State),
		Status:               data.Status,
		VerificationProposal: data.VerificationProposal,
		VerificationResult:   VerificationResult(data.VerificationResult),
		PublishedResult:      ConvertV1alpha1StorageSpec(data.PublishedResult),
		RunOutput:            ConvertV1alpha1RunCommandResult(data.RunOutput),
	}
}

func ConvertV1alpha1JobState(data v1alpha1.JobState) JobState {
	nodes := map[string]JobNodeState{}
	for nodeID, nodeData := range data.Nodes {
		shards := map[int]JobShardState{}
		for shardID, shardData := range nodeData.Shards {
			shards[shardID] = ConvertV1alpha1ShardState(shardData)
		}
		nodes[nodeID] = JobNodeState{
			Shards: shards,
		}
	}
	return JobState{
		Nodes: nodes,
	}
}

func ConvertV1alpha1JobEvent(event v1alpha1.JobEvent) JobEvent {
	return JobEvent{
		APIVersion:           APIVersionLatest().String(),
		JobID:                event.JobID,
		ShardIndex:           event.ShardIndex,
		ClientID:             event.ClientID,
		SourceNodeID:         event.SourceNodeID,
		TargetNodeID:         event.TargetNodeID,
		EventName:            JobEventType(event.EventName),
		Spec:                 ConvertV1alpha1Spec(event.Spec, event.JobExecutionPlan, event.Deal),
		JobExecutionPlan:     JobExecutionPlan(event.JobExecutionPlan),
		Deal:                 Deal(event.Deal),
		Status:               event.Status,
		VerificationProposal: event.VerificationProposal,
		VerificationResult:   VerificationResult(event.VerificationResult),
		PublishedResult:      ConvertV1alpha1StorageSpec(event.PublishedResult),
		EventTime:            event.EventTime,
		SenderPublicKey:      PublicKey(event.SenderPublicKey),
		RunOutput:            ConvertV1alpha1RunCommandResult(event.RunOutput),
	}
}

func ConvertV1alpha1JobEvents(events []v1alpha1.JobEvent) []JobEvent {
	if events == nil {
		return nil
	}
	ret := []JobEvent{}
	for _, event := range events {
		ret = append(ret, ConvertV1alpha1JobEvent(event))
	}
	return ret
}

func ConvertV1alpha1JobLocalEvent(event v1alpha1.JobLocalEvent) JobLocalEvent {
	return JobLocalEvent{
		EventName:    JobLocalEventType(event.EventName),
		JobID:        event.JobID,
		ShardIndex:   event.ShardIndex,
		TargetNodeID: event.TargetNodeID,
	}
}

func ConvertV1alpha1JobLocalEvents(events []v1alpha1.JobLocalEvent) []JobLocalEvent {
	if events == nil {
		return nil
	}
	ret := []JobLocalEvent{}
	for _, event := range events {
		ret = append(ret, ConvertV1alpha1JobLocalEvent(event))
	}
	return ret
}

func ConvertV1alpha1Job(data v1alpha1.Job) Job {
	return Job{
		APIVersion: APIVersionLatest().String(),
		Metadata: Metadata{
			ID:        data.ID,
			CreatedAt: data.CreatedAt,
			ClientID:  data.ClientID,
		},
		Spec: ConvertV1alpha1Spec(data.Spec, data.ExecutionPlan, data.Deal),
		Status: JobStatus{
			State:       ConvertV1alpha1JobState(data.State),
			Events:      ConvertV1alpha1JobEvents(data.Events),
			LocalEvents: ConvertV1alpha1JobLocalEvents(data.LocalEvents),
			Requester: JobRequester{
				RequesterNodeID:    data.RequesterNodeID,
				RequesterPublicKey: PublicKey(data.RequesterPublicKey),
			},
		},
	}
}
