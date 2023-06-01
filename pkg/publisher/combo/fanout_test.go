//go:build unit || !integration

package combo

import (
	"fmt"
	"testing"
	"time"

	storagetesting "github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/testing"
)

func sleepyPublisher(t testing.TB) mockPublisher {
	return mockPublisher{
		isInstalled:     true,
		ValidateJobErr:  nil,
		PublishedResult: storagetesting.MakeIpfsStorageSpec(t, "TODO", "TODO", storagetesting.TestCID1.String()),
		sleepTime:       50 * time.Millisecond,
	}
}

var uninstalledPublisher = mockPublisher{
	isInstalled:        false,
	ValidateJobErr:     fmt.Errorf("invalid publisher spec"),
	PublishedResultErr: fmt.Errorf("not installed"),
	sleepTime:          0,
}

func TestFanoutPublisher(t *testing.T) {
	sleepPub := sleepyPublisher(t)
	runTestCases(t, map[string]comboTestCase{
		"single publisher":                 {NewFanoutPublisher(&healthyPublisher), healthyPublisher},
		"takes first value":                {NewFanoutPublisher(&healthyPublisher, &sleepPub), healthyPublisher},
		"waits for installed":              {NewFanoutPublisher(&uninstalledPublisher, &sleepPub), sleepPub},
		"noone is installed":               {NewFanoutPublisher(&uninstalledPublisher), uninstalledPublisher},
		"waits for good value":             {NewFanoutPublisher(&errorPublisher, &sleepPub), sleepPub},
		"returns error for all":            {NewFanoutPublisher(&errorPublisher, &errorPublisher), errorPublisher},
		"waits for highest priority value": {NewPrioritizedFanoutPublisher(time.Millisecond*100, &sleepPub, &healthyPublisher), sleepPub},
		"only waits for max time":          {NewPrioritizedFanoutPublisher(time.Millisecond*20, &sleepPub, &healthyPublisher), healthyPublisher},
		"waits for unprioritized value":    {NewPrioritizedFanoutPublisher(time.Millisecond*100, &errorPublisher, &sleepPub), sleepPub},
	})
}
