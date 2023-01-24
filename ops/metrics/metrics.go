package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	bacalhau_model_v1beta1 "github.com/filecoin-project/bacalhau/pkg/model"
	bacalhau_model_v1alpha1 "github.com/filecoin-project/bacalhau/pkg/model/v1alpha1"
)

// Based on code at:
//
//	github.com/filecoin-project/bacalhau/dashboard/api/cmd/dashboard/import.go
type LogLineAlpha struct {
	Type  string
	Event bacalhau_model_v1alpha1.JobEvent
}

type LogLineBeta struct {
	Type  string
	Event bacalhau_model_v1beta1.JobEvent
}

func parseEvent(text string) (*bacalhau_model_v1beta1.JobEvent, error) {
	var event *bacalhau_model_v1beta1.JobEvent
	if strings.Contains(text, `"APIVersion":"V1beta1"`) {
		var line LogLineBeta
		err := json.Unmarshal([]byte(text), &line)
		if err != nil {
			return nil, err
		}
		if line.Type != "model.JobEvent" {
			return nil, fmt.Errorf("expected JobEvent, got %s", line.Type)
		}
		event = &line.Event
	} else {
		var line LogLineAlpha
		err := json.Unmarshal([]byte(text), &line)
		if err != nil {
			return nil, err
		}
		if line.Type != "model.JobEvent" {
			return nil, fmt.Errorf("expected JobEvent, got %s", line.Type)
		}
		converted := bacalhau_model_v1beta1.ConvertV1alpha1JobEvent(line.Event)
		event = &converted
	}
	return event, nil
}

func isJobCreatedEvent(event *bacalhau_model_v1beta1.JobEvent) bool {
	return event.EventName == bacalhau_model_v1beta1.JobEventCreated
}

func isCanaryEvent(event *bacalhau_model_v1beta1.JobEvent) bool {
	cmd := strings.Join(event.Spec.Docker.Entrypoint, " ")
	return strings.Contains(cmd, "λ")
}

func importEvents(filename string) ([]*bacalhau_model_v1beta1.JobEvent, error) {
	events := []*bacalhau_model_v1beta1.JobEvent{}
	if filename == "" {
		return events, fmt.Errorf("please specify a filename")
	}
	_, err := os.Stat(filename)
	if errors.Is(err, os.ErrNotExist) {
		return events, fmt.Errorf("filename does not exist: %s", filename)
	}
	file, err := os.Open(filename)
	if err != nil {
		return events, err
	}
	defer file.Close()

	total := 0
	parsed := 0
	notCanary := 0
	jobCreated := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		total++
		event, err := parseEvent(scanner.Text())
		if err != nil {
			fmt.Printf("Error parsing: %s", err.Error())
			continue
		}
		parsed++
		if isCanaryEvent(event) {
			continue
		}
		notCanary++
		if !isJobCreatedEvent(event) {
			continue
		}
		jobCreated++
		//if counter > 1000 {
		//	return events, nil
		//}
		events = append(events, event)
	}
	fmt.Printf("Events: total=%d, parsed=%d, notCanary=%d, jobCreated=%d\n",
		total, parsed, notCanary, jobCreated)
	if err := scanner.Err(); err != nil {
		return events, err
	}
	return events, nil
}

func writeCSVFile(filename string, header []string, rows [][]string) {
	csvFile, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	defer csvFile.Close()

	csvWriter := csv.NewWriter(csvFile)
	if err := csvWriter.Write(header); err != nil {
		fmt.Printf("Failed to write CSV header: %s\n", err.Error())
	}
	if err := csvWriter.WriteAll(rows); err != nil {
		fmt.Printf("Failed to write CSV rows: %s\n", err.Error())
	}
}

type KeyCount struct {
	key   string
	count int
}

func writeCSVMap(filename string, key string, m map[string]int) {
	hs := []string{key, "count"}
	kcs := []KeyCount{}
	for k, v := range m {
		kcs = append(kcs, KeyCount{k, v})
	}
	sort.Slice(kcs, func(i, j int) bool {
		return kcs[i].key < kcs[j].key
	})
	rs := [][]string{}
	for _, kc := range kcs {
		rs = append(rs, []string{kc.key, strconv.Itoa(kc.count)})
	}
	writeCSVFile(filename, hs, rs)
}

func writeCSVStats(events []*bacalhau_model_v1beta1.JobEvent) {
	clients := map[string]struct{}{}
	jobsPerDay := map[string]int{}
	jobsByType := map[string]int{}
	jobsByGPU := map[string]int{}
	jobsByImage := map[string]int{}
	for _, event := range events {
		clients[event.ClientID] = struct{}{}
		jobsPerDay[event.EventTime.Format("2006-01-02")]++
		jobsByType[event.Spec.Engine.String()]++
		jobsByGPU[event.Spec.Resources.GPU]++
		jobsByImage[event.Spec.Docker.Image]++
	}
	jobCount := len(events)
	clientCount := len(clients)

	fmt.Printf("Job count: %d\n", jobCount)
	writeCSVMap("jobs_total.csv", "jobs",
		map[string]int{"all": jobCount})

	fmt.Printf("Client count: %d\n", clientCount)
	writeCSVMap("clients_total.csv", "clients",
		map[string]int{"all": jobCount})

	fmt.Printf("Jobs per day: %v\n", jobsPerDay)
	writeCSVMap("jobs_per_day.csv", "date", jobsPerDay)

	fmt.Printf("Jobs by type: %v\n", jobsByType)
	writeCSVMap("jobs_by_type.csv", "type", jobsByType)

	fmt.Printf("Jobs by GPU: %v\n", jobsByGPU)
	writeCSVMap("jobs_by_gpu.csv", "GPUs", jobsByGPU)

	fmt.Printf("Jobs by image: %v\n", jobsByImage)
	writeCSVMap("jobs_by_image.csv", "docker_image", jobsByImage)
}

func main() {
	if len(os.Args) != 2 {
		fmt.Printf("usage: %s eventfile\n", os.Args[0])
		return
	}
	events, err := importEvents(os.Args[1])
	if err != nil {
		fmt.Println(err.Error())
	}
	writeCSVStats(events)
}
