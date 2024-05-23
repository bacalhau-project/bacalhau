package kvstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

func migrateNodeInfoToNodeState(entry jetstream.KeyValueEntry) ([]byte, error) {
	var nodeinfo models.NodeInfo
	if err := json.Unmarshal(entry.Value(), &nodeinfo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal node info: %w", err)
	}

	nodestate := models.NodeState{
		Info:       nodeinfo,
		Membership: models.NodeMembership.PENDING,
		Connection: models.NodeStates.DISCONNECTED,
	}
	migratedData, err := json.Marshal(nodestate)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal node state: %w", err)
	}
	return migratedData, nil
}

func migrateJetStreamBucket(
	ctx context.Context,
	js jetstream.JetStream,
	from string,
	to string,
	migrateFunc func(entry jetstream.KeyValueEntry) ([]byte, error),
) (retErr error) {
	defer func() {
		if retErr == nil {
			if err := js.DeleteKeyValue(ctx, from); err != nil {
				if errors.Is(err, jetstream.ErrBucketNotFound) {
					// migration is successful since there isn't previous state to migrate from
					retErr = nil
				} else {
					retErr = fmt.Errorf("NodeStore migration succeeded, but failed to remove old bucket: %w", err)
				}
			}
		}
	}()

	fromKV, err := js.KeyValue(ctx, from)
	if err != nil {
		if errors.Is(err, jetstream.ErrBucketNotFound) {
			// migration is successful since there isn't previous state to migrate from
			return nil
		}
		return fmt.Errorf("NodeStore migration failed: failed to open 'from' bucket: %w", err)
	}

	keys, err := fromKV.Keys(ctx)
	if err != nil {
		if errors.Is(err, jetstream.ErrNoKeysFound) {
			// if the store is empty the migration is successful as there isn't anything to migrate
			return nil
		}
		return fmt.Errorf("NodeStore migration failed: failed to list store: %w", err)
	}

	start := time.Now()
	log.Info().Str("from_bucket", from).Str("to_bucket", to).Msgf("Begin NodeStore migration")
	toKV, err := js.CreateKeyValue(ctx, jetstream.KeyValueConfig{
		Bucket: to,
	})
	if err != nil {
		return fmt.Errorf("NodeStore migration failed: failed to open to bucket: %w", err)
	}

	for _, key := range keys {
		// Check if the key exists in the 'to' bucket
		_, err := toKV.Get(ctx, key)
		if err == nil {
			// Key already exists in the 'to' bucket, skip to the next key
			continue
		}
		if !errors.Is(err, jetstream.ErrKeyNotFound) {
			// An unexpected error occurred while checking the key in the 'to' bucket
			return fmt.Errorf("NodeStore migration failed: failed to check key in 'to' bucket: %w", err)
		}

		// Read the entry from the 'from' bucket
		entry, err := fromKV.Get(ctx, key)
		if err != nil {
			return fmt.Errorf("NodeStore migration failed: failed to read entry with key: %s: %w", key, err)
		}

		// Apply the migration function
		migratedData, err := migrateFunc(entry)
		if err != nil {
			return fmt.Errorf("NodeStore migration failed: %w", err)
		}

		// Write the migrated data to the 'to' bucket
		if _, err := toKV.Put(ctx, key, migratedData); err != nil {
			return fmt.Errorf("NodeStore migration failed: failed to write migrated data to store: %w", err)
		}
	}

	log.Info().Str("from_bucket", from).Str("to_bucket", to).Str("duration", time.Since(start).String()).
		Msgf("Completed NodeStore migration")
	return nil
}
