//go:build unit || !integration

package orchestrator

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
	"github.com/google/uuid"
)

// MockJobTransformer for testing
type MockJobTransformer struct {
	TransformCalled bool
	TransformError  error
}

func (m *MockJobTransformer) Transform(ctx context.Context, job *models.Job) error {
	m.TransformCalled = true
	if m.TransformError != nil {
		return m.TransformError
	}

	// Ensure the job has valid times after transformation/normalization
	if job.CreateTime == 0 {
		job.CreateTime = time.Now().UnixNano()
	}
	if job.ModifyTime == 0 {
		job.ModifyTime = job.CreateTime
	}

	return nil
}

type EndpointTestSuite struct {
	suite.Suite
	ctrl               *gomock.Controller
	mockJobStore       *jobstore.MockStore
	mockTxCtx          *jobstore.MockTxContext
	mockJobTransformer *MockJobTransformer
	endpoint           *BaseEndpoint
}

func (s *EndpointTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.mockJobStore = jobstore.NewMockStore(s.ctrl)
	s.mockTxCtx = jobstore.NewMockTxContext(s.ctrl)
	s.mockJobTransformer = &MockJobTransformer{
		TransformCalled: false,
		TransformError:  nil,
	}

	s.endpoint = NewBaseEndpoint(&BaseEndpointParams{
		ID:                "test-endpoint",
		Store:             s.mockJobStore,
		LogstreamServer:   nil,
		JobTransformer:    s.mockJobTransformer,
		ResultTransformer: nil,
	})
}

func (s *EndpointTestSuite) TearDownTest() {
	s.ctrl.Finish()
}

func (s *EndpointTestSuite) TestRerunJob_Success() {
	ctx := context.Background()
	jobID := uuid.NewString()
	jobVersion := uint64(2)
	namespace := "default-namespace"

	// Create a job in a rerunnable state (completed)
	job := s.createTestJob(jobID, models.JobStateTypeCompleted)
	job.Version = jobVersion

	request := &RerunJobRequest{
		JobIDOrName: jobID,
		JobVersion:  0, // Use latest version
		Namespace:   namespace,
		Reason:      "Test rerun",
	}

	// Setup expectations
	s.mockJobStore.EXPECT().BeginTx(ctx).Return(s.mockTxCtx, nil)
	s.mockJobStore.EXPECT().GetJobByIDOrName(s.mockTxCtx, jobID, namespace).Return(job, nil)
	s.mockJobStore.EXPECT().UpdateJob(s.mockTxCtx, job).Return(nil)
	s.mockJobStore.EXPECT().UpdateJobState(s.mockTxCtx, gomock.Any()).Return(nil)
	s.mockJobStore.EXPECT().AddJobHistory(s.mockTxCtx, jobID, jobVersion+jobVersionIncrement, gomock.Any()).Return(nil)
	s.mockJobStore.EXPECT().CreateEvaluation(s.mockTxCtx, gomock.Any()).Return(nil)
	s.mockTxCtx.EXPECT().Commit().Return(nil)
	s.mockTxCtx.EXPECT().Rollback().Return(nil)

	response, err := s.endpoint.RerunJob(ctx, request)

	s.NoError(err)
	s.NotNil(response)
	s.Equal(jobID, response.JobID)
	s.Equal(jobVersion+jobVersionIncrement, response.JobVersion)
	s.NotEmpty(response.EvaluationID)
	s.Nil(response.Warnings)
}

func (s *EndpointTestSuite) TestRerunJob_SuccessWithSpecificVersion() {
	ctx := context.Background()
	jobID := uuid.NewString()
	jobVersion := uint64(1)
	requestVersion := uint64(1)
	namespace := "default-namespace"

	job := s.createTestJob(jobID, models.JobStateTypeCompleted)
	job.Version = jobVersion

	specificVersionJob := s.createTestJob(jobID, models.JobStateTypeCompleted)
	specificVersionJob.Version = requestVersion

	request := &RerunJobRequest{
		JobIDOrName: jobID,
		JobVersion:  requestVersion,
		Namespace:   namespace,
		Reason:      "Test rerun specific version",
	}

	// Setup expectations
	s.mockJobStore.EXPECT().BeginTx(ctx).Return(s.mockTxCtx, nil)
	s.mockJobStore.EXPECT().GetJobByIDOrName(s.mockTxCtx, jobID, namespace).Return(job, nil)
	s.mockJobStore.EXPECT().GetJobVersion(s.mockTxCtx, jobID, requestVersion).Return(specificVersionJob, nil)
	s.mockJobStore.EXPECT().UpdateJob(s.mockTxCtx, specificVersionJob).Return(nil)
	s.mockJobStore.EXPECT().UpdateJobState(s.mockTxCtx, gomock.Any()).Return(nil)
	s.mockJobStore.EXPECT().AddJobHistory(s.mockTxCtx, jobID, requestVersion+jobVersionIncrement, gomock.Any()).Return(nil)
	s.mockJobStore.EXPECT().CreateEvaluation(s.mockTxCtx, gomock.Any()).Return(nil)
	s.mockTxCtx.EXPECT().Commit().Return(nil)
	s.mockTxCtx.EXPECT().Rollback().Return(nil)

	response, err := s.endpoint.RerunJob(ctx, request)

	s.NoError(err)
	s.NotNil(response)
	s.Equal(jobID, response.JobID)
	s.Equal(requestVersion+jobVersionIncrement, response.JobVersion)
	s.NotEmpty(response.EvaluationID)
}

