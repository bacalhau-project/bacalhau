//go:build unit || !integration

package fanout

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/publisher"
)

func TestFanoutPublisher(t *testing.T) {
	runTestCases(t, map[string]fanoutTestCase{
		"single publisher": {
			NewFanoutPublisher([]publisher.Publisher{&healthyPublisher}),
			healthyPublisher,
		},
		"takes first value": {
			NewFanoutPublisher([]publisher.Publisher{&healthyPublisher, &sleepyPublisher}),
			healthyPublisher,
		},
		"waits for installed": {
			NewFanoutPublisher([]publisher.Publisher{&uninstalledPublisher, &sleepyPublisher}),
			sleepyPublisher,
		},
		"noone is installed": {
			NewFanoutPublisher([]publisher.Publisher{&uninstalledPublisher}),
			uninstalledPublisher,
		},
		"waits for good value": {
			NewFanoutPublisher([]publisher.Publisher{&errorPublisher, &sleepyPublisher}),
			sleepyPublisher,
		},
		"returns error for all": {
			NewFanoutPublisher([]publisher.Publisher{&errorPublisher, &errorPublisher}),
			errorPublisher,
		},
		"waits for highest priority value": {
			NewFanoutPublisher([]publisher.Publisher{&sleepyPublisher, &healthyPublisher}, WithTimeout(time.Millisecond*100), WithPrioritization()),
			sleepyPublisher,
		},
		"only waits for max time": {
			NewFanoutPublisher([]publisher.Publisher{&sleepyPublisher, &healthyPublisher}, WithTimeout(time.Millisecond*20), WithPrioritization()),
			healthyPublisher,
		},
		"waits for unprioritized value": {
			NewFanoutPublisher([]publisher.Publisher{&errorPublisher, &sleepyPublisher}, WithTimeout(time.Millisecond*100), WithPrioritization()),
			sleepyPublisher,
		},
	})
}

type FakePublisher struct {
	isInstalled        bool
	isInstalledErr     error
	ValidateJobErr     error
	PublishedResult    models.SpecConfig
	PublishedResultErr error
	// TODO(forrest): use a mockable clock to avoid test flakes
	sleepTime time.Duration
}

// IsInstalled implements publisher.Publisher
func (m *FakePublisher) IsInstalled(context.Context) (bool, error) {
	time.Sleep(m.sleepTime)
	return m.isInstalled, m.isInstalledErr
}

// ValidateJob implements publisher.Publisher
func (m *FakePublisher) ValidateJob(context.Context, models.Job) error {
	time.Sleep(m.sleepTime)
	return m.ValidateJobErr
}

// PublishResult implements publisher.Publisher
func (m *FakePublisher) PublishResult(context.Context, *models.Execution, string) (models.SpecConfig, error) {
	time.Sleep(m.sleepTime)
	return m.PublishedResult, m.PublishedResultErr
}

var _ publisher.Publisher = (*FakePublisher)(nil)

type fanoutTestCase struct {
	publisher       publisher.Publisher
	expectPublisher FakePublisher
}

var sleepyPublisher = FakePublisher{
	isInstalled:    true,
	ValidateJobErr: nil,
	PublishedResult: models.SpecConfig{
		Type: models.StorageSourceIPFS,
		Params: map[string]interface{}{
			"CID": "123",
		},
	},
	sleepTime: 50 * time.Millisecond,
}

var uninstalledPublisher = FakePublisher{
	isInstalled:        false,
	ValidateJobErr:     fmt.Errorf("invalid publisher spec"),
	PublishedResultErr: fmt.Errorf("not installed"),
	sleepTime:          0,
}

var healthyPublisher = FakePublisher{
	isInstalled: true,
	PublishedResult: models.SpecConfig{
		Type: models.StorageSourceIPFS,
		Params: map[string]interface{}{
			"CID": "123",
		},
	},
}

var errorPublisher = FakePublisher{
	isInstalledErr:     fmt.Errorf("test error"),
	PublishedResultErr: fmt.Errorf("test error"),
}

func runTestCase(t *testing.T, name string, testCase fanoutTestCase) {
	t.Run(name+"/IsInstalled", func(t *testing.T) {
		result, err := testCase.publisher.IsInstalled(context.Background())
		require.Equal(t, testCase.expectPublisher.isInstalledErr == nil, err == nil, err)
		require.Equal(t, testCase.expectPublisher.isInstalled, result)
	})
	t.Run(name+"/PublishResult", func(t *testing.T) {
		result, err := testCase.publisher.PublishResult(context.Background(), &models.Execution{}, "")
		require.Equal(t, testCase.expectPublisher.PublishedResultErr == nil, err == nil, err)
		require.Equal(t, testCase.expectPublisher.PublishedResult, result)
	})
}

func runTestCases(t *testing.T, testCases map[string]fanoutTestCase) {
	for name, testCase := range testCases {
		runTestCase(t, name, testCase)
	}
}
