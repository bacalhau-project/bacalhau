//go:build unit || !integration

package combo

import (
	"context"
	"errors"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/publisher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestPiggybackedPublisher_IsInstalled(t *testing.T) {
	for _, test := range []struct {
		name        string
		primary     []interface{}
		piggyback   []interface{}
		expected    bool
		expectedErr error
	}{
		{
			name:        "all_successful",
			primary:     []interface{}{true, nil},
			piggyback:   []interface{}{true, nil},
			expected:    true,
			expectedErr: nil,
		},
		{
			name:        "primary_error",
			primary:     []interface{}{false, errors.New("failed")},
			piggyback:   []interface{}{true, nil},
			expected:    false,
			expectedErr: errors.New("failed"),
		},
		{
			name:        "piggyback_error",
			primary:     []interface{}{true, nil},
			piggyback:   []interface{}{true, errors.New("failed")},
			expected:    false,
			expectedErr: errors.New("failed"),
		},
		{
			name:        "primary_not_installed",
			primary:     []interface{}{false, nil},
			piggyback:   []interface{}{true, nil},
			expected:    false,
			expectedErr: nil,
		},
		{
			name:        "piggyback_not_installed",
			primary:     []interface{}{true, nil},
			piggyback:   []interface{}{false, nil},
			expected:    false,
			expectedErr: nil,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			primary := new(testifyPublisher)
			piggyback := new(testifyPublisher)

			subject := NewPiggybackedPublisher(primary, piggyback)

			primary.On("IsInstalled", mock.Anything).Return(test.primary...)
			piggyback.On("IsInstalled", mock.Anything).Return(test.piggyback...)

			actual, err := subject.IsInstalled(context.Background())
			assert.Equal(t, test.expectedErr, err)
			assert.Equal(t, test.expected, actual)
		})
	}
}

func TestPiggybackedPublisher_PublishResult(t *testing.T) {
	for _, test := range []struct {
		name        string
		primary     []interface{}
		piggyback   []interface{}
		expected    model.StorageSpec
		expectedErr error
	}{
		{
			name:        "all_successful",
			primary:     []interface{}{model.StorageSpec{Name: "primary", StorageSource: model.StorageSourceIPFS, CID: "123"}, nil},
			piggyback:   []interface{}{model.StorageSpec{Name: "piggy", StorageSource: model.StorageSourceFilecoin, CID: "456"}, nil},
			expected:    model.StorageSpec{Name: "primary", StorageSource: model.StorageSourceIPFS, CID: "123", Metadata: map[string]string{"Filecoin": "456"}},
			expectedErr: nil,
		},
		{
			name:        "primary_error",
			primary:     []interface{}{model.StorageSpec{}, errors.New("failed")},
			piggyback:   []interface{}{model.StorageSpec{Name: "piggy"}, nil},
			expected:    model.StorageSpec{},
			expectedErr: errors.New("failed"),
		},
		{
			name:        "piggyback_error",
			primary:     []interface{}{model.StorageSpec{Name: "primary"}, nil},
			piggyback:   []interface{}{model.StorageSpec{}, errors.New("failed")},
			expected:    model.StorageSpec{},
			expectedErr: errors.New("failed"),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			primary := new(testifyPublisher)
			piggyback := new(testifyPublisher)

			subject := NewPiggybackedPublisher(primary, piggyback)

			job := model.Job{}
			executionID := "1234"
			resultsPath := "/some/path"

			primary.On("PublishResult", mock.Anything, executionID, job, resultsPath).Return(test.primary...)
			piggyback.On("PublishResult", mock.Anything, executionID, job, resultsPath).Return(test.piggyback...)

			actual, err := subject.PublishResult(context.Background(), executionID, job, resultsPath)
			assert.Equal(t, test.expectedErr, err)
			assert.Equal(t, test.expected, actual)
		})
	}
}

type testifyPublisher struct {
	mock.Mock
}

func (t *testifyPublisher) IsInstalled(ctx context.Context) (bool, error) {
	args := t.Called(ctx)
	return args.Bool(0), args.Error(1)
}

func (t *testifyPublisher) PublishResult(ctx context.Context, executionID string, job model.Job, resultPath string) (model.StorageSpec, error) {
	args := t.Called(ctx, executionID, job, resultPath)
	return args.Get(0).(model.StorageSpec), args.Error(1)
}

func (t *testifyPublisher) ComposeResultReferences(ctx context.Context, jobID string) ([]model.StorageSpec, error) {
	args := t.Called(ctx, jobID)
	return args.Get(0).([]model.StorageSpec), args.Error(1)
}

var _ publisher.Publisher = &testifyPublisher{}