func (s *EndpointTestSuite) TestRerunJob_BeginTxFails() {
	ctx := context.Background()
	request := &RerunJobRequest{
		JobIDOrName: "test-job",
		Namespace:   "default",
	}

	expectedErr := fmt.Errorf("transaction failed")
	s.mockJobStore.EXPECT().BeginTx(ctx).Return(nil, expectedErr)

	response, err := s.endpoint.RerunJob(ctx, request)

	s.Error(err)
	s.NotNil(response)
	s.Empty(response.JobID)
}

func (s *EndpointTestSuite) TestRerunJob_GetJobByIDOrNameFails() {
	ctx := context.Background()
	jobID := "non-existent-job"
	namespace := "default"

	request := &RerunJobRequest{
		JobIDOrName: jobID,
		Namespace:   namespace,
	}

	expectedErr := bacerrors.New("job not found").WithCode(bacerrors.NotFoundError)
	s.mockJobStore.EXPECT().BeginTx(ctx).Return(s.mockTxCtx, nil)
	s.mockJobStore.EXPECT().GetJobByIDOrName(s.mockTxCtx, jobID, namespace).Return(models.Job{}, expectedErr)

	response, err := s.endpoint.RerunJob(ctx, request)

	s.Error(err)
	s.Nil(response)
}

func (s *EndpointTestSuite) TestRerunJob_GetJobVersionFails() {
	ctx := context.Background()
	jobID := uuid.NewString()
	requestVersion := uint64(5)
	namespace := "default"

	job := s.createTestJob(jobID, models.JobStateTypeCompleted)
	request := &RerunJobRequest{
		JobIDOrName: jobID,
		JobVersion:  requestVersion,
		Namespace:   namespace,
	}

	expectedErr := bacerrors.New("version not found").WithCode(bacerrors.NotFoundError)
	s.mockJobStore.EXPECT().BeginTx(ctx).Return(s.mockTxCtx, nil)
	s.mockJobStore.EXPECT().GetJobByIDOrName(s.mockTxCtx, jobID, namespace).Return(job, nil)
	s.mockJobStore.EXPECT().GetJobVersion(s.mockTxCtx, jobID, requestVersion).Return(models.Job{}, expectedErr)

	response, err := s.endpoint.RerunJob(ctx, request)

	s.Error(err)
	s.Nil(response)
}

func (s *EndpointTestSuite) TestRerunJob_JobNotRerunnable() {
	ctx := context.Background()
	jobID := uuid.NewString()
	namespace := "default"

	// Create a job in a non-rerunnable state (pending)
	job := s.createTestJob(jobID, models.JobStateTypePending)

	request := &RerunJobRequest{
		JobIDOrName: jobID,
		Namespace:   namespace,
	}

	s.mockJobStore.EXPECT().BeginTx(ctx).Return(s.mockTxCtx, nil)
	s.mockJobStore.EXPECT().GetJobByIDOrName(s.mockTxCtx, jobID, namespace).Return(job, nil)
	s.mockTxCtx.EXPECT().Rollback().Return(nil)

	response, err := s.endpoint.RerunJob(ctx, request)

	s.Error(err)
	s.Nil(response)
	s.Contains(err.Error(), "cannot rerun job in state")
}

func (s *EndpointTestSuite) TestRerunJob_UpdateJobFails() {
	ctx := context.Background()
	jobID := uuid.NewString()
	namespace := "default"

	job := s.createTestJob(jobID, models.JobStateTypeCompleted)
	request := &RerunJobRequest{
		JobIDOrName: jobID,
		Namespace:   namespace,
	}

	expectedErr := fmt.Errorf("update job failed")
	s.mockJobStore.EXPECT().BeginTx(ctx).Return(s.mockTxCtx, nil)
	s.mockJobStore.EXPECT().GetJobByIDOrName(s.mockTxCtx, jobID, namespace).Return(job, nil)
	s.mockJobStore.EXPECT().UpdateJob(s.mockTxCtx, job).Return(expectedErr)
	s.mockTxCtx.EXPECT().Rollback().Return(nil)

	response, err := s.endpoint.RerunJob(ctx, request)

	s.Error(err)
	s.Nil(response)
	s.Equal(expectedErr, err)
}

func (s *EndpointTestSuite) TestRerunJob_UpdateJobStateFails() {
	ctx := context.Background()
	jobID := uuid.NewString()
	namespace := "default"

	job := s.createTestJob(jobID, models.JobStateTypeCompleted)
	request := &RerunJobRequest{
		JobIDOrName: jobID,
		Namespace:   namespace,
	}

	expectedErr := fmt.Errorf("update job state failed")
	s.mockJobStore.EXPECT().BeginTx(ctx).Return(s.mockTxCtx, nil)
	s.mockJobStore.EXPECT().GetJobByIDOrName(s.mockTxCtx, jobID, namespace).Return(job, nil)
	s.mockJobStore.EXPECT().UpdateJob(s.mockTxCtx, job).Return(nil)
	s.mockJobStore.EXPECT().UpdateJobState(s.mockTxCtx, gomock.Any()).Return(expectedErr)
	s.mockTxCtx.EXPECT().Rollback().Return(nil)

	response, err := s.endpoint.RerunJob(ctx, request)

	s.Error(err)
	s.Nil(response)
	s.Equal(expectedErr, err)
}

