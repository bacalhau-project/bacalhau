//go:build unit || !integration

package combo

import (
	"fmt"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/model"
)

var sleepyPublisher = mockPublisher{
	isInstalled:        true,
	publishShardResult: model.StorageSpec{CID: "123"},
	sleepTime:          50 * time.Millisecond,
}

var uninstalledPublisher = mockPublisher{
	isInstalled:           false,
	publishShardResultErr: fmt.Errorf("not installed"),
	sleepTime:             0,
}

func TestFanoutPublisher(t *testing.T) {
	runTestCases(t, map[string]comboTestCase{
		"single publisher":                 {NewFanoutPublisher(&healthyPublisher), healthyPublisher},
		"takes first value":                {NewFanoutPublisher(&healthyPublisher, &sleepyPublisher), healthyPublisher},
		"waits for installed":              {NewFanoutPublisher(&uninstalledPublisher, &sleepyPublisher), sleepyPublisher},
		"noone is installed":               {NewFanoutPublisher(&uninstalledPublisher), uninstalledPublisher},
		"waits for good value":             {NewFanoutPublisher(&errorPublisher, &sleepyPublisher), sleepyPublisher},
		"returns error for all":            {NewFanoutPublisher(&errorPublisher, &errorPublisher), errorPublisher},
		"waits for highest priority value": {NewPrioritizedFanoutPublisher(time.Millisecond*100, &sleepyPublisher, &healthyPublisher), sleepyPublisher},
		"only waits for max time":          {NewPrioritizedFanoutPublisher(time.Millisecond*20, &sleepyPublisher, &healthyPublisher), healthyPublisher},
		"waits for unprioritized value":    {NewPrioritizedFanoutPublisher(time.Millisecond*100, &errorPublisher, &sleepyPublisher), sleepyPublisher},
	})
}
