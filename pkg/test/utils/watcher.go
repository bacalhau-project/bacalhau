package testutils

import (
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/lib/envelope"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	boltdb_watcher "github.com/bacalhau-project/bacalhau/pkg/lib/watcher/boltdb"
	watchertest "github.com/bacalhau-project/bacalhau/pkg/lib/watcher/test"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

const (
	TypeString = "string"
)

func CreateComputeEventStore(t *testing.T) watcher.EventStore {
	eventObjectSerializer := watcher.NewJSONSerializer()
	err := errors.Join(
		eventObjectSerializer.RegisterType(compute.EventObjectExecutionUpsert, reflect.TypeOf(models.ExecutionUpsert{})),
		eventObjectSerializer.RegisterType(compute.EventObjectExecutionEvent, reflect.TypeOf(models.Event{})),
	)
	require.NoError(t, err)

	database := watchertest.CreateBoltDB(t)

	eventStore, err := boltdb_watcher.NewEventStore(database,
		boltdb_watcher.WithEventsBucket("events"),
		boltdb_watcher.WithCheckpointBucket("checkpoints"),
		boltdb_watcher.WithEventSerializer(eventObjectSerializer),
	)
	require.NoError(t, err)
	return eventStore
}

func CreateJobEventStore(t *testing.T) watcher.EventStore {
	eventObjectSerializer := watcher.NewJSONSerializer()
	err := errors.Join(
		eventObjectSerializer.RegisterType(jobstore.EventObjectExecutionUpsert, reflect.TypeOf(models.ExecutionUpsert{})),
		eventObjectSerializer.RegisterType(jobstore.EventObjectEvaluation, reflect.TypeOf(models.Event{})),
	)
	require.NoError(t, err)

	database := watchertest.CreateBoltDB(t)

	eventStore, err := boltdb_watcher.NewEventStore(database,
		boltdb_watcher.WithEventsBucket("events"),
		boltdb_watcher.WithCheckpointBucket("checkpoints"),
		boltdb_watcher.WithEventSerializer(eventObjectSerializer),
	)
	require.NoError(t, err)
	return eventStore
}

func CreateStringEventStore(t *testing.T) (watcher.EventStore, *envelope.Registry) {
	eventObjectSerializer := watcher.NewJSONSerializer()
	require.NoError(t, eventObjectSerializer.RegisterType(TypeString, reflect.TypeOf("")))

	database := watchertest.CreateBoltDB(t)

	eventStore, err := boltdb_watcher.NewEventStore(database,
		boltdb_watcher.WithEventsBucket("events"),
		boltdb_watcher.WithCheckpointBucket("checkpoints"),
		boltdb_watcher.WithEventSerializer(eventObjectSerializer),
	)
	require.NoError(t, err)

	registry := envelope.NewRegistry()
	require.NoError(t, registry.Register(TypeString, ""))

	return eventStore, registry
}