func (s *EndpointTestSuite) TestRerunJob_AddJobHistoryFails() {
	ctx := context.Background()
	jobID := uuid.NewString()
	namespace := "default"

	job := s.createTestJob(jobID, models.JobStateTypeCompleted)
	request := &RerunJobRequest{
		JobIDOrName: jobID,
		Namespace:   namespace,
	}

	expectedErr := fmt.Errorf("add job history failed")
	s.mockJobStore.EXPECT().BeginTx(ctx).Return(s.mockTxCtx, nil)
	s.mockJobStore.EXPECT().GetJobByIDOrName(s.mockTxCtx, jobID, namespace).Return(job, nil)
	s.mockJobStore.EXPECT().UpdateJob(s.mockTxCtx, job).Return(nil)
	s.mockJobStore.EXPECT().UpdateJobState(s.mockTxCtx, gomock.Any()).Return(nil)
	s.mockJobStore.EXPECT().AddJobHistory(s.mockTxCtx, jobID, job.Version+jobVersionIncrement, gomock.Any()).Return(expectedErr)
	s.mockTxCtx.EXPECT().Rollback().Return(nil)

	response, err := s.endpoint.RerunJob(ctx, request)

	s.Error(err)
	s.Nil(response)
	s.Equal(expectedErr, err)
}

func (s *EndpointTestSuite) TestRerunJob_CreateEvaluationFails() {
	ctx := context.Background()
	jobID := uuid.NewString()
	namespace := "default"

	job := s.createTestJob(jobID, models.JobStateTypeCompleted)
	request := &RerunJobRequest{
		JobIDOrName: jobID,
		Namespace:   namespace,
	}

	expectedErr := fmt.Errorf("create evaluation failed")
	s.mockJobStore.EXPECT().BeginTx(ctx).Return(s.mockTxCtx, nil)
	s.mockJobStore.EXPECT().GetJobByIDOrName(s.mockTxCtx, jobID, namespace).Return(job, nil)
	s.mockJobStore.EXPECT().UpdateJob(s.mockTxCtx, job).Return(nil)
	s.mockJobStore.EXPECT().UpdateJobState(s.mockTxCtx, gomock.Any()).Return(nil)
	s.mockJobStore.EXPECT().AddJobHistory(s.mockTxCtx, jobID, job.Version+jobVersionIncrement, gomock.Any()).Return(nil)
	s.mockJobStore.EXPECT().CreateEvaluation(s.mockTxCtx, gomock.Any()).Return(expectedErr)
	s.mockTxCtx.EXPECT().Rollback().Return(nil)

	response, err := s.endpoint.RerunJob(ctx, request)

	s.Error(err)
	s.NotNil(response)
	s.Empty(response.JobID)
}

func (s *EndpointTestSuite) TestRerunJob_CommitFails() {
	ctx := context.Background()
	jobID := uuid.NewString()
	namespace := "default"

	job := s.createTestJob(jobID, models.JobStateTypeCompleted)
	request := &RerunJobRequest{
		JobIDOrName: jobID,
		Namespace:   namespace,
	}

	expectedErr := fmt.Errorf("commit failed")
	s.mockJobStore.EXPECT().BeginTx(ctx).Return(s.mockTxCtx, nil)
	s.mockJobStore.EXPECT().GetJobByIDOrName(s.mockTxCtx, jobID, namespace).Return(job, nil)
	s.mockJobStore.EXPECT().UpdateJob(s.mockTxCtx, job).Return(nil)
	s.mockJobStore.EXPECT().UpdateJobState(s.mockTxCtx, gomock.Any()).Return(nil)
	s.mockJobStore.EXPECT().AddJobHistory(s.mockTxCtx, jobID, job.Version+jobVersionIncrement, gomock.Any()).Return(nil)
	s.mockJobStore.EXPECT().CreateEvaluation(s.mockTxCtx, gomock.Any()).Return(nil)
	s.mockTxCtx.EXPECT().Commit().Return(expectedErr)
	s.mockTxCtx.EXPECT().Rollback().Return(nil)

	response, err := s.endpoint.RerunJob(ctx, request)

	s.Error(err)
	s.NotNil(response)
	s.Empty(response.JobID)
}

func (s *EndpointTestSuite) TestRerunJob_ValidatesUpdateJobStateRequest() {
	ctx := context.Background()
	jobID := uuid.NewString()
	namespace := "default"

	job := s.createTestJob(jobID, models.JobStateTypeCompleted)
	request := &RerunJobRequest{
		JobIDOrName: jobID,
		Namespace:   namespace,
	}

	// Custom matcher to validate the UpdateJobStateRequest
	updateJobStateMatcher := gomock.Any()

	s.mockJobStore.EXPECT().BeginTx(ctx).Return(s.mockTxCtx, nil)
	s.mockJobStore.EXPECT().GetJobByIDOrName(s.mockTxCtx, jobID, namespace).Return(job, nil)
	s.mockJobStore.EXPECT().UpdateJob(s.mockTxCtx, job).Return(nil)
	s.mockJobStore.EXPECT().UpdateJobState(s.mockTxCtx, updateJobStateMatcher).DoAndReturn(
		func(ctx context.Context, req jobstore.UpdateJobStateRequest) error {
			s.Equal(jobID, req.JobID)
			s.Equal(models.JobStateTypePending, req.NewState)
			s.Equal("job rerun", req.Message)
			s.Contains(req.Condition.UnexpectedStates, models.JobStateTypeQueued)
			s.Len(req.Condition.UnexpectedStates, 1)
			return nil
		},
	)
	s.mockJobStore.EXPECT().AddJobHistory(s.mockTxCtx, jobID, job.Version+jobVersionIncrement, gomock.Any()).Return(nil)
	s.mockJobStore.EXPECT().CreateEvaluation(s.mockTxCtx, gomock.Any()).DoAndReturn(
		func(ctx context.Context, eval models.Evaluation) error {
			s.Equal(jobID, eval.JobID)
			s.Equal(models.EvalTriggerJobRerun, eval.TriggeredBy)
			s.Equal(job.Type, eval.Type)
			s.Equal(models.EvalStatusPending, eval.Status)
			s.NotEmpty(eval.ID)
			s.Greater(eval.CreateTime, int64(0))
			s.Greater(eval.ModifyTime, int64(0))
			return nil
		},
	)
	s.mockTxCtx.EXPECT().Commit().Return(nil)
	s.mockTxCtx.EXPECT().Rollback().Return(nil)

	response, err := s.endpoint.RerunJob(ctx, request)

	s.NoError(err)
	s.NotNil(response)
}

