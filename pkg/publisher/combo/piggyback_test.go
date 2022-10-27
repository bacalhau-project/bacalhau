package combo

import (
	"context"
	"errors"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publisher"
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

func TestPiggybackedPublisher_PublishShardResult(t *testing.T) {
	for _, test := range []struct {
		name        string
		primary     []interface{}
		piggyback   []interface{}
		expected    model.StorageSpec
		expectedErr error
	}{
		{
			name:        "all_successful",
			primary:     []interface{}{model.StorageSpec{Name: "primary"}, nil},
			piggyback:   []interface{}{model.StorageSpec{Name: "piggy"}, nil},
			expected:    model.StorageSpec{Name: "primary"},
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

			shard := model.JobShard{Index: 4}
			hostId := "host"
			resultsPath := "/some/path"

			primary.On("PublishShardResult", mock.Anything, shard, hostId, resultsPath).Return(test.primary...)
			piggyback.On("PublishShardResult", mock.Anything, shard, hostId, resultsPath).Return(test.piggyback...)

			actual, err := subject.PublishShardResult(context.Background(), shard, hostId, resultsPath)
			assert.Equal(t, test.expectedErr, err)
			assert.Equal(t, test.expected, actual)
		})
	}
}

func TestPiggybackedPublisher_ComposeResultReferences(t *testing.T) {
	for _, test := range []struct {
		name        string
		primary     []interface{}
		piggyback   []interface{}
		expected    []model.StorageSpec
		expectedErr error
	}{
		{
			name:        "all_successful",
			primary:     []interface{}{[]model.StorageSpec{{Name: "primary"}}, nil},
			piggyback:   []interface{}{[]model.StorageSpec{{Name: "piggy"}}, nil},
			expected:    []model.StorageSpec{{Name: "primary"}},
			expectedErr: nil,
		},
		{
			name:        "primary_error",
			primary:     []interface{}{[]model.StorageSpec(nil), errors.New("failed")},
			piggyback:   []interface{}{[]model.StorageSpec{{Name: "piggy"}}, nil},
			expected:    nil,
			expectedErr: errors.New("failed"),
		},
		{
			name:        "piggyback_error",
			primary:     []interface{}{[]model.StorageSpec{{Name: "primary"}}, nil},
			piggyback:   []interface{}{[]model.StorageSpec(nil), errors.New("failed")},
			expected:    nil,
			expectedErr: errors.New("failed"),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			primary := new(testifyPublisher)
			piggyback := new(testifyPublisher)

			subject := NewPiggybackedPublisher(primary, piggyback)

			jobId := "42"

			primary.On("ComposeResultReferences", mock.Anything, jobId).Return(test.primary...)
			piggyback.On("ComposeResultReferences", mock.Anything, jobId).Return(test.piggyback...)

			actual, err := subject.ComposeResultReferences(context.Background(), jobId)
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

func (t *testifyPublisher) PublishShardResult(ctx context.Context, shard model.JobShard, hostID string, shardResultPath string) (model.StorageSpec, error) {
	args := t.Called(ctx, shard, hostID, shardResultPath)
	return args.Get(0).(model.StorageSpec), args.Error(1)
}

func (t *testifyPublisher) ComposeResultReferences(ctx context.Context, jobID string) ([]model.StorageSpec, error) {
	args := t.Called(ctx, jobID)
	return args.Get(0).([]model.StorageSpec), args.Error(1)
}

var _ publisher.Publisher = &testifyPublisher{}
