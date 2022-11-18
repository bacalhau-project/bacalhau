//go:build unit || !integration

package combo

import (
	"testing"
)

func TestFallbackPublisher(t *testing.T) {
	runTestCases(t, map[string]comboTestCase{
		"empty":   {NewFallbackPublisher(), mockPublisher{}},
		"single":  {NewFallbackPublisher(&healthyPublisher), healthyPublisher},
		"healthy": {NewFallbackPublisher(&errorPublisher, &healthyPublisher), healthyPublisher},
		"error":   {NewFallbackPublisher(&errorPublisher, &errorPublisher), errorPublisher},
	})
}
