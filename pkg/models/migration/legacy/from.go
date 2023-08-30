package legacy

import (
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	s3helper "github.com/bacalhau-project/bacalhau/pkg/s3"
	"github.com/bacalhau-project/bacalhau/pkg/storage/inline"
	ipfs_storage "github.com/bacalhau-project/bacalhau/pkg/storage/ipfs"
	localdirectory "github.com/bacalhau-project/bacalhau/pkg/storage/local_directory"
	"github.com/bacalhau-project/bacalhau/pkg/storage/repo"
	"github.com/bacalhau-project/bacalhau/pkg/storage/url/urldownload"
)

// FromLegacyJob converts legacy job sped to current
func FromLegacyJob(legacy *model.Job) (*models.Job, error) {
	typ := models.JobTypeBatch
	if legacy.Spec.Deal.TargetingMode == model.TargetAll {
		typ = models.JobTypeOps
	}

	constraints, err := FromLegacyLabelSelector(legacy.Spec.NodeSelectors)
	if err != nil {
		return nil, err
	}

	metadata := make(map[string]string)
	metadata[models.MetaRequesterID] = legacy.Metadata.Requester.RequesterNodeID
	metadata[models.MetaRequesterPublicKey] = legacy.Metadata.Requester.RequesterPublicKey.String()
	metadata[models.MetaClientID] = legacy.Metadata.ClientID

	labels := make(map[string]string)
	for _, v := range legacy.Spec.Annotations {
		labels[v] = ""
	}

	task, err := FromLegacyJobSpec(legacy.Spec, typ)
	if err != nil {
		return nil, err
	}
	job := &models.Job{
		ID:          legacy.ID(),
		Name:        legacy.ID(),
		Namespace:   legacy.Metadata.ClientID,
		Type:        typ,
		Count:       legacy.Spec.Deal.Concurrency,
		Constraints: constraints,
		Meta:        metadata,
		Labels:      labels,
		Tasks:       []*models.Task{task},
		State:       models.NewJobState(models.JobStateTypeUndefined),
		Version:     1,
		Revision:    1,
		CreateTime:  legacy.Metadata.CreatedAt.UnixNano(),
		ModifyTime:  legacy.Metadata.CreatedAt.UnixNano(),
	}

	job.Normalize()
	return job, job.Validate()
}

func FromLegacyJobSpec(legacy model.Spec, jobtype string) (*models.Task, error) {
	inputs := make([]*models.InputSource, 0, len(legacy.Inputs))
	for i, input := range legacy.Inputs {
		source, err := FromLegacyStorageSpec(input)
		if err != nil {
			return nil, fmt.Errorf("failed to convert input %d: %w", i, err)
		}
		inputs = append(inputs, &models.InputSource{
			Source: source,
			Alias:  input.Name,
			Target: input.Path,
		})
	}

	outputs := make([]*models.ResultPath, 0, len(legacy.Outputs))
	for _, output := range legacy.Outputs {
		outputs = append(outputs, &models.ResultPath{
			Name: output.Name,
			Path: output.Path,
		})
	}

	network, err := FromLegacyNetworkConfig(legacy.Network)
	if err != nil {
		return nil, err
	}

	task := &models.Task{
		Name: "main",
		Engine: &models.SpecConfig{
			Type:   legacy.EngineSpec.Type,
			Params: legacy.EngineSpec.Params,
		},
		Publisher: &models.SpecConfig{
			Type:   legacy.PublisherSpec.Type.String(),
			Params: legacy.PublisherSpec.Params,
		},
		InputSources: inputs,
		ResultPaths:  outputs,
		ResourcesConfig: &models.ResourcesConfig{
			CPU:    legacy.Resources.CPU,
			Memory: legacy.Resources.Memory,
			Disk:   legacy.Resources.Disk,
			GPU:    legacy.Resources.GPU,
		},
		Network: network,
		Timeouts: &models.TimeoutConfig{
			ExecutionTimeout: legacy.Timeout,
		},
		RestartPolicy: models.NewRestartPolicy(jobtype),
	}
	return task, nil
}