func (s *EndpointTestSuite) TestRerunJob_VerifiesJobVersion() {
	ctx := context.Background()
	jobID := uuid.NewString()
	jobVersion := uint64(3)
	namespace := "default-namespace"

	// Create a job in a rerunnable state (completed)
	job := s.createTestJob(jobID, models.JobStateTypeCompleted)
	job.Version = jobVersion

	request := &RerunJobRequest{
		JobIDOrName: jobID,
		JobVersion:  0, // Use latest version
		Namespace:   namespace,
		Reason:      "Test rerun version",
	}

	// Setup expectations
	s.mockJobStore.EXPECT().BeginTx(ctx).Return(s.mockTxCtx, nil)
	s.mockJobStore.EXPECT().GetJobByIDOrName(s.mockTxCtx, jobID, namespace).Return(job, nil)
	s.mockJobStore.EXPECT().UpdateJob(s.mockTxCtx, job).Return(nil)
	s.mockJobStore.EXPECT().UpdateJobState(s.mockTxCtx, gomock.Any()).Return(nil)
	s.mockJobStore.EXPECT().AddJobHistory(s.mockTxCtx, jobID, jobVersion+jobVersionIncrement, gomock.Any()).Return(nil)
	s.mockJobStore.EXPECT().CreateEvaluation(s.mockTxCtx, gomock.Any()).Return(nil)
	s.mockTxCtx.EXPECT().Commit().Return(nil)
	s.mockTxCtx.EXPECT().Rollback().Return(nil)

	response, err := s.endpoint.RerunJob(ctx, request)

	s.NoError(err)
	s.NotNil(response)
	s.Equal(jobID, response.JobID)
	s.Equal(jobVersion+jobVersionIncrement, response.JobVersion, "JobVersion should be incremented by jobVersionIncrement")
	s.NotEmpty(response.EvaluationID)
	s.Nil(response.Warnings)
}

func (s *EndpointTestSuite) TestRerunJob_VerifiesJobVersionWithSpecificVersion() {
	ctx := context.Background()
	jobID := uuid.NewString()
	latestVersion := uint64(5)
	requestedVersion := uint64(2)
	namespace := "default-namespace"

	// Create a job in a rerunnable state (completed) with latest version
	latestJob := s.createTestJob(jobID, models.JobStateTypeCompleted)
	latestJob.Version = latestVersion

	// Create a specific version of the job to rerun
	specificVersionJob := s.createTestJob(jobID, models.JobStateTypeCompleted)
	specificVersionJob.Version = requestedVersion

	request := &RerunJobRequest{
		JobIDOrName: jobID,
		JobVersion:  requestedVersion,
		Namespace:   namespace,
		Reason:      "Test rerun specific version",
	}

	// Setup expectations
	s.mockJobStore.EXPECT().BeginTx(ctx).Return(s.mockTxCtx, nil)
	s.mockJobStore.EXPECT().GetJobByIDOrName(s.mockTxCtx, jobID, namespace).Return(latestJob, nil)
	s.mockJobStore.EXPECT().GetJobVersion(s.mockTxCtx, jobID, requestedVersion).Return(specificVersionJob, nil)
	s.mockJobStore.EXPECT().UpdateJob(s.mockTxCtx, specificVersionJob).Return(nil)
	s.mockJobStore.EXPECT().UpdateJobState(s.mockTxCtx, gomock.Any()).Return(nil)
	s.mockJobStore.EXPECT().AddJobHistory(s.mockTxCtx, jobID, latestVersion+jobVersionIncrement, gomock.Any()).Return(nil)
	s.mockJobStore.EXPECT().CreateEvaluation(s.mockTxCtx, gomock.Any()).Return(nil)
	s.mockTxCtx.EXPECT().Commit().Return(nil)
	s.mockTxCtx.EXPECT().Rollback().Return(nil)

	response, err := s.endpoint.RerunJob(ctx, request)

	s.NoError(err)
	s.NotNil(response)
	s.Equal(jobID, response.JobID)
	s.Equal(latestVersion+jobVersionIncrement, response.JobVersion, "JobVersion should be based on latest version plus increment")
	s.NotEmpty(response.EvaluationID)
	s.Nil(response.Warnings)
}

// createTestJob creates a test job with the specified ID and state
func (s *EndpointTestSuite) createTestJob(jobID string, state models.JobStateType) models.Job {
	job := mock.Job()
	job.ID = jobID
	job.Version = 1
	job.State = models.NewJobState(state).WithMessage("Test job")
	job.CreateTime = time.Now().UnixNano()
	job.ModifyTime = time.Now().UnixNano()
	return *job
}

