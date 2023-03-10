package inlocalstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
)

type StoreProxy struct {
	store store.ExecutionStore
}

type JobStats struct {
	JobsCompleted uint
}

// Check if the json file exists, and create it if it doesn't.
// olgibbons add an error to this function. Or find an existing function
func EnsureJobStatsJSONExists() (string, error) {
	configDir := ".bacalhau"
	home, _ := os.UserHomeDir()
	jsonFilepath := filepath.Join(home, configDir, "jobStats.json")
	if _, err := os.Stat(jsonFilepath); errors.Is(err, os.ErrNotExist) {
		log.Println(err, ": Creating jobStats.json...")
		f, err := os.Create(jsonFilepath)
		if err != nil {
			return "", fmt.Errorf("Failed to create JSON file %s: %w", jsonFilepath, err)
		}
		emptyJobStat := JobStats{JobsCompleted: 0}
		bs, err := json.Marshal(emptyJobStat)
		if err != nil {
			return "", err
		}
		_, err = f.Write(bs)
		if err != nil {
			return "", err
		}
	}
	return jsonFilepath, nil
}

// don't know if I need this yet olgibbons
func NewStoreProxy(store store.ExecutionStore) *StoreProxy {
	return &StoreProxy{
		store: store,
	}
}

// CreateExecution implements store.ExecutionStore
func (proxy *StoreProxy) CreateExecution(ctx context.Context, execution store.Execution) error {
	return proxy.store.CreateExecution(ctx, execution)
}

// DeleteExecution implements store.ExecutionStore
func (proxy *StoreProxy) DeleteExecution(ctx context.Context, id string) error {
	return proxy.store.DeleteExecution(ctx, id)
}

// GetExecution implements store.ExecutionStore
func (proxy *StoreProxy) GetExecution(ctx context.Context, id string) (store.Execution, error) {
	return proxy.store.GetExecution(ctx, id)
}

// GetExecutionCount implements store.ExecutionStore
func (proxy *StoreProxy) GetExecutionCount(ctx context.Context) uint {
	//read counter from file (olgibbons)
	jsonFilepath := EnsureJobStatsJSONExists()
	jsonbs, err := os.ReadFile(jsonFilepath)
	var jobStore JobStats
	if err != nil {
		log.Fatal(err)
	}
	json.Unmarshal(jsonbs, &jobStore)
	return jobStore.JobsCompleted
}

// GetExecutionHistory implements store.ExecutionStore
func (proxy *StoreProxy) GetExecutionHistory(ctx context.Context, id string) ([]store.ExecutionHistory, error) {
	return proxy.store.GetExecutionHistory(ctx, id)
}

// GetExecutions implements store.ExecutionStore
func (proxy *StoreProxy) GetExecutions(ctx context.Context, sharedID string) ([]store.Execution, error) {
	return proxy.store.GetExecutions(ctx, sharedID)
}

// UpdateExecutionState implements store.ExecutionStore
func (proxy *StoreProxy) UpdateExecutionState(ctx context.Context, request store.UpdateExecutionStateRequest) error {
	err := proxy.store.UpdateExecutionState(ctx, request)
	//check json file exists in .bacalhau config dir
	jsonFilepath := EnsureJobStatsJSONExists()

	if err == nil && request.NewState == store.ExecutionStateCompleted {
		//write to file
		var jobStore JobStats
		//Read json file into byte string
		jsonbs, err := os.ReadFile(jsonFilepath)
		fmt.Println("jsonbs is", string(jsonbs))
		if err != nil {
			log.Fatal(err)
		}
		//unmarshall byte string and store in jobStore
		err = json.Unmarshal(jsonbs, &jobStore)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println("unmarshalling successful")
		//update jobCount
		jobStore.JobsCompleted++
		//write back to file
		bs, err := json.Marshal(jobStore)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("marshaling successful")
		err = os.WriteFile(jsonFilepath, bs, 0666)
		if err != nil {
			return err
		}
	}
	return err
}

// does StoreProxy implement
var _ store.ExecutionStore = (*StoreProxy)(nil)
