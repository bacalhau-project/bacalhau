//go:generate stringer -type=JobStateType --trimprefix=JobStateType --output job_string.go
package models

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/exp/maps"

	"github.com/bacalhau-project/bacalhau/pkg/lib/validate"
)

type JobStateType int

const (
	JobStateTypeUndefined JobStateType = iota

	// JobStateTypePending is the state of a job that has been submitted but not
	// yet scheduled.
	JobStateTypePending

	// JobStateTypeRunning is the state of a job that has been scheduled, with at
	// least one active execution.
	JobStateTypeRunning

	// JobStateTypeCompleted is the state of a job that has successfully completed.
	// Only valid for batch jobs.
	JobStateTypeCompleted

	// JobStateTypeFailed is the state of a job that has failed.
	JobStateTypeFailed

	// JobStateTypeStopped is the state of a job that has been stopped by the user.
	JobStateTypeStopped
)

// IsUndefined returns true if the job state is undefined
func (s JobStateType) IsUndefined() bool {
	return s == JobStateTypeUndefined
}

func JobStateTypes() []JobStateType {
	var res []JobStateType
	for typ := JobStateTypePending; typ <= JobStateTypeStopped; typ++ {
		res = append(res, typ)
	}
	return res
}

func (s JobStateType) MarshalText() ([]byte, error) {
	return []byte(s.String()), nil
}

func (s *JobStateType) UnmarshalText(text []byte) (err error) {
	name := strings.TrimSpace(string(text))
	for _, typ := range JobStateTypes() {
		if strings.EqualFold(typ.String(), name) {
			*s = typ
			return
		}
	}
	return
}

type Job struct {
	// ID is a unique identifier assigned to this job.
	// It helps to distinguish jobs with the same name after they have been deleted and re-created.
	// The ID is generated by the server and should not be set directly by the client.
	ID string `json:"ID"`

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

	// State is the current state of the job.
	State State[JobStateType] `json:"Liveness"`

	// Version is a per-job monotonically increasing version number that is incremented
	// on each job specification update.
	Version uint64 `json:"Version"`

	// Revision is a per-job monotonically increasing revision number that is incremented
	// on each update to the job's state or specification
	Revision uint64 `json:"Revision"`

	CreateTime int64 `json:"CreateTime"`
	ModifyTime int64 `json:"ModifyTime"`
}

func (j *Job) MetricAttributes() []attribute.KeyValue {
	// TODO(forrest): will need to re-think how we tag metrics from jobs with more than one task when ever that happens.
	return append(j.Task().MetricAttributes(), attribute.String("job_type", j.Type))
}

func (j *Job) String() string {
	return j.ID
}

// NamespacedID returns the namespaced id useful for logging
func (j *Job) NamespacedID() NamespacedID {
	return NamespacedID{
		ID:        j.ID,
		Namespace: j.Namespace,
	}
}

