package traces

import (
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/muesli/clusters"
	"github.com/muesli/kmeans"
	"github.com/rs/zerolog/log"
	"gonum.org/v1/gonum/stat"
)

// tools for comparing traces

type Trace struct {
	ResultId string
	Filename string
}

type TraceCollection struct {
	Traces []Trace

	Tolerance float64

	// internal
	// resultId -> list of samples -> column -> value
	data      map[string][]map[string]float64
	columns   []string
	waypoints []float64 // timestamp waypoints

	// resultId -> column -> waypoint -> closest value to that waypoint
	valuesPerWaypoint map[string]map[string]map[float64]float64

	// column -> waypoints -> average for that waypoint across all the samples
	averages map[string]map[float64]float64
}

func (t *TraceCollection) parseFiles() error {
	t.data = make(map[string][]map[string]float64)
	for _, trace := range t.Traces {
		bs, err := os.ReadFile(trace.Filename)
		if err != nil {
			return err
		}
		lines := strings.Split(string(bs), "\n")
		// ignore header line
		for _, line := range lines[1:] {
			cells := strings.Fields(line)
			if len(cells) == 4 {
				time, err := strconv.ParseFloat(cells[0], 64)
				if err != nil {
					return err
				}
				cpu, err := strconv.ParseFloat(cells[1], 64)
				if err != nil {
					return err
				}
				real, err := strconv.ParseFloat(cells[2], 64)
				if err != nil {
					return err
				}
				virtual, err := strconv.ParseFloat(cells[3], 64)
				if err != nil {
					return err
				}
				_, ok := t.data[trace.ResultId]
				if !ok {
					t.data[trace.ResultId] = make([]map[string]float64, 0)
				}
				t.data[trace.ResultId] = append(t.data[trace.ResultId], map[string]float64{
					"time":    time,
					"cpu":     cpu,
					"real":    real,
					"virtual": virtual,
				})
			} else {
				if os.Getenv("DEBUG") != "" {
					log.Printf("Unexpected line '%s', ignoring", line)
				}
			}
		}
	}
	return nil
}

var NUM_WAYPOINTS = 10

func (t *TraceCollection) calculateWaypoints() error {
	maxTimestamp := 0.0
	for _, jobData := range t.data {
		for _, sample := range jobData {
			if sample["time"] > maxTimestamp {
				maxTimestamp = sample["time"]
			}
		}
	}

	// + 1 so that the last waypoint is before the very end of the data
	interval := maxTimestamp / float64(NUM_WAYPOINTS+1)
	for i := 0; i < NUM_WAYPOINTS; i++ {
		t.waypoints = append(t.waypoints, float64(i)*interval)
	}

	return nil
}

func (t *TraceCollection) calculateValuesPerWaypoint() error {
	// resultId -> list of samples -> column -> value
	// map[string][]map[string]float64
	t.valuesPerWaypoint = make(map[string]map[string]map[float64]float64)
	for resultId := range t.data {
		for _, column := range t.columns {
			t.calculateValuesPerWaypointForResultIdAndColumn(resultId, column)
		}
	}
	return nil
}

func (t *TraceCollection) calculateValuesPerWaypointForResultIdAndColumn(resultId, column string) {
	log.Debug().Msgf("Doing %s\n", resultId)
	jobData := t.data[resultId]
	currentWaypointIdx := 0
	maxWaypointIdx := NUM_WAYPOINTS - 1

	for _, sample := range jobData {
		timestamp := sample["time"]
		thisValue := sample[column]
		if timestamp > t.waypoints[currentWaypointIdx] {
			// resultId -> column -> waypoint -> closest value to that waypoint
			// map[string]map[string]map[float64]float64
			_, ok := t.valuesPerWaypoint[resultId]
			if !ok {
				t.valuesPerWaypoint[resultId] = make(map[string]map[float64]float64)
			}
			_, ok = t.valuesPerWaypoint[resultId][column]
			if !ok {
				t.valuesPerWaypoint[resultId][column] = make(map[float64]float64)
			}
			t.valuesPerWaypoint[resultId][column][t.waypoints[currentWaypointIdx]] = thisValue
			currentWaypointIdx += 1
			if currentWaypointIdx > maxWaypointIdx {
				// our work here is done
				return
			}
		}
	}

}

