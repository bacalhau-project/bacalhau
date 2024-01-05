// Code generated by MockGen. DO NOT EDIT.
// Source: types.go
//
// Generated by this command:
//
//	mockgen --source types.go --destination mocks.go --package jobstore
//

// Package jobstore is a generated GoMock package.
package jobstore

import (
	context "context"
	reflect "reflect"

	models "github.com/bacalhau-project/bacalhau/pkg/models"
	gomock "go.uber.org/mock/gomock"
)

// MockStore is a mock of Store interface.
type MockStore struct {
	ctrl     *gomock.Controller
	recorder *MockStoreMockRecorder
}

// MockStoreMockRecorder is the mock recorder for MockStore.
type MockStoreMockRecorder struct {
	mock *MockStore
}

// NewMockStore creates a new mock instance.
func NewMockStore(ctrl *gomock.Controller) *MockStore {
	mock := &MockStore{ctrl: ctrl}
	mock.recorder = &MockStoreMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockStore) EXPECT() *MockStoreMockRecorder {
	return m.recorder
}

// Close mocks base method.
func (m *MockStore) Close(ctx context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Close", ctx)
	ret0, _ := ret[0].(error)
	return ret0
}

// Close indicates an expected call of Close.
func (mr *MockStoreMockRecorder) Close(ctx any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Close", reflect.TypeOf((*MockStore)(nil).Close), ctx)
}

// CreateEvaluation mocks base method.
func (m *MockStore) CreateEvaluation(ctx context.Context, eval models.Evaluation) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateEvaluation", ctx, eval)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreateEvaluation indicates an expected call of CreateEvaluation.
func (mr *MockStoreMockRecorder) CreateEvaluation(ctx, eval any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateEvaluation", reflect.TypeOf((*MockStore)(nil).CreateEvaluation), ctx, eval)
}

// CreateExecution mocks base method.
func (m *MockStore) CreateExecution(ctx context.Context, execution models.Execution) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateExecution", ctx, execution)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreateExecution indicates an expected call of CreateExecution.
func (mr *MockStoreMockRecorder) CreateExecution(ctx, execution any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateExecution", reflect.TypeOf((*MockStore)(nil).CreateExecution), ctx, execution)
}

// CreateJob mocks base method.
func (m *MockStore) CreateJob(ctx context.Context, j models.Job) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateJob", ctx, j)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreateJob indicates an expected call of CreateJob.
func (mr *MockStoreMockRecorder) CreateJob(ctx, j any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateJob", reflect.TypeOf((*MockStore)(nil).CreateJob), ctx, j)
}

// DeleteEvaluation mocks base method.
func (m *MockStore) DeleteEvaluation(ctx context.Context, id string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteEvaluation", ctx, id)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteEvaluation indicates an expected call of DeleteEvaluation.
func (mr *MockStoreMockRecorder) DeleteEvaluation(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteEvaluation", reflect.TypeOf((*MockStore)(nil).DeleteEvaluation), ctx, id)
}

// DeleteJob mocks base method.
func (m *MockStore) DeleteJob(ctx context.Context, jobID string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteJob", ctx, jobID)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteJob indicates an expected call of DeleteJob.
func (mr *MockStoreMockRecorder) DeleteJob(ctx, jobID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteJob", reflect.TypeOf((*MockStore)(nil).DeleteJob), ctx, jobID)
}

// GetEvaluation mocks base method.
func (m *MockStore) GetEvaluation(ctx context.Context, id string) (models.Evaluation, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetEvaluation", ctx, id)
	ret0, _ := ret[0].(models.Evaluation)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetEvaluation indicates an expected call of GetEvaluation.
func (mr *MockStoreMockRecorder) GetEvaluation(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetEvaluation", reflect.TypeOf((*MockStore)(nil).GetEvaluation), ctx, id)
}

// GetExecutions mocks base method.
func (m *MockStore) GetExecutions(ctx context.Context, jobID string) ([]models.Execution, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetExecutions", ctx, jobID)
	ret0, _ := ret[0].([]models.Execution)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetExecutions indicates an expected call of GetExecutions.
func (mr *MockStoreMockRecorder) GetExecutions(ctx, jobID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetExecutions", reflect.TypeOf((*MockStore)(nil).GetExecutions), ctx, jobID)
}

// GetInProgressJobs mocks base method.
func (m *MockStore) GetInProgressJobs(ctx context.Context) ([]models.Job, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetInProgressJobs", ctx)
	ret0, _ := ret[0].([]models.Job)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetInProgressJobs indicates an expected call of GetInProgressJobs.
func (mr *MockStoreMockRecorder) GetInProgressJobs(ctx any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetInProgressJobs", reflect.TypeOf((*MockStore)(nil).GetInProgressJobs), ctx)
}

// GetJob mocks base method.
func (m *MockStore) GetJob(ctx context.Context, id string) (models.Job, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetJob", ctx, id)
	ret0, _ := ret[0].(models.Job)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetJob indicates an expected call of GetJob.
func (mr *MockStoreMockRecorder) GetJob(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetJob", reflect.TypeOf((*MockStore)(nil).GetJob), ctx, id)
}

// GetJobHistory mocks base method.
func (m *MockStore) GetJobHistory(ctx context.Context, jobID string, options JobHistoryFilterOptions) ([]models.JobHistory, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetJobHistory", ctx, jobID, options)
	ret0, _ := ret[0].([]models.JobHistory)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetJobHistory indicates an expected call of GetJobHistory.
func (mr *MockStoreMockRecorder) GetJobHistory(ctx, jobID, options any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetJobHistory", reflect.TypeOf((*MockStore)(nil).GetJobHistory), ctx, jobID, options)
}

// GetJobs mocks base method.
func (m *MockStore) GetJobs(ctx context.Context, query JobQuery) ([]models.Job, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetJobs", ctx, query)
	ret0, _ := ret[0].([]models.Job)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetJobs indicates an expected call of GetJobs.
func (mr *MockStoreMockRecorder) GetJobs(ctx, query any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetJobs", reflect.TypeOf((*MockStore)(nil).GetJobs), ctx, query)
}

// UpdateExecution mocks base method.
func (m *MockStore) UpdateExecution(ctx context.Context, request UpdateExecutionRequest) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateExecution", ctx, request)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateExecution indicates an expected call of UpdateExecution.
func (mr *MockStoreMockRecorder) UpdateExecution(ctx, request any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateExecution", reflect.TypeOf((*MockStore)(nil).UpdateExecution), ctx, request)
}

// UpdateJobState mocks base method.
func (m *MockStore) UpdateJobState(ctx context.Context, request UpdateJobStateRequest) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateJobState", ctx, request)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateJobState indicates an expected call of UpdateJobState.
func (mr *MockStoreMockRecorder) UpdateJobState(ctx, request any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateJobState", reflect.TypeOf((*MockStore)(nil).UpdateJobState), ctx, request)
}

// Watch mocks base method.
func (m *MockStore) Watch(ctx context.Context, types StoreWatcherType, events StoreEventType) chan WatchEvent {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Watch", ctx, types, events)
	ret0, _ := ret[0].(chan WatchEvent)
	return ret0
}

// Watch indicates an expected call of Watch.
func (mr *MockStoreMockRecorder) Watch(ctx, types, events any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Watch", reflect.TypeOf((*MockStore)(nil).Watch), ctx, types, events)
}
