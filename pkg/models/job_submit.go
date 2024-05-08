package models

import (
	"errors"
	"fmt"
)

type JobSpec struct {
	// Name is the logical name of the job used to refer to it.
	// Submitting a job with the same name as an existing job will result in an
	// update to the existing job.
	Name string `json:"Name"`

	// Namespace is the namespace this job is running in.
	Namespace string `json:"Namespace"`

	// Type is the type of job this is, e.g. "daemon" or "batch".
	Type string `json:"Type"`

	// Priority defines the scheduling priority of this job.
	Priority int `json:"Priority"`

	// Count is the number of replicas that should be scheduled.
	Count int `json:"Count"`

	// Constraints is a selector which must be true for the compute node to run this job.
	Constraints []*LabelSelectorRequirement `json:"Constraints"`

	// Meta is used to associate arbitrary metadata with this job.
	Meta map[string]string `json:"Meta"`

	// Labels is used to associate arbitrary labels with this job, which can be used
	// for filtering.
	Labels map[string]string `json:"Labels"`

	Tasks []*Task `json:"Tasks"`
}

// ValidateSubmission is used to check a job for reasonable configuration when it is submitted.
// It is a subset of Validate that does not check fields with defaults, such as job ID
func (j *JobSpec) ValidateSubmission() error {
	if j == nil {
		return errors.New("empty/nil job")
	}

	var mErr error
	switch j.Type {
	case JobTypeService, JobTypeBatch, JobTypeDaemon, JobTypeOps:
	case "":
		mErr = errors.Join(mErr, errors.New("missing job type"))
	default:
		mErr = errors.Join(mErr, fmt.Errorf("invalid job type: %q", j.Type))
	}

	if j.Count < 0 {
		mErr = errors.Join(mErr, errors.New("job count must be >= 0"))
	}
	if len(j.Tasks) == 0 {
		mErr = errors.Join(mErr, errors.New("missing job tasks"))
	}
	for idx, constr := range j.Constraints {
		if err := constr.Validate(); err != nil {
			outer := fmt.Errorf("constraint %d validation failed: %s", idx+1, err)
			mErr = errors.Join(mErr, outer)
		}
	}

	// Validate the task group
	for _, task := range j.Tasks {
		if err := task.ValidateSubmission(); err != nil {
			outer := fmt.Errorf("task %s validation failed: %v", task.Name, err)
			mErr = errors.Join(mErr, outer)
		}
	}

	return mErr
}

// Normalize is used to canonicalize fields in the Job. This should be
// called when registering a Job.
func (j *JobSpec) Normalize() {
	if j == nil {
		return
	}

	// Ensure that an empty and nil map are treated the same to avoid scheduling
	// problems since we use reflect DeepEquals.
	if j.Meta == nil {
		j.Meta = make(map[string]string)
	}

	if j.Labels == nil {
		j.Labels = make(map[string]string)
	}

	if j.Constraints == nil {
		j.Constraints = make([]*LabelSelectorRequirement, 0)
	}

	if j.Tasks == nil {
		j.Tasks = make([]*Task, 0)
	}

	// Ensure the job is in a namespace.
	if j.Namespace == "" {
		j.Namespace = DefaultNamespace
	}

	if (j.Type == JobTypeDaemon || j.Type == JobTypeOps) && j.Count == 0 {
		j.Count = 1
	}

	for _, task := range j.Tasks {
		task.Normalize()
	}
}