func (t *TraceCollection) calcAvgs(col string) {
	// for a given column, calculate the averages per waypoint
	for _, waypoint := range t.waypoints {
		values := []float64{}
		for resultId := range t.valuesPerWaypoint {
			values = append(values, t.valuesPerWaypoint[resultId][col][waypoint])
		}
		average := stat.Mean(values, nil)
		_, ok := t.averages[col]
		if !ok {
			t.averages[col] = make(map[float64]float64)
		}
		t.averages[col][waypoint] = average
	}
}

func (t *TraceCollection) distance(resultId, column string) float64 {
	distances := []float64{}
	for _, w := range t.waypoints {
		value := t.averages[column][w] - t.valuesPerWaypoint[resultId][column][w]
		distances = append(distances, value)
	}
	return stat.Mean(distances, nil)
}

func (t *TraceCollection) Scores() (map[string]map[string]float64, error) {
	// map resultId -> column -> score (average distance from average for that column)
	res := map[string]map[string]float64{}

	t.columns = []string{"cpu", "real", "virtual"}

	err := t.parseFiles()
	if err != nil {
		return nil, err
	}

	err = t.calculateWaypoints()
	if err != nil {
		return nil, err
	}

	log.Debug().Msgf("WAYPOINTS --> %+v\n", t.waypoints)

	err = t.calculateValuesPerWaypoint()
	if err != nil {
		return nil, err
	}
	log.Debug().Msgf("VALUES PER WAYPOINT --> %+v\n", t.valuesPerWaypoint)

	t.averages = make(map[string]map[float64]float64)
	for _, col := range t.columns {
		t.calcAvgs(col)
	}

	log.Debug().Msgf("AVERAGE VALUES PER WAYPOINT --> %+v\n", t.averages)

	for resultId := range t.data {
		for _, col := range t.columns {
			_, ok := res[resultId]
			if !ok {
				res[resultId] = make(map[string]float64)
			}
			res[resultId][col] = t.distance(resultId, col)
		}
	}
	return res, nil

	// TODO Maybe return a single value? Or just use memory for now

}

// naively (for now) cluster the results into two buckets: "right" and "wrong"
func (t *TraceCollection) Cluster() ([]string, []string, error) {

	scores, _ := t.Scores()

	if len(scores) == 0 {
		err := fmt.Errorf("Could not run clustering, no scores attached to traces.")
		log.Error().Err(err)
		return nil, nil, err
	}

	var resultsToResultIdMap = make(map[string][]string)

	var d clusters.Observations

	for resultId, score := range scores {
		r := fmt.Sprintf("%f", score["real"])
		_, ok := resultsToResultIdMap[r]
		if !ok {
			resultsToResultIdMap[r] = []string{}
		}

		resultsToResultIdMap[r] = append(resultsToResultIdMap[r], resultId)

		d = append(d, clusters.Coordinates{
			score["real"],
		})
	}
	log.Debug().Msgf("resultsToResultIdMap = %+v\n", resultsToResultIdMap)

	// map[string]map[string]float64
	// map resultId -> column -> score (average distance from average for that column)

	km := kmeans.New()
	clusters, _ := km.Partition(d, 2)

	// TODO: Print the means here

	if math.Abs(float64(clusters[0].Center[0])-float64(clusters[1].Center[0])) < t.Tolerance {

		allIds := []string{}
		emptyIds := []string{}

		for resultId := range scores {
			allIds = append(allIds, resultId)
		}
		sort.Strings(allIds)
		return allIds, emptyIds, nil
	}

	for _, c := range clusters {
		log.Debug().Msgf("Centered at x: %.2f\n", c.Center[0])
		log.Debug().Msgf("Matching data points: %+v\n\n", c.Observations)
	}

	// reconstitute the results from the clusters...
	l := make([][]string, 2)

	for i := 0; i < 2; i++ {
		deduped := map[string]bool{}
		for _, obs := range clusters[i].Observations {
			o := fmt.Sprintf("%f", obs.Coordinates()[0])
			ids := resultsToResultIdMap[o]
			log.Debug().Msgf("Picked ids for %s: %+v\n", o, ids)
			for _, id := range ids {
				deduped[id] = true
			}
		}
		for k := range deduped {
			l[i] = append(l[i], k)
		}
		sort.Strings(l[i])
	}

	if len(l[0]) == len(l[1]) {
		// a tie, OH NO, NO MAJORITY - HUNG PARLIAMENT 😱
		return nil, nil, fmt.Errorf("no majority, please set concurrency to an odd number")
	} else if len(l[0]) > len(l[1]) {
		// 0 is winner
		return l[0], l[1], nil
	} else { // len(l[0]) < len(l[1])
		// 1 is winner
		return l[1], l[0], nil
	}

}
