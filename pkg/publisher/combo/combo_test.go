package combo

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/publisher"
	"github.com/stretchr/testify/require"
)

type mockPublisher struct {
	isInstalled           bool
	isInstalledErr        error
	publishShardResult    model.StorageSpec
	publishShardResultErr error
	sleepTime             time.Duration
}

// IsInstalled implements publisher.Publisher
func (m *mockPublisher) IsInstalled(context.Context) (bool, error) {
	time.Sleep(m.sleepTime)
	return m.isInstalled, m.isInstalledErr
}

// PublishShardResult implements publisher.Publisher
func (m *mockPublisher) PublishShardResult(context.Context, model.JobShard, string, string) (model.StorageSpec, error) {
	time.Sleep(m.sleepTime)
	return m.publishShardResult, m.publishShardResultErr
}

var _ publisher.Publisher = (*mockPublisher)(nil)

var healthyPublisher = mockPublisher{
	isInstalled:        true,
	publishShardResult: model.StorageSpec{Name: "test output"},
}

var errorPublisher = mockPublisher{
	isInstalledErr:        fmt.Errorf("test error"),
	publishShardResultErr: fmt.Errorf("test error"),
}

type comboTestCase struct {
	publisher       publisher.Publisher
	expectPublisher mockPublisher
}

func runTestCase(t *testing.T, name string, testCase comboTestCase) {
	t.Run(name+"/IsInstalled", func(t *testing.T) {
		result, err := testCase.publisher.IsInstalled(context.Background())
		require.Equal(t, testCase.expectPublisher.isInstalledErr == nil, err == nil, err)
		require.Equal(t, testCase.expectPublisher.isInstalled, result)
	})
	t.Run(name+"/PublishShardResult", func(t *testing.T) {
		result, err := testCase.publisher.PublishShardResult(context.Background(), model.JobShard{}, "", "")
		require.Equal(t, testCase.expectPublisher.publishShardResultErr == nil, err == nil, err)
		require.Equal(t, testCase.expectPublisher.publishShardResult, result)
	})
}

func runTestCases(t *testing.T, testCases map[string]comboTestCase) {
	for name, testCase := range testCases {
		runTestCase(t, name, testCase)
	}
}