// SubmitJob Tests

func (s *EndpointTestSuite) TestSubmitJob_Success_NewJob() {
	ctx := context.Background()
	job := s.createTestJobForSubmission("test-job", "default")

	request := &SubmitJobRequest{
		Job:                  job,
		ClientInstallationID: "test-install-id",
		ClientInstanceID:     "test-instance-id",
		Force:                false,
	}

	// Setup expectations - job doesn't exist, so it's a new job
	s.mockJobStore.EXPECT().GetJobByName(ctx, job.Name, job.Namespace).Return(models.Job{}, bacerrors.New("not found").WithCode(bacerrors.NotFoundError))
	s.mockJobStore.EXPECT().BeginTx(ctx).Return(s.mockTxCtx, nil)
	s.mockJobStore.EXPECT().CreateJob(s.mockTxCtx, gomock.Any()).Return(nil)
	s.mockJobStore.EXPECT().AddJobHistory(s.mockTxCtx, gomock.Any(), uint64(initialJobVersion), gomock.Any()).Return(nil)
	s.mockJobStore.EXPECT().CreateEvaluation(s.mockTxCtx, gomock.Any()).DoAndReturn(
		func(ctx context.Context, eval models.Evaluation) error {
			// Verify the evaluation has an ID
			s.NotEmpty(eval.ID, "Evaluation ID should not be empty when passed to CreateEvaluation")
			s.Equal(models.EvalTriggerJobRegister, eval.TriggeredBy)
			s.Equal(models.EvalStatusPending, eval.Status)
			return nil
		},
	)
	s.mockTxCtx.EXPECT().Commit().Return(nil)
	s.mockTxCtx.EXPECT().Rollback().Return(nil)

	response, err := s.endpoint.SubmitJob(ctx, request)

	s.NoError(err)
	s.NotNil(response)
	s.NotEmpty(response.EvaluationID)
	s.True(s.mockJobTransformer.TransformCalled)
	s.Equal("test-install-id", job.Meta[models.MetaClientInstallationID])
	s.Equal("test-instance-id", job.Meta[models.MetaClientInstanceID])
}

func (s *EndpointTestSuite) TestSubmitJob_Success_UpdateExistingJob() {
	ctx := context.Background()
	job := s.createTestJobForSubmission("existing-job", "default")
	existingJob := s.createTestJob(job.ID, models.JobStateTypeCompleted)
	existingJob.Name = job.Name
	existingJob.Namespace = job.Namespace
	existingJob.Version = 2

	// Make jobs different to bypass the "no changes" check
	job.Meta["new-key"] = "new-value"

	request := &SubmitJobRequest{
		Job:   job,
		Force: false,
	}

	// Setup expectations - job exists, so it's an update
	s.mockJobStore.EXPECT().GetJobByName(ctx, job.Name, job.Namespace).Return(existingJob, nil)
	s.mockJobStore.EXPECT().BeginTx(ctx).Return(s.mockTxCtx, nil)
	s.mockJobStore.EXPECT().UpdateJob(s.mockTxCtx, gomock.Any()).Return(nil)
	s.mockJobStore.EXPECT().UpdateJobState(s.mockTxCtx, gomock.Any()).Return(nil)
	s.mockJobStore.EXPECT().AddJobHistory(s.mockTxCtx, gomock.Any(), existingJob.Version+uint64(jobVersionIncrement), gomock.Any()).Return(nil)
	s.mockJobStore.EXPECT().CreateEvaluation(s.mockTxCtx, gomock.Any()).DoAndReturn(
		func(ctx context.Context, eval models.Evaluation) error {
			s.Equal(models.EvalTriggerJobUpdate, eval.TriggeredBy)
			s.Equal(job.Type, eval.Type)
			s.Equal(models.EvalStatusPending, eval.Status)
			return nil
		},
	)
	s.mockTxCtx.EXPECT().Commit().Return(nil)
	s.mockTxCtx.EXPECT().Rollback().Return(nil)

	response, err := s.endpoint.SubmitJob(ctx, request)

	s.NoError(err)
	s.NotNil(response)
	s.NotEmpty(response.JobID)
	s.NotEmpty(response.EvaluationID)
}

func (s *EndpointTestSuite) TestSubmitJob_Success_ForceUpdateWithNoChanges() {
	ctx := context.Background()
	job := s.createTestJobForSubmission("force-job", "default")
	existingJob := s.createTestJob(job.ID, models.JobStateTypeCompleted)
	existingJob.Name = job.Name
	existingJob.Namespace = job.Namespace
	existingJob.Version = 1

	request := &SubmitJobRequest{
		Job:   job,
		Force: true, // Force should bypass "no changes" check
	}

	s.mockJobStore.EXPECT().GetJobByName(ctx, job.Name, job.Namespace).Return(existingJob, nil)
	s.mockJobStore.EXPECT().BeginTx(ctx).Return(s.mockTxCtx, nil)
	s.mockJobStore.EXPECT().UpdateJob(s.mockTxCtx, gomock.Any()).Return(nil)
	s.mockJobStore.EXPECT().UpdateJobState(s.mockTxCtx, gomock.Any()).Return(nil)
	s.mockJobStore.EXPECT().AddJobHistory(s.mockTxCtx, gomock.Any(), existingJob.Version+uint64(jobVersionIncrement), gomock.Any()).Return(nil)
	s.mockJobStore.EXPECT().CreateEvaluation(s.mockTxCtx, gomock.Any()).Return(nil)
	s.mockTxCtx.EXPECT().Commit().Return(nil)
	s.mockTxCtx.EXPECT().Rollback().Return(nil)

	response, err := s.endpoint.SubmitJob(ctx, request)

	s.NoError(err)
	s.NotNil(response)
	s.NotEmpty(response.JobID)
}

