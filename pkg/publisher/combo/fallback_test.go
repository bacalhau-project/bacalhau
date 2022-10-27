package combo

import (
	"context"
	"fmt"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publisher"
	"github.com/stretchr/testify/require"
)

type mockPublisher struct {
	isInstalled                bool
	isInstalledErr             error
	composeResultReferences    []model.StorageSpec
	composeResultReferencesErr error
	publishShardResult         model.StorageSpec
	publishShardResultErr      error
}

// ComposeResultReferences implements publisher.Publisher
func (m *mockPublisher) ComposeResultReferences(context.Context, string) ([]model.StorageSpec, error) {
	return m.composeResultReferences, m.composeResultReferencesErr
}

// IsInstalled implements publisher.Publisher
func (m *mockPublisher) IsInstalled(context.Context) (bool, error) {
	return m.isInstalled, m.isInstalledErr
}

// PublishShardResult implements publisher.Publisher
func (m *mockPublisher) PublishShardResult(context.Context, model.JobShard, string, string) (model.StorageSpec, error) {
	return m.publishShardResult, m.publishShardResultErr
}

var _ publisher.Publisher = (*mockPublisher)(nil)

var healthyPublisher = mockPublisher{
	isInstalled:             true,
	composeResultReferences: []model.StorageSpec{{Name: "test spec"}},
	publishShardResult:      model.StorageSpec{Name: "test output"},
}

var errorPublisher = mockPublisher{
	isInstalledErr:             fmt.Errorf("test error"),
	composeResultReferencesErr: fmt.Errorf("test error"),
	publishShardResultErr:      fmt.Errorf("test error"),
}

func TestFallbackPublisher(t *testing.T) {
	var testCases = map[string]struct {
		publisher       publisher.Publisher
		expectPublisher mockPublisher
	}{
		"empty":   {NewFallbackPublisher(), mockPublisher{}},
		"single":  {NewFallbackPublisher(&healthyPublisher), healthyPublisher},
		"healthy": {NewFallbackPublisher(&errorPublisher, &healthyPublisher), healthyPublisher},
		"error":   {NewFallbackPublisher(&errorPublisher, &errorPublisher), errorPublisher},
	}

	for name, testCase := range testCases {
		t.Run(name+"/IsInstalled", func(t *testing.T) {
			result, err := testCase.publisher.IsInstalled(context.Background())
			require.Equal(t, testCase.expectPublisher.isInstalledErr == nil, err == nil)
			require.Equal(t, testCase.expectPublisher.isInstalled, result)
		})
		t.Run(name+"/ComposeResultReferences", func(t *testing.T) {
			result, err := testCase.publisher.ComposeResultReferences(context.Background(), "")
			require.Equal(t, testCase.expectPublisher.composeResultReferencesErr == nil, err == nil)
			require.Equal(t, testCase.expectPublisher.composeResultReferences, result)
		})
		t.Run(name+"/PublishShardResult", func(t *testing.T) {
			result, err := testCase.publisher.PublishShardResult(context.Background(), model.JobShard{}, "", "")
			require.Equal(t, testCase.expectPublisher.publishShardResultErr == nil, err == nil)
			require.Equal(t, testCase.expectPublisher.publishShardResult, result)
		})
	}
}
