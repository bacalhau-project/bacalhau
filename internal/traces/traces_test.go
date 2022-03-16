package traces

import (
	"fmt"
	"reflect"
	"strconv"
	"testing"

	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"

	_ "github.com/filecoin-project/bacalhau/internal/logger"
)

func truncateFloat(f float64) float64 {
	ret, _ := strconv.ParseFloat(fmt.Sprintf("%.32f", f), 32)
	return ret
}

// truncate the floats down in precision to
// avoid non-determinism in floating point arithmetic
func processResults(data map[string]map[string]float64) map[string]map[string]float64 {
	for _, sample := range data {
		sample["real"] = truncateFloat(sample["real"])
	}
	return data
}

func TestFixtureMetrics(t *testing.T) {
	clustered := TraceCollection{Traces: []Trace{
		{ResultId: "job-1", Filename: "fixtures/metrics-1.log"},
		{ResultId: "job-2", Filename: "fixtures/metrics-2.log"},
		{ResultId: "job-3", Filename: "fixtures/metrics-3.log"},
	}}
	scores, err := clustered.Scores()
	if err != nil {
		log.Debug().Msgf("Error getting scores: %s\n", err)
		panic(err)
	}
	log.Debug().Msgf("Scores: %+v\n", scores)

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

	assert.True(t, reflect.DeepEqual(processResults(scores), processResults(shouldEqual)))
}