func (s *EndpointTestSuite) TestSubmitJob_Error_NoChangesDetected() {
	ctx := context.Background()
	job := s.createTestJobForSubmission("no-changes-job", "default")
	existingJob := s.createTestJob(job.ID, models.JobStateTypeCompleted)
	existingJob.Name = job.Name
	existingJob.Namespace = job.Namespace
	existingJob.Version = 1
	// Make jobs identical so CompareWith returns empty string

	request := &SubmitJobRequest{
		Job:   job,
		Force: false,
	}

	s.mockJobStore.EXPECT().GetJobByName(ctx, job.Name, job.Namespace).Return(existingJob, nil)

	response, err := s.endpoint.SubmitJob(ctx, request)

	s.Error(err)
	s.Nil(response)
	s.Contains(err.Error(), "no changes detected")
}

func (s *EndpointTestSuite) TestSubmitJob_Error_JobTransformerFails() {
	ctx := context.Background()
	job := s.createTestJobForSubmission("transform-fail-job", "default")

	request := &SubmitJobRequest{
		Job: job,
	}

	// Set the transformer to return an error
	s.mockJobTransformer.TransformError = fmt.Errorf("transformation failed")

	// No GetJobByName expectation because transformer fails early and method returns

	response, err := s.endpoint.SubmitJob(ctx, request)

	s.Error(err)
	s.Nil(response)
	s.Equal("transformation failed", err.Error())
}

func (s *EndpointTestSuite) TestSubmitJob_Error_GetJobByNameFails() {
	ctx := context.Background()
	job := s.createTestJobForSubmission("get-fail-job", "default")

	request := &SubmitJobRequest{
		Job: job,
	}

	expectedErr := fmt.Errorf("database error")
	s.mockJobStore.EXPECT().GetJobByName(ctx, job.Name, job.Namespace).Return(models.Job{}, expectedErr)

	response, err := s.endpoint.SubmitJob(ctx, request)

	s.Error(err)
	s.Nil(response)
	s.Equal(expectedErr, err)
}

func (s *EndpointTestSuite) TestSubmitJob_Error_BeginTxFails() {
	ctx := context.Background()
	job := s.createTestJobForSubmission("tx-fail-job", "default")

	request := &SubmitJobRequest{
		Job: job,
	}

	expectedErr := fmt.Errorf("transaction failed")
	s.mockJobStore.EXPECT().GetJobByName(ctx, job.Name, job.Namespace).Return(models.Job{}, bacerrors.New("not found").WithCode(bacerrors.NotFoundError))
	s.mockJobStore.EXPECT().BeginTx(ctx).Return(nil, expectedErr)

	response, err := s.endpoint.SubmitJob(ctx, request)

	s.Error(err)
	s.Nil(response)
	s.Contains(err.Error(), "failed to begin transaction")
}

func (s *EndpointTestSuite) TestSubmitJob_Error_CreateJobFails() {
	ctx := context.Background()
	job := s.createTestJobForSubmission("create-fail-job", "default")

	request := &SubmitJobRequest{
		Job: job,
	}

	expectedErr := fmt.Errorf("create job failed")
	s.mockJobStore.EXPECT().GetJobByName(ctx, job.Name, job.Namespace).Return(models.Job{}, bacerrors.New("not found").WithCode(bacerrors.NotFoundError))
	s.mockJobStore.EXPECT().BeginTx(ctx).Return(s.mockTxCtx, nil)
	s.mockJobStore.EXPECT().CreateJob(s.mockTxCtx, gomock.Any()).Return(expectedErr)
	s.mockTxCtx.EXPECT().Rollback().Return(nil)

	response, err := s.endpoint.SubmitJob(ctx, request)

	s.Error(err)
	s.Nil(response)
	s.Equal(expectedErr, err)
}

func (s *EndpointTestSuite) TestSubmitJob_Error_UpdateJobFails() {
	ctx := context.Background()
	job := s.createTestJobForSubmission("update-fail-job", "default")
	existingJob := s.createTestJob(job.ID, models.JobStateTypeCompleted)
	existingJob.Name = job.Name
	existingJob.Namespace = job.Namespace
	job.Meta["diff-key"] = "diff-value" // Make jobs different

	request := &SubmitJobRequest{
		Job: job,
	}

	s.mockJobStore.EXPECT().GetJobByName(ctx, job.Name, job.Namespace).Return(existingJob, nil)
	s.mockJobStore.EXPECT().BeginTx(ctx).Return(s.mockTxCtx, nil)
	s.mockJobStore.EXPECT().UpdateJob(s.mockTxCtx, gomock.Any()).Return(fmt.Errorf("update job failed"))
	s.mockTxCtx.EXPECT().Rollback().Return(nil)

	response, err := s.endpoint.SubmitJob(ctx, request)

	s.Error(err)
	s.Nil(response)
	s.Equal(fmt.Errorf("update job failed"), err)
}

