package inlocalstore

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/storage/util"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
)

type PersistentJobStore struct {
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
		log.Debug().Err(err).Msg("Creating jobStats.json")
		//Initialise JSON with counter of 0
		err = writeCounter(jsonFilepath, 0)
		if err != nil {
			return "", err
		}
	}
	return jsonFilepath, nil
}

func NewPersistentJobStore(store store.ExecutionStore) *PersistentJobStore {
	return &PersistentJobStore{
		store: store,
	}
}

// CreateExecution implements store.ExecutionStore
func (proxy *PersistentJobStore) CreateExecution(ctx context.Context, execution store.Execution) error {
	return proxy.store.CreateExecution(ctx, execution)
}

// DeleteExecution implements store.ExecutionStore
func (proxy *PersistentJobStore) DeleteExecution(ctx context.Context, id string) error {
	return proxy.store.DeleteExecution(ctx, id)
}

// GetExecution implements store.ExecutionStore
func (proxy *PersistentJobStore) GetExecution(ctx context.Context, id string) (store.Execution, error) {
	return proxy.store.GetExecution(ctx, id)
}

// GetExecutionCount implements store.ExecutionStore
func (proxy *PersistentJobStore) GetExecutionCount(ctx context.Context) (uint, error) {
	jsonFilepath, err := EnsureJobStatsJSONExists()
	if err != nil {
		return 0, err
	}
	return readCounter(jsonFilepath)
}

// GetExecutionHistory implements store.ExecutionStore
func (proxy *PersistentJobStore) GetExecutionHistory(ctx context.Context, id string) ([]store.ExecutionHistory, error) {
	return proxy.store.GetExecutionHistory(ctx, id)
}

// GetExecutions implements store.ExecutionStore
func (proxy *PersistentJobStore) GetExecutions(ctx context.Context, sharedID string) ([]store.Execution, error) {
	return proxy.store.GetExecutions(ctx, sharedID)
}

// UpdateExecutionState implements store.ExecutionStore
func (proxy *PersistentJobStore) UpdateExecutionState(ctx context.Context, request store.UpdateExecutionStateRequest) error {
	err := proxy.store.UpdateExecutionState(ctx, request)
	if err != nil {
		return err
	}
	//check json file exists in .bacalhau config dir
	jsonFilepath, err := EnsureJobStatsJSONExists()
	if err != nil {
		return err
	}
	if request.NewState == store.ExecutionStateCompleted {
		count, err := readCounter(jsonFilepath)
		if err != nil {
			return err
		}
		err = writeCounter(jsonFilepath, count+1)
		if err != nil {
			return err
		}
	}
	return err
}

func writeCounter(filepath string, count uint) error {
	var jobStore JobStats
	jobStore.JobsCompleted += count
	bs, err := json.Marshal(jobStore)
	if err != nil {
		return err
	}
	err = os.WriteFile(filepath, bs, util.OS_USER_RW|util.OS_ALL_R)
	if err != nil {
		return err
	}
	return err
}

func readCounter(filepath string) (uint, error) {
	jsonbs, err := os.ReadFile(filepath)
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

var _ store.ExecutionStore = (*PersistentJobStore)(nil)
