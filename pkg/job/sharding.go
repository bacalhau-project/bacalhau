package job

import (
	"context"
	"fmt"
	"strings"

	doublestar "github.com/bmatcuk/doublestar/v4"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/storage"
)

/*
	givem a flat list of all files - group them using a glob pattern

	so if we have:

	 /a/file1.txt
	 /a/file2.txt
	 /b/file1.txt
	 /b/file2.txt

	the following is how different patterns would group:

	/* = [/a/, /b/]
	/**\/*.txt = [/a/file1.txt, /a/file2.txt, /b/file1.txt, /b/file2.txt]

*/
func ApplyGlobPattern(
	files []storage.StorageSpec,
	pattern string,
	basePath string,
) ([]storage.StorageSpec, error) {
	var result []storage.StorageSpec
	for _, file := range files {
		usePath := file.Path
		if basePath != "" {
			usePath = strings.TrimPrefix(file.Path, basePath)
		}
		matches, err := doublestar.Match(pattern, usePath)
		if err != nil {
			return result, err
		}
		if matches {
			result = append(result, file)
		}
	}
	return result, nil
}

func GetTotalExecutionCount(job executor.Job) int {
	shardCount := job.ExecutionPlan.TotalShards
	if shardCount == 0 {
		shardCount = 1
	}
	return job.Deal.Concurrency * shardCount
}

// given a sharding config and storage drivers
// explode the config into the total set of volumes
// we want to spread across jobs - this is before
// we group into batches using job.Spec.Sharding.BatchSize
func ExplodeShardedVolumes(
	ctx context.Context,
	spec executor.JobSpec,
	storageProviders map[storage.StorageSourceType]storage.StorageProvider,
) ([]storage.StorageSpec, error) {
	// this is an exploded list of all storage inodes
	// once the storage driver has expanded the volume
	// we will filter these using the glob pattern
	// and then group them based on batch size
	allVolumes := []storage.StorageSpec{}
	config := spec.Sharding

	// this means there is no sharding and we use the input volumes as is
	if config.GlobPattern == "" {
		return spec.Inputs, nil
	}

	// loop over each input volume and explode it using the storage driver
	for _, volume := range spec.Inputs {
		storageProvider, ok := storageProviders[volume.Engine]
		if !ok {
			return allVolumes, fmt.Errorf("storage provider not found for engine %s", volume.Engine)
		}
		explodedVolumes, err := storageProvider.Explode(ctx, volume)
		if !ok {
			return allVolumes, err
		}
		allVolumes = append(allVolumes, explodedVolumes...)
	}
	// let's filter all of the combined volumes down using the glob pattern
	return ApplyGlobPattern(allVolumes, config.GlobPattern, config.BasePath)
}

// given an exploded set of volumes - we now group them based on batch size
func GetShards(
	ctx context.Context,
	spec executor.JobSpec,
	storageProviders map[storage.StorageSourceType]storage.StorageProvider,
) ([][]storage.StorageSpec, error) {
	config := spec.Sharding
	batchSize := config.BatchSize
	if batchSize <= 0 {
		batchSize = 1
	}
	results := [][]storage.StorageSpec{}
	filteredVolumes, err := ExplodeShardedVolumes(ctx, spec, storageProviders)
	if err != nil {
		return results, err
	}
	currentArray := []storage.StorageSpec{}
	for _, volume := range filteredVolumes {
		currentArray = append(currentArray, volume)
		if len(currentArray) == int(batchSize) {
			results = append(results, currentArray)
			currentArray = []storage.StorageSpec{}
		}
	}
	if len(currentArray) > 0 {
		results = append(results, currentArray)
	}
	return results, nil
}

// used by executors to explode a volume spec into the
// things it should actually mount into the job once the sharding
// has been applied to a volume and a single shard is now running
func GetShard(
	ctx context.Context,
	spec executor.JobSpec,
	storageProviders map[storage.StorageSourceType]storage.StorageProvider,
	shard int,
) ([]storage.StorageSpec, error) {
	shards, err := GetShards(ctx, spec, storageProviders)
	if err != nil {
		return []storage.StorageSpec{}, err
	}
	if len(shards) <= int(shard) {
		return []storage.StorageSpec{}, fmt.Errorf("shard %d is out of range", shard)
	}
	return shards[shard], nil
}

// we explode each sharded volume and calculate the batch size
func GenerateExecutionPlan(
	ctx context.Context,
	spec executor.JobSpec,
	storageProviders map[storage.StorageSourceType]storage.StorageProvider,
) (executor.JobExecutionPlan, error) {
	config := spec.Sharding
	// this means there is no sharding and we use the input volumes as is
	if config.GlobPattern == "" {
		return executor.JobExecutionPlan{
			TotalShards: 1,
		}, nil
	}
	shards, err := GetShards(ctx, spec, storageProviders)
	if err != nil {
		return executor.JobExecutionPlan{}, err
	}
	return executor.JobExecutionPlan{
		TotalShards: len(shards),
	}, nil
}