func (s *EndpointTestSuite) TestSubmitJob_Error_UpdateJobStateFails() {
	ctx := context.Background()
	job := s.createTestJobForSubmission("update-state-fail-job", "default")
	existingJob := s.createTestJob(job.ID, models.JobStateTypeCompleted)
	existingJob.Name = job.Name
	existingJob.Namespace = job.Namespace
	job.Meta["diff-key"] = "diff-value"

	request := &SubmitJobRequest{
		Job: job,
	}

	s.mockJobStore.EXPECT().GetJobByName(ctx, job.Name, job.Namespace).Return(existingJob, nil)
	s.mockJobStore.EXPECT().BeginTx(ctx).Return(s.mockTxCtx, nil)
	s.mockJobStore.EXPECT().UpdateJob(s.mockTxCtx, gomock.Any()).Return(nil)
	s.mockJobStore.EXPECT().UpdateJobState(s.mockTxCtx, gomock.Any()).Return(fmt.Errorf("update job state failed"))
	s.mockTxCtx.EXPECT().Rollback().Return(nil)

	response, err := s.endpoint.SubmitJob(ctx, request)

	s.Error(err)
	s.Nil(response)
	s.Equal(fmt.Errorf("update job state failed"), err)
}

func (s *EndpointTestSuite) TestSubmitJob_Error_AddJobHistoryFails_NewJob() {
	ctx := context.Background()
	job := s.createTestJobForSubmission("history-fail-new-job", "default")

	request := &SubmitJobRequest{
		Job: job,
	}

	expectedErr := fmt.Errorf("add job history failed")
	s.mockJobStore.EXPECT().GetJobByName(ctx, job.Name, job.Namespace).Return(models.Job{}, bacerrors.New("not found").WithCode(bacerrors.NotFoundError))
	s.mockJobStore.EXPECT().BeginTx(ctx).Return(s.mockTxCtx, nil)
	s.mockJobStore.EXPECT().CreateJob(s.mockTxCtx, gomock.Any()).Return(nil)
	s.mockJobStore.EXPECT().AddJobHistory(s.mockTxCtx, gomock.Any(), uint64(initialJobVersion), gomock.Any()).Return(expectedErr)
	s.mockTxCtx.EXPECT().Rollback().Return(nil)

	response, err := s.endpoint.SubmitJob(ctx, request)

	s.Error(err)
	s.Nil(response)
	s.Equal(expectedErr, err)
}

func (s *EndpointTestSuite) TestSubmitJob_Error_AddJobHistoryFails_UpdateJob() {
	ctx := context.Background()
	job := s.createTestJobForSubmission("history-fail-update-job", "default")
	existingJob := s.createTestJob(job.ID, models.JobStateTypeCompleted)
	existingJob.Name = job.Name
	existingJob.Namespace = job.Namespace
	existingJob.Version = 3
	job.Meta["diff-key"] = "diff-value"

	request := &SubmitJobRequest{
		Job: job,
	}

	s.mockJobStore.EXPECT().GetJobByName(ctx, job.Name, job.Namespace).Return(existingJob, nil)
	s.mockJobStore.EXPECT().BeginTx(ctx).Return(s.mockTxCtx, nil)
	s.mockJobStore.EXPECT().UpdateJob(s.mockTxCtx, gomock.Any()).Return(nil)
	s.mockJobStore.EXPECT().UpdateJobState(s.mockTxCtx, gomock.Any()).Return(nil)
	s.mockJobStore.EXPECT().AddJobHistory(s.mockTxCtx, gomock.Any(), existingJob.Version+uint64(jobVersionIncrement), gomock.Any()).Return(fmt.Errorf("add job history failed"))
	s.mockTxCtx.EXPECT().Rollback().Return(nil)

	response, err := s.endpoint.SubmitJob(ctx, request)

	s.Error(err)
	s.Nil(response)
	s.Equal(fmt.Errorf("add job history failed"), err)
}

func (s *EndpointTestSuite) TestSubmitJob_Error_CreateEvaluationFails() {
	ctx := context.Background()
	job := s.createTestJobForSubmission("eval-fail-job", "default")

	request := &SubmitJobRequest{
		Job: job,
	}

	expectedErr := fmt.Errorf("create evaluation failed")
	s.mockJobStore.EXPECT().GetJobByName(ctx, job.Name, job.Namespace).Return(models.Job{}, bacerrors.New("not found").WithCode(bacerrors.NotFoundError))
	s.mockJobStore.EXPECT().BeginTx(ctx).Return(s.mockTxCtx, nil)
	s.mockJobStore.EXPECT().CreateJob(s.mockTxCtx, gomock.Any()).Return(nil)
	s.mockJobStore.EXPECT().AddJobHistory(s.mockTxCtx, gomock.Any(), uint64(initialJobVersion), gomock.Any()).Return(nil)
	s.mockJobStore.EXPECT().CreateEvaluation(s.mockTxCtx, gomock.Any()).Return(expectedErr)
	s.mockTxCtx.EXPECT().Rollback().Return(nil)

	response, err := s.endpoint.SubmitJob(ctx, request)

	s.Error(err)
	s.Nil(response)
	s.Equal(expectedErr, err)
}