// Normalize is used to canonicalize fields in the Job. This should be
// called when registering a Job.
func (j *Job) Normalize() {
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

// Copy returns a deep copy of the Job. It is expected that callers use recover.
// This job can panic if the deep copy failed as it uses reflection.
func (j *Job) Copy() *Job {
	if j == nil {
		return nil
	}
	nj := new(Job)
	*nj = *j
	nj.Constraints = CopySlice[*LabelSelectorRequirement](nj.Constraints)

	if j.Tasks != nil {
		tasks := make([]*Task, len(nj.Tasks))
		for i, t := range nj.Tasks {
			tasks[i] = t.Copy()
		}
		nj.Tasks = tasks
	}

	nj.Meta = maps.Clone(nj.Meta)
	return nj
}

// Validate is used to check a job for reasonable configuration
func (j *Job) Validate() error {
	var mErr error

	// Validate the job ID
	if validate.IsBlank(j.ID) {
		mErr = errors.Join(mErr, errors.New("missing job ID"))
	} else if validate.ContainsSpaces(j.ID) {
		mErr = errors.Join(mErr, errors.New("job ID contains a space"))
	} else if validate.ContainsNull(j.ID) {
		mErr = errors.Join(mErr, errors.New("job ID contains a null character"))
	}

	// Validate the job name
	if validate.IsBlank(j.Name) {
		mErr = errors.Join(mErr, errors.New("missing job name"))
	} else if validate.ContainsNull(j.Name) {
		mErr = errors.Join(mErr, errors.New("job Name contains a null character"))
	}

	// Validate the job namespace
	if validate.IsBlank(j.Namespace) {
		mErr = errors.Join(mErr, errors.New("job must be in a namespace"))
	}

	mErr = errors.Join(mErr, j.ValidateSubmission())

	// Validate the task group
	for _, task := range j.Tasks {
		if err := task.Validate(); err != nil {
			outer := fmt.Errorf("task %s validation failed: %v", task.Name, err)
			mErr = errors.Join(mErr, outer)
		}
	}
	return mErr
}

// ValidateSubmission is used to check a job for reasonable configuration when it is submitted.
// It is a subset of Validate that does not check fields with defaults, such as job ID
func (j *Job) ValidateSubmission() error {
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

// SanitizeSubmission is used to sanitize a job for reasonable configuration when it is submitted.
func (j *Job) SanitizeSubmission() (warnings []string) {
	if !j.State.StateType.IsUndefined() {
		warnings = append(warnings, "job state is ignored when submitting a job")
		j.State = NewJobState(JobStateTypeUndefined)
	}
	if j.Revision != 0 {
		warnings = append(warnings, "job revision is ignored when submitting a job")
		j.Revision = 0
	}
	if j.Version != 0 {
		warnings = append(warnings, "job version is ignored when submitting a job")
		j.Version = 0
	}
	if j.CreateTime != 0 {
		warnings = append(warnings, "job create time is ignored when submitting a job")
		j.CreateTime = 0
	}
	if j.ModifyTime != 0 {
		warnings = append(warnings, "job modify time is ignored when submitting a job")
		j.ModifyTime = 0
	}
	if j.Type == JobTypeBatch || j.Type == JobTypeOps {
		if j.ID != "" {
			warnings = append(warnings, "job ID is ignored when submitting a batch job")
			j.ID = ""
		}
	}
	// TODO: remove this once we have multiple tasks per job
	if len(j.Tasks) > 1 {
		warnings = append(warnings, "only one task is supported per job")
		j.Tasks = j.Tasks[:1]
	}
	for k := range j.Meta {
		if strings.HasPrefix(k, MetaReservedPrefix) {
			warnings = append(warnings, fmt.Sprintf("job meta key %q is reserved and will be ignored", k))
			delete(j.Meta, k)
		}
	}
	return warnings
}

// IsTerminal returns true if the job is in a terminal state
func (j *Job) IsTerminal() bool {
	switch j.State.StateType {
	case JobStateTypeCompleted, JobStateTypeFailed, JobStateTypeStopped:
		return true
	default:
		return false
	}
}

// Task returns the job task
// TODO: remove this once we have multiple tasks per job
func (j *Job) Task() *Task {
	if j == nil {
		return nil
	}
	return j.Tasks[0]
}

// GetCreateTime returns the creation time
func (j *Job) GetCreateTime() time.Time {
	return time.Unix(0, j.CreateTime).UTC()
}

// GetModifyTime returns the modify time
func (j *Job) GetModifyTime() time.Time {
	return time.Unix(0, j.ModifyTime).UTC()
}

// AllStorageTypes returns keys of all storage types required by the job
func (j *Job) AllStorageTypes() []string {
	var storageTypes []string
	if j == nil {
		return storageTypes
	}
	for _, task := range j.Tasks {
		storageTypes = append(storageTypes, task.AllStorageTypes()...)
	}
	return storageTypes
}

// IsLongRunning returns true if the job is long running
func (j *Job) IsLongRunning() bool {
	return j.Type == JobTypeService || j.Type == JobTypeDaemon
}
