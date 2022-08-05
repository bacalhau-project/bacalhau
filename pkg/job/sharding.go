package job

import (
	"context"
	"fmt"
	"strings"

	doublestar "github.com/bmatcuk/doublestar/v4"
	"github.com/davecgh/go-spew/spew"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/storage"
)

func prependSlash(st string) string {
	if st == "" {
		return st
	}
	if !strings.HasPrefix(st, "/") {
		return "/" + st
	} else {
		return st
	}
}

/*
given a flat list of all files - group them using a glob pattern

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
	if pattern == "" {
		return files, nil
	}
	var result []storage.StorageSpec
	usePattern := prependSlash(pattern)
	useBasePath := prependSlash(basePath)
	for _, file := range files {
		file.Path = prependSlash(file.Path)

		usePath := file.Path

		// remove the base path from the file path because
		// we will apply the glob pattern from below the base path
		if useBasePath != "" {
			usePath = strings.TrimPrefix(file.Path, useBasePath)
		}

		matches, err := doublestar.Match(usePattern, usePath)
		if err != nil {
			return result, err
		}
		if matches {
			result = append(result, file)
		}
	}
	return result, nil
}

func GetJobTotalShards(job executor.Job) int {
	shardCount := job.ExecutionPlan.TotalShards
	if shardCount == 0 {
		shardCount = 1
	}
	return shardCount
}

func GetJobConcurrency(job executor.Job) int {
	concurrency := job.Deal.Concurrency
	if concurrency < 1 {
		concurrency = 1
	}
	return concurrency
}

func GetJobTotalExecutionCount(job executor.Job) int {
	return GetJobConcurrency(job) * GetJobTotalShards(job)
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

	fmt.Printf("allVolumes --------------------------------------\n")
	spew.Dump(allVolumes)

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
		if len(currentArray) == batchSize {
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
	// if we have no volumes at all - we are still processing
	// shard #0 so just return empty array
	if len(shards) == 0 {
		return []storage.StorageSpec{}, nil
	}
	if len(shards) <= shard {
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
	if len(shards) == 0 {
		return executor.JobExecutionPlan{}, fmt.Errorf("no sharding atoms found for glob pattern %s", config.GlobPattern)
	}
	return executor.JobExecutionPlan{
		TotalShards: len(shards),
	}, nil
}