func (s *EndpointTestSuite) TestSubmitJob_Error_CommitFails() {
	ctx := context.Background()
	job := s.createTestJobForSubmission("commit-fail-job", "default")

	request := &SubmitJobRequest{
		Job: job,
	}

	expectedErr := fmt.Errorf("commit failed")
	s.mockJobStore.EXPECT().GetJobByName(ctx, job.Name, job.Namespace).Return(models.Job{}, bacerrors.New("not found").WithCode(bacerrors.NotFoundError))
	s.mockJobStore.EXPECT().BeginTx(ctx).Return(s.mockTxCtx, nil)
	s.mockJobStore.EXPECT().CreateJob(s.mockTxCtx, gomock.Any()).Return(nil)
	s.mockJobStore.EXPECT().AddJobHistory(s.mockTxCtx, gomock.Any(), uint64(initialJobVersion), gomock.Any()).Return(nil)
	s.mockJobStore.EXPECT().CreateEvaluation(s.mockTxCtx, gomock.Any()).Return(nil)
	s.mockTxCtx.EXPECT().Commit().Return(expectedErr)
	s.mockTxCtx.EXPECT().Rollback().Return(nil)

	response, err := s.endpoint.SubmitJob(ctx, request)

	s.Error(err)
	s.Nil(response)
	s.Equal(expectedErr, err)
}

func (s *EndpointTestSuite) TestSubmitJob_ValidatesUpdateJobStateRequestForUpdate() {
	ctx := context.Background()
	job := s.createTestJobForSubmission("validate-update-job", "default")
	existingJob := s.createTestJob(job.ID, models.JobStateTypeCompleted)
	existingJob.Name = job.Name
	existingJob.Namespace = job.Namespace
	existingJob.Version = 2
	job.Meta["diff-key"] = "diff-value"

	request := &SubmitJobRequest{
		Job: job,
	}

	s.mockJobStore.EXPECT().GetJobByName(ctx, job.Name, job.Namespace).Return(existingJob, nil)
	s.mockJobStore.EXPECT().BeginTx(ctx).Return(s.mockTxCtx, nil)
	s.mockJobStore.EXPECT().UpdateJob(s.mockTxCtx, gomock.Any()).Return(nil)
	s.mockJobStore.EXPECT().UpdateJobState(s.mockTxCtx, gomock.Any()).DoAndReturn(
		func(ctx context.Context, req jobstore.UpdateJobStateRequest) error {
			s.Equal(models.JobStateTypePending, req.NewState)
			s.Equal("Job update requested by user", req.Message)
			return nil
		},
	)
	s.mockJobStore.EXPECT().AddJobHistory(s.mockTxCtx, gomock.Any(), existingJob.Version+uint64(jobVersionIncrement), gomock.Any()).Return(nil)
	s.mockJobStore.EXPECT().CreateEvaluation(s.mockTxCtx, gomock.Any()).DoAndReturn(
		func(ctx context.Context, eval models.Evaluation) error {
			s.Equal(models.EvalTriggerJobUpdate, eval.TriggeredBy)
			s.Equal(job.Type, eval.Type)
			s.Equal(models.EvalStatusPending, eval.Status)
			s.NotEmpty(eval.ID)
			s.Greater(eval.CreateTime, int64(0))
			s.Greater(eval.ModifyTime, int64(0))
			return nil
		},
	)
	s.mockTxCtx.EXPECT().Commit().Return(nil)
	s.mockTxCtx.EXPECT().Rollback().Return(nil)

	response, err := s.endpoint.SubmitJob(ctx, request)

	s.NoError(err)
	s.NotNil(response)
}

func (s *EndpointTestSuite) TestSubmitJob_ValidatesEvaluationForNewJob() {
	ctx := context.Background()
	job := s.createTestJobForSubmission("validate-new-job", "default")

	request := &SubmitJobRequest{
		Job: job,
	}

	s.mockJobStore.EXPECT().GetJobByName(ctx, job.Name, job.Namespace).Return(models.Job{}, bacerrors.New("not found").WithCode(bacerrors.NotFoundError))
	s.mockJobStore.EXPECT().BeginTx(ctx).Return(s.mockTxCtx, nil)
	s.mockJobStore.EXPECT().CreateJob(s.mockTxCtx, gomock.Any()).Return(nil)
	s.mockJobStore.EXPECT().AddJobHistory(s.mockTxCtx, gomock.Any(), uint64(initialJobVersion), gomock.Any()).Return(nil)
	s.mockJobStore.EXPECT().CreateEvaluation(s.mockTxCtx, gomock.Any()).DoAndReturn(
		func(ctx context.Context, eval models.Evaluation) error {
			s.Equal(models.EvalTriggerJobRegister, eval.TriggeredBy)
			s.Equal(job.Type, eval.Type)
			s.Equal(models.EvalStatusPending, eval.Status)
			s.NotEmpty(eval.ID)
			s.Greater(eval.CreateTime, int64(0))
			s.Greater(eval.ModifyTime, int64(0))
			return nil
		},
	)
	s.mockTxCtx.EXPECT().Commit().Return(nil)
	s.mockTxCtx.EXPECT().Rollback().Return(nil)

	response, err := s.endpoint.SubmitJob(ctx, request)

	s.NoError(err)
	s.NotNil(response)
}

// createTestJobForSubmission creates a test job specifically for submission tests
func (s *EndpointTestSuite) createTestJobForSubmission(name, namespace string) *models.Job {
	job := mock.Job()
	job.Name = name
	job.Namespace = namespace
	job.ID = uuid.NewString()
	job.Type = models.JobTypeBatch
	job.Meta = make(map[string]string)
	job.CreateTime = time.Now().UnixNano()
	job.ModifyTime = time.Now().UnixNano()
	return job
}

func TestEndpointTestSuite(t *testing.T) {
	suite.Run(t, new(EndpointTestSuite))
}
