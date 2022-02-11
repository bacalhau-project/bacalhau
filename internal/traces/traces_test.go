package traces

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFixtureMetrics(t *testing.T) {
	clustered := TraceCollection{Traces: []Trace{
		{ResultId: "job-1", Filename: "fixtures/metrics-1.log"},
		{ResultId: "job-2", Filename: "fixtures/metrics-2.log"},
		{ResultId: "job-3", Filename: "fixtures/metrics-3.log"},
	}}
	scores, err := clustered.Scores()
	if err != nil {
		panic(err)
	}
	if os.Getenv("DEBUG") != "" {
		fmt.Printf("Scores: %+v\n", scores)
	}

	shouldEqual := map[string]map[string]float64{
		"job-1": {
			"cpu":     0,
			"real":    0.02733333333333343,
			"virtual": 0,
		},
		"job-2": {
			"cpu":     0,
			"real":    0.035733333333333527,
			"virtual": 0,
		},
		"job-3": {
			"cpu":     0,
			"real":    -0.06306666666666665,
			"virtual": 0,
		},
	}

	assert.True(t, reflect.DeepEqual(scores, shouldEqual))
}
