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
	"github.com/bacalhau-project/bacalhau/pkg/storage/util"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

type StoreProxy struct {
	store store.ExecutionStore
}

type JobStats struct {
	JobsCompleted uint
}

// Check if the json file exists, and create it if it doesn't.
func EnsureJobStatsJSONExists() (string, error) {
	configDir, err := system.EnsureConfigDir()
	if err != nil {
		return "", err
	}
	jsonFilepath := filepath.Join(configDir, "jobStats.json")
	if _, err := os.Stat(jsonFilepath); errors.Is(err, os.ErrNotExist) {
		log.Println(err, ": Creating jobStats.json...")
		f, err := os.Create(jsonFilepath)
		if err != nil {
			return "", fmt.Errorf("failed to create JSON file %s: %w", jsonFilepath, err)
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
func (proxy *StoreProxy) GetExecutionCount(ctx context.Context) (uint, error) {
	jsonFilepath, err := EnsureJobStatsJSONExists()
	if err != nil {
		return 0, err
	}
	jsonbs, err := os.ReadFile(jsonFilepath)
	var jobStore JobStats
	if err != nil {
		return 0, err
	}
	err = json.Unmarshal(jsonbs, &jobStore)
	if err != nil {
		return 0, err
	}
	return jobStore.JobsCompleted, nil
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
	if err != nil {
		return fmt.Errorf("failed to update execution state: %w", err)
	}
	//check json file exists in .bacalhau config dir
	jsonFilepath, err := EnsureJobStatsJSONExists()
	if err != nil {
		return err
	}
	if request.NewState == store.ExecutionStateCompleted {
		var jobStore JobStats
		jsonbs, err := os.ReadFile(jsonFilepath)
		if err != nil {
			return err
		}
		err = json.Unmarshal(jsonbs, &jobStore)
		if err != nil {
			return err
		}
		jobStore.JobsCompleted++
		bs, err := json.Marshal(jobStore)
		if err != nil {
			return err
		}
		err = os.WriteFile(jsonFilepath, bs, util.OS_USER_RW|util.OS_ALL_R)
		if err != nil {
			return err
		}
	}
	return err
}

// does StoreProxy implement
var _ store.ExecutionStore = (*StoreProxy)(nil)
