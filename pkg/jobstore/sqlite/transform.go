package sqlite

import (
	"encoding/json"

	"gorm.io/datatypes"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

func JobAndStateToDt(j Job, s JobState) (models.Job, error) {
	out := models.Job{
		ID:        j.JobID,
		Name:      j.Name,
		Namespace: j.Namespace,
		Type:      j.Type,
		Priority:  j.Priority,
		Count:     j.Count,
		State: models.State[models.JobStateType]{
			StateType: models.JobStateType(s.State),
			Message:   s.Message,
		},
		Version:    s.Version,
		Revision:   s.Revision,
		CreateTime: s.CreatedTime,
		ModifyTime: s.ModifiedTime,
	}
	var constraints []*models.LabelSelectorRequirement
	if err := json.Unmarshal(j.Constraints, &constraints); err != nil {
		panic(err)
	}
	out.Constraints = constraints

	var meta map[string]string
	meta, err := ToMap(j.Meta)
	if err != nil {
		return models.Job{}, err
	}
	out.Meta = meta

	labels, err := ToMap(j.Labels)
	if err != nil {
		return models.Job{}, err
	}
	out.Labels = labels

	tasks, err := TasksFromDt(j.Tasks)
	if err != nil {
		return models.Job{}, err
	}
	out.Tasks = tasks

	out.Normalize()
	return out, nil
}

func TasksToDt(jobID string, ts ...*models.Task) ([]Task, error) {
	var tasks []Task
	for _, dt := range ts {
		engine, err := SpecConfigToDt(dt.Engine)
		if err != nil {
			return nil, err
		}
		publisher, err := SpecConfigToDt(dt.Publisher)
		if err != nil {
			return nil, err
		}
		env, err := json.Marshal(dt.Env)
		if err != nil {
			return nil, err
		}
		meta, err := json.Marshal(dt.Meta)
		if err != nil {
			return nil, err
		}
		inputs, err := InputSourceToDt(dt.InputSources...)
		if err != nil {
			return nil, err
		}
		network, err := NetworkConfigToDt(dt.Network)
		if err != nil {
			return nil, err
		}
		// Convert each domain task to a GORM Task model
		// This includes converting any nested structures or fields as necessary
		task := Task{
			JobID:        jobID,
			Name:         dt.Name,
			Engine:       engine,
			Publisher:    publisher,
			Env:          datatypes.JSON(env),
			Meta:         datatypes.JSON(meta),
			InputSources: inputs,
			ResultPaths:  ToResultPathModel(dt.ResultPaths...),
			Resources: ResourceConfig{
				CPU:    dt.ResourcesConfig.CPU,
				Memory: dt.ResourcesConfig.Memory,
				Disk:   dt.ResourcesConfig.Disk,
				GPU:    dt.ResourcesConfig.CPU,
			},
			Network:  network,
			Timeouts: TimeoutConfig{ExecutionTimeout: dt.Timeouts.ExecutionTimeout},
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}

func TasksFromDt(m []Task) ([]*models.Task, error) {
	var tasks []*models.Task
	for _, dt := range m {
		engine, err := ToSpecConfig(dt.Engine)
		if err != nil {
			return nil, err
		}
		publisher, err := ToSpecConfig(dt.Publisher)
		if err != nil {
			return nil, err
		}
		env, err := ToMap(dt.Env)
		if err != nil {
			return nil, err
		}
		meta, err := ToMap(dt.Meta)
		if err != nil {
			return nil, err
		}
		inputs, err := ToInputSources(dt.InputSources...)
		if err != nil {
			return nil, err
		}
		network, err := ToNetworkConfig(dt.Network)
		task := &models.Task{
			Name:         dt.Name,
			Engine:       engine,
			Publisher:    publisher,
			Env:          env,
			Meta:         meta,
			InputSources: inputs,
			ResultPaths:  ToResultPaths(dt.ResultPaths...),
			ResourcesConfig: &models.ResourcesConfig{
				CPU:    dt.Resources.CPU,
				Memory: dt.Resources.Memory,
				Disk:   dt.Resources.Disk,
				GPU:    dt.Resources.GPU,
			},
			Network:  network,
			Timeouts: &models.TimeoutConfig{ExecutionTimeout: dt.Timeouts.ExecutionTimeout},
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}

func SpecConfigToDt(s *models.SpecConfig) (SpecConfig, error) {
	if s == nil {
		return SpecConfig{}, nil
	}
	out := SpecConfig{
		Type: s.Type,
	}
	if s.Params != nil {
		pb, err := json.Marshal(s.Params)
		if err != nil {
			return SpecConfig{}, err
		}
		out.Params = pb
	}
	return out, nil
}

func ToSpecConfig(s SpecConfig) (*models.SpecConfig, error) {
	out := &models.SpecConfig{
		Type: s.Type,
	}
	if s.Params != nil {
		var p map[string]interface{}
		if err := json.Unmarshal(s.Params, &p); err != nil {
			return nil, err
		}
		out.Params = p
	}
	out.Normalize()
	return out, nil
}

func ToInputSources(s ...InputSource) ([]*models.InputSource, error) {
	var out []*models.InputSource
	for _, i := range s {
		source, err := ToSpecConfig(i.Source)
		if err != nil {
			return nil, err
		}
		out = append(out, &models.InputSource{
			Source: source,
			Alias:  i.Alias,
			Target: i.Target,
		})
	}
	return out, nil
}

func ToResultPaths(r ...ResultPath) []*models.ResultPath {
	var out []*models.ResultPath
	for _, i := range r {
		out = append(out, &models.ResultPath{
			Name: i.Name,
			Path: i.Path,
		})
	}
	return out
}

func ToNetworkConfig(n NetworkConfig) (*models.NetworkConfig, error) {
	out := new(models.NetworkConfig)
	ntype, err := models.ParseNetwork(n.Type)
	if err != nil {
		return nil, err
	}
	var domains []string
	if err := json.Unmarshal(n.Domains, &domains); err != nil {
		return nil, err
	}
	out.Type = ntype
	out.Domains = domains
	return out, nil
}

func NetworkConfigToDt(n *models.NetworkConfig) (NetworkConfig, error) {
	db, err := json.Marshal(n.Domains)
	if err != nil {
		return NetworkConfig{}, err
	}
	return NetworkConfig{
		Type:    n.Type.String(),
		Domains: db,
	}, nil
}

func InputSourceToDt(sources ...*models.InputSource) ([]InputSource, error) {
	var inputs []InputSource
	for _, di := range sources {
		source, err := SpecConfigToDt(di.Source)
		if err != nil {
			return nil, err
		}
		i := InputSource{
			Alias:  di.Alias,
			Target: di.Target,
			Source: source,
		}
		inputs = append(inputs, i)
	}
	return inputs, nil
}

func ToResultPathModel(rps ...*models.ResultPath) []ResultPath {
	var results []ResultPath
	for _, dr := range rps {
		r := ResultPath{
			Name: dr.Name,
			Path: dr.Path,
		}
		results = append(results, r)
	}
	return results
}

func ToMap(m datatypes.JSON) (map[string]string, error) {
	var out map[string]string
	if err := json.Unmarshal(m, &out); err != nil {
		return nil, err
	}
	return out, nil
}