func FromLegacyStorageSpec(legacy model.StorageSpec) (*models.SpecConfig, error) {
	var res *models.SpecConfig
	switch legacy.StorageSource {
	case model.StorageSourceIPFS:
		res = &models.SpecConfig{
			Type: models.StorageSourceIPFS,
			Params: ipfs_storage.Source{
				CID: legacy.CID,
			}.ToMap(),
		}
	case model.StorageSourceURLDownload:
		res = &models.SpecConfig{
			Type: models.StorageSourceURL,
			Params: urldownload.Source{
				URL: legacy.URL,
			}.ToMap(),
		}
	case model.StorageSourceRepoClone:
		res = &models.SpecConfig{
			Type: models.StorageSourceRepoClone,
			Params: repo.Source{
				Repo: legacy.Repo,
			}.ToMap(),
		}
	case model.StorageSourceRepoCloneLFS:
		res = &models.SpecConfig{
			Type: models.StorageSourceRepoCloneLFS,
			Params: repo.Source{
				Repo: legacy.Repo,
			}.ToMap(),
		}
	case model.StorageSourceInline:
		res = &models.SpecConfig{
			Type: models.StorageSourceInline,
			Params: inline.Source{
				URL: legacy.URL,
			}.ToMap(),
		}
	case model.StorageSourceLocalDirectory:
		res = &models.SpecConfig{
			Type: models.StorageSourceLocalDirectory,
			Params: localdirectory.Source{
				SourcePath: legacy.Path,
				ReadWrite:  legacy.ReadWrite,
			}.ToMap(),
		}
	case model.StorageSourceS3:
		res = &models.SpecConfig{
			Type: models.StorageSourceS3,
			Params: s3helper.SourceSpec{
				Bucket:         legacy.S3.Bucket,
				Key:            legacy.S3.Key,
				Region:         legacy.S3.Region,
				Endpoint:       legacy.S3.Endpoint,
				ChecksumSHA256: legacy.S3.ChecksumSHA256,
			}.ToMap(),
		}
	default:
		return nil, fmt.Errorf("unhandled storage spec: %s", legacy.StorageSource)
	}
	return res, nil
}

func FromLegacyStorageSpecToInputSource(spec model.StorageSpec) (*models.InputSource, error) {
	source, err := FromLegacyStorageSpec(spec)
	if err != nil {
		return nil, err
	}

	return &models.InputSource{
		Source: source,
		Alias:  spec.Name,
		Target: spec.Path,
	}, nil
}

func FromLegacyLabelSelector(legacy []model.LabelSelectorRequirement) ([]*models.LabelSelectorRequirement, error) {
	res := make([]*models.LabelSelectorRequirement, len(legacy))
	for i, l := range legacy {
		res[i] = &models.LabelSelectorRequirement{
			Key:      l.Key,
			Operator: l.Operator,
			Values:   l.Values,
		}
		err := res[i].Validate()
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

func FromLegacyNetworkConfig(legacy model.NetworkConfig) (*models.NetworkConfig, error) {
	var typ models.Network
	switch legacy.Type {
	case model.NetworkNone:
		typ = models.NetworkNone
	case model.NetworkFull:
		typ = models.NetworkFull
	case model.NetworkHTTP:
		typ = models.NetworkHTTP
	default:
		return nil, fmt.Errorf("unhandled network type: %s", legacy.Type)
	}
	return &models.NetworkConfig{
		Type:    typ,
		Domains: legacy.Domains,
	}, nil
}

func FromLegacyResourceUsageConfig(legacy model.ResourceUsageConfig) models.ResourcesConfig {
	return models.ResourcesConfig{
		CPU:    legacy.CPU,
		Memory: legacy.Memory,
		Disk:   legacy.Disk,
		GPU:    legacy.GPU,
	}
}

func FromLegacyJobStateType(legacy model.JobStateType) models.JobStateType {
	switch legacy {
	case model.JobStateNew:
		return models.JobStateTypePending
	case model.JobStateInProgress:
		return models.JobStateTypeRunning
	case model.JobStateCancelled:
		return models.JobStateTypeStopped
	case model.JobStateError:
		return models.JobStateTypeFailed
	case model.JobStateCompleted:
		return models.JobStateTypeCompleted
	default:
		return models.JobStateTypeUndefined
	}
}

func FromLegacyJobHistoryType(historyType model.JobHistoryType) models.JobHistoryType {
	switch historyType {
	case model.JobHistoryTypeExecutionLevel:
		return models.JobHistoryTypeExecutionLevel
	default:
		return models.JobHistoryTypeJobLevel
	}
}
