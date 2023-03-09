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
	isInstalled        bool
	isInstalledErr     error
	PublishedResult    model.StorageSpec
	PublishedResultErr error
	sleepTime          time.Duration
}

// IsInstalled implements publisher.Publisher
func (m *mockPublisher) IsInstalled(context.Context) (bool, error) {
	time.Sleep(m.sleepTime)
	return m.isInstalled, m.isInstalledErr
}

// PublishResult implements publisher.Publisher
func (m *mockPublisher) PublishResult(context.Context, model.Job, string, string) (model.StorageSpec, error) {
	time.Sleep(m.sleepTime)
	return m.PublishedResult, m.PublishedResultErr
}

var _ publisher.Publisher = (*mockPublisher)(nil)

var healthyPublisher = mockPublisher{
	isInstalled:     true,
	PublishedResult: model.StorageSpec{Name: "test output"},
}

var errorPublisher = mockPublisher{
	isInstalledErr:     fmt.Errorf("test error"),
	PublishedResultErr: fmt.Errorf("test error"),
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
	t.Run(name+"/PublishResult", func(t *testing.T) {
		result, err := testCase.publisher.PublishResult(context.Background(), model.Job{}, "", "")
		require.Equal(t, testCase.expectPublisher.PublishedResultErr == nil, err == nil, err)
		require.Equal(t, testCase.expectPublisher.PublishedResult, result)
	})
}

func runTestCases(t *testing.T, testCases map[string]comboTestCase) {
	for name, testCase := range testCases {
		runTestCase(t, name, testCase)
	}
}
