package legacy

import (
	"fmt"
	"strings"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

func ToLegacyJob(job *models.Job) (*model.Job, error) {
	pk := new(model.PublicKey)
	err := pk.UnmarshalText([]byte(job.Meta[models.MetaRequesterPublicKey]))
	if err != nil {
		return nil, err
	}

	spec, err := ToLegacyJobSpec(job)
	if err != nil {
		return nil, err
	}

	legacy := &model.Job{
		APIVersion: model.V1beta2.String(),
		Metadata: model.Metadata{
			ID:        job.ID,
			CreatedAt: time.Unix(0, job.CreateTime),
			ClientID:  job.Meta[models.MetaClientID],
			Requester: model.JobRequester{
				RequesterNodeID:    job.Meta[models.MetaRequesterID],
				RequesterPublicKey: *pk,
			},
		},
		Spec: *spec,
	}
	return legacy, nil
}

func ToLegacyJobSpec(job *models.Job) (*model.Spec, error) {
	inputs := make([]model.StorageSpec, 0, len(job.Task().InputSources))
	for _, input := range job.Task().InputSources {
		source, err := ToLegacyStorageSpec(input.Source)
		if err != nil {
			return nil, fmt.Errorf("failed to convert input: %w", err)
		}
		source.Path = input.Target
		source.Name = input.Alias
		inputs = append(inputs, source)
	}

	outputs := make([]model.StorageSpec, 0, len(job.Task().ResultPaths))
	for _, output := range job.Task().ResultPaths {
		outputs = append(outputs, model.StorageSpec{
			Name: output.Name,
			Path: output.Path,
		})
	}

	annotations := make([]string, 0, len(job.Labels))
	for k := range job.Labels {
		annotations = append(annotations, k)
	}

	publisherSpec := model.PublisherSpec{}
	if job.Task().Publisher.Type != "" {
		publisherType, err := model.ParsePublisher(job.Task().Publisher.Type)
		if err != nil {
			return nil, err
		}
		publisherSpec = model.PublisherSpec{
			Type:   publisherType,
			Params: job.Task().Publisher.Params,
		}
	}

	networkConfig, err := ToLegacyNetworkConfig(job.Task().Network)
	if err != nil {
		return nil, err
	}

	deal := model.Deal{
		Concurrency:   job.Count,
		TargetingMode: model.TargetAny,
	}
	if job.Type == models.JobTypeOps {
		deal.TargetingMode = model.TargetAll
	}

	legacy := &model.Spec{
		EngineSpec: model.EngineSpec{
			Type:   job.Task().Engine.Type,
			Params: job.Task().Engine.Params,
		},
		PublisherSpec: publisherSpec,
		Resources: model.ResourceUsageConfig{
			CPU:    job.Task().ResourcesConfig.CPU,
			Memory: job.Task().ResourcesConfig.Memory,
			Disk:   job.Task().ResourcesConfig.Disk,
			GPU:    job.Task().ResourcesConfig.GPU,
		},
		Network:       networkConfig,
		Timeout:       job.Task().Timeouts.ExecutionTimeout,
		Inputs:        inputs,
		Outputs:       outputs,
		Annotations:   annotations,
		NodeSelectors: ToLegacyNodeSelectors(job.Constraints),
		Deal:          deal,
	}

	return legacy, nil
}

func ToLegacyNetworkConfig(network *models.NetworkConfig) (model.NetworkConfig, error) {
	var typ model.Network
	switch network.Type {
	case models.NetworkNone:
		typ = model.NetworkNone
	case models.NetworkFull:
		typ = model.NetworkFull
	case models.NetworkHTTP:
		typ = model.NetworkHTTP
	default:
		return model.NetworkConfig{}, fmt.Errorf("unhandled network type: %s", network.Type)
	}
	return model.NetworkConfig{
		Type:    typ,
		Domains: network.Domains,
	}, nil
}

func ToLegacyStorageSpec(storage *models.SpecConfig) (model.StorageSpec, error) {
	if storage == nil || storage.Type == "" {
		return model.StorageSpec{}, nil
	}
	switch storage.Type {
	case models.StorageSourceIPFS:
		return model.StorageSpec{
			StorageSource: model.StorageSourceIPFS,
			CID:           storage.Params["CID"].(string),
		}, nil
	case models.StorageSourceURL:
		return model.StorageSpec{
			StorageSource: model.StorageSourceURLDownload,
			URL:           storage.Params["URL"].(string),
		}, nil
	case models.StorageSourceRepoClone:
		return model.StorageSpec{
			StorageSource: model.StorageSourceRepoClone,
			Repo:          storage.Params["Repo"].(string),
		}, nil
	case models.StorageSourceRepoCloneLFS:
		return model.StorageSpec{
			StorageSource: model.StorageSourceRepoCloneLFS,
			Repo:          storage.Params["Repo"].(string),
		}, nil
	case models.StorageSourceInline:
		return model.StorageSpec{
			StorageSource: model.StorageSourceInline,
			URL:           storage.Params["URL"].(string),
		}, nil
	case models.StorageSourceLocalDirectory:
		storageSpec := model.StorageSpec{
			StorageSource: model.StorageSourceLocalDirectory,
			Path:          storage.Params["SourcePath"].(string),
		}
		if readWrite, ok := storage.Params["ReadWrite"].(bool); ok {
			storageSpec.ReadWrite = readWrite
		}
		return storageSpec, nil
	case models.StorageSourceS3:
		s3Spec := &model.S3StorageSpec{
			Bucket: storage.Params["Bucket"].(string),
			Key:    storage.Params["Key"].(string),
		}
		if storage.Params["Region"] != nil {
			s3Spec.Region = storage.Params["Region"].(string)
		}
		if storage.Params["Endpoint"] != nil {
			s3Spec.Endpoint = storage.Params["Endpoint"].(string)
		}
		return model.StorageSpec{
			StorageSource: model.StorageSourceS3,
			S3:            s3Spec,
		}, nil
	default:
		return model.StorageSpec{}, fmt.Errorf("unhandled storage source type: %s", storage.Type)
	}
}

func ToLegacyNodeSelectors(constraints []*models.LabelSelectorRequirement) []model.LabelSelectorRequirement {
	res := make([]model.LabelSelectorRequirement, len(constraints))
	for i, c := range constraints {
		res[i] = model.LabelSelectorRequirement{
			Key:      c.Key,
			Operator: c.Operator,
			Values:   c.Values,
		}
	}
	return res
}

func ToLegacyJobStatus(job models.Job, executions []models.Execution) (*model.JobState, error) {
	executionStates := make([]model.ExecutionState, len(executions))
	for i := range executions {
		executionState, err := ToLegacyExecutionState(&executions[i])
		if err != nil {
			return nil, err
		}
		executionStates[i] = *executionState
	}

	return &model.JobState{
		JobID:      job.ID,
		Executions: executionStates,
		State:      ToLegacyJobStateType(job.State.StateType),
		Version:    int(job.Revision),
		CreateTime: time.Unix(0, job.CreateTime),
		UpdateTime: time.Unix(0, job.ModifyTime),
		TimeoutAt:  time.Unix(job.Task().Timeouts.ExecutionTimeout, job.CreateTime),
	}, nil
}

func ToLegacyExecutionState(execution *models.Execution) (*model.ExecutionState, error) {
	publishedResult, err := ToLegacyStorageSpec(execution.PublishedResult)
	if err != nil {
		return nil, err
	}

	return &model.ExecutionState{
		JobID:            execution.JobID,
		NodeID:           execution.NodeID,
		ComputeReference: execution.ID,
		State:            ToLegacyExecutionStateType(execution.ComputeState.StateType),
		DesiredState:     ToLegacyExecutionDesiredStateType(execution.DesiredState.StateType),
		Status:           strings.Join([]string{execution.ComputeState.Message, execution.DesiredState.Message}, ". "),
		PublishedResult:  publishedResult,
		RunOutput:        ToLegacyRunCommandResult(execution.RunOutput),
		Version:          int(execution.Revision),
		CreateTime:       time.Unix(0, execution.CreateTime),
		UpdateTime:       time.Unix(0, execution.ModifyTime),
	}, nil
}

// ToLegacyRunCommandResult converts a models.RunCommandResult to a model.RunCommandResult
func ToLegacyRunCommandResult(result *models.RunCommandResult) *model.RunCommandResult {
	if result == nil {
		return nil
	}
	return &model.RunCommandResult{
		STDOUT:          result.STDOUT,
		StdoutTruncated: result.StdoutTruncated,
		STDERR:          result.STDERR,
		StderrTruncated: result.StderrTruncated,
		ExitCode:        result.ExitCode,
		ErrorMsg:        result.ErrorMsg,
	}
}

// ToLegacyJobHistory converts a models.JobHistory to a model.JobHistory
func ToLegacyJobHistory(history *models.JobHistory) *model.JobHistory {
	return &model.JobHistory{
		Type:             ToLegacyJobHistoryType(history.Type),
		JobID:            history.JobID,
		NodeID:           history.NodeID,
		ComputeReference: history.ExecutionID,
		JobState:         ToLegacyStateChange[models.JobStateType, model.JobStateType](history.JobState, ToLegacyJobStateType),
		ExecutionState: ToLegacyStateChange[models.ExecutionStateType, model.ExecutionStateType](
			history.ExecutionState, ToLegacyExecutionStateType),
		NewVersion: int(history.NewRevision),
		Comment:    history.Comment,
		Time:       history.Time,
	}
}

func ToLegacyStateChange[From any, To any](state *models.StateChange[From], converter func(From) To) *model.StateChange[To] {
	if state == nil {
		return nil
	}
	return &model.StateChange[To]{
		Previous: converter(state.Previous),
		New:      converter(state.New),
	}
}

func ToLegacyJobStateType(state models.JobStateType) model.JobStateType {
	switch state {
	case models.JobStateTypePending:
		return model.JobStateNew
	case models.JobStateTypeRunning:
		return model.JobStateInProgress
	case models.JobStateTypeFailed:
		return model.JobStateError
	case models.JobStateTypeCompleted:
		return model.JobStateCompleted
	case models.JobStateTypeStopped:
		return model.JobStateCancelled
	default:
		return model.JobStateUndefined
	}
}

func ToLegacyExecutionStateType(state models.ExecutionStateType) model.ExecutionStateType {
	switch state {
	case models.ExecutionStateNew:
		return model.ExecutionStateNew
	case models.ExecutionStateAskForBid:
		return model.ExecutionStateAskForBid
	case models.ExecutionStateAskForBidAccepted:
		return model.ExecutionStateAskForBidAccepted
	case models.ExecutionStateAskForBidRejected:
		return model.ExecutionStateAskForBidRejected
	case models.ExecutionStateBidAccepted:
		return model.ExecutionStateBidAccepted
	case models.ExecutionStateBidRejected:
		return model.ExecutionStateBidRejected
	case models.ExecutionStateCompleted:
		return model.ExecutionStateCompleted
	case models.ExecutionStateFailed:
		return model.ExecutionStateFailed
	case models.ExecutionStateCancelled:
		return model.ExecutionStateCancelled
	default:
		return model.ExecutionStateUndefined
	}
}

func ToLegacyExecutionDesiredStateType(state models.ExecutionDesiredStateType) model.ExecutionDesiredState {
	switch state {
	case models.ExecutionDesiredStatePending:
		return model.ExecutionDesiredStatePending
	case models.ExecutionDesiredStateRunning:
		return model.ExecutionDesiredStateRunning
	case models.ExecutionDesiredStateStopped:
		return model.ExecutionDesiredStateStopped
	default:
		return model.ExecutionDesiredStatePending
	}
}

func ToLegacyJobHistoryType(historyType models.JobHistoryType) model.JobHistoryType {
	switch historyType {
	case models.JobHistoryTypeExecutionLevel:
		return model.JobHistoryTypeExecutionLevel
	default:
		return model.JobHistoryTypeJobLevel
	}
}
