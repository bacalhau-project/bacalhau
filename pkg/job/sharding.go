package job

import (
	"context"
	"fmt"
	"strings"

	doublestar "github.com/bmatcuk/doublestar/v4"
	"github.com/filecoin-project/bacalhau/pkg/model"
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
	files []model.StorageSpec,
	pattern string,
	basePath string,
) ([]model.StorageSpec, error) {
	if pattern == "" {
		return files, nil
	}
	var result []model.StorageSpec
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

func GetJobTotalShards(j model.Job) int {
	shardCount := j.Spec.ExecutionPlan.TotalShards
	if shardCount == 0 {
		shardCount = 1
	}
	return shardCount
}

func GetJobConcurrency(j model.Job) int {
	concurrency := j.Spec.Deal.Concurrency
	if concurrency < 1 {
		concurrency = 1
	}
	return concurrency
}

func GetJobTotalExecutionCount(j model.Job) int {
	return GetJobConcurrency(j) * GetJobTotalShards(j)
}

// given a sharding config and storage drivers
// explode the config into the total set of volumes
// we want to spread across jobs - this is before
// we group into batches using job.Spec.Sharding.BatchSize
func ExplodeShardedVolumes(
	ctx context.Context,
	spec model.Spec,
	storageProviders storage.StorageProvider,
) ([]model.StorageSpec, error) {
	// this is an exploded list of all storage inodes
	// once the storage driver has expanded the volume
	// we will filter these using the glob pattern
	// and then group them based on batch size
	allVolumes := []model.StorageSpec{}
	config := spec.Sharding

	// this means there is no sharding and we use the input volumes as is
	if config.GlobPattern == "" {
		return spec.Inputs, nil
	}

	// loop over each input volume and explode it using the storage driver
	for _, volume := range spec.Inputs {
		volumeStorage, err := storageProviders.Get(ctx, volume.StorageSource)
		if err != nil {
			return allVolumes, err
		}
		explodedVolumes, err := volumeStorage.Explode(ctx, volume)
		if err != nil {
			return allVolumes, err
		}
		allVolumes = append(allVolumes, explodedVolumes...)
	}
	// let's filter all of the combined volumes down using the glob pattern
	return ApplyGlobPattern(allVolumes, config.GlobPattern, config.BasePath)
}

// given an exploded set of volumes - we now group them based on batch size
func GetShardsStorageSpecs(
	ctx context.Context,
	spec model.Spec,
	storageProviders storage.StorageProvider,
) ([][]model.StorageSpec, error) {
	config := spec.Sharding
	batchSize := config.BatchSize
	if batchSize <= 0 {
		batchSize = 1
	}
	// this means there is no sharding and we use the input volumes as is
	if config.GlobPattern == "" {
		return [][]model.StorageSpec{spec.Inputs}, nil
	}
	results := [][]model.StorageSpec{}
	filteredVolumes, err := ExplodeShardedVolumes(ctx, spec, storageProviders)
	if err != nil {
		return results, err
	}
	currentArray := []model.StorageSpec{}
	for _, volume := range filteredVolumes {
		currentArray = append(currentArray, volume)
		if len(currentArray) == batchSize {
			results = append(results, currentArray)
			currentArray = []model.StorageSpec{}
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
func GetShardStorageSpec(
	ctx context.Context,
	shard model.JobShard,
	storageProviders storage.StorageProvider,
) ([]model.StorageSpec, error) {
	shards, err := GetShardsStorageSpecs(ctx, shard.Job.Spec, storageProviders)
	if err != nil {
		return []model.StorageSpec{}, err
	}

	// if we have no volumes at all - we are still processing
	// shard #0 so just return empty array
	if len(shards) == 0 {
		return []model.StorageSpec{}, nil
	}
	if len(shards) <= shard.Index {
		return []model.StorageSpec{}, fmt.Errorf("shard %s is out of range", shard)
	}
	return shards[shard.Index], nil
}

// we explode each sharded volume and calculate the batch size
func GenerateExecutionPlan(
	ctx context.Context,
	spec model.Spec,
	storageProviders storage.StorageProvider,
) (model.JobExecutionPlan, error) {
	config := spec.Sharding
	// this means there is no sharding and we use the input volumes as is
	if config.GlobPattern == "" {
		return model.JobExecutionPlan{
			TotalShards: 1,
		}, nil
	}
	shards, err := GetShardsStorageSpecs(ctx, spec, storageProviders)
	if err != nil {
		return model.JobExecutionPlan{}, err
	}
	if len(shards) == 0 {
		return model.JobExecutionPlan{}, fmt.Errorf("no sharding atoms found for glob pattern %s", config.GlobPattern)
	}
	return model.JobExecutionPlan{
		TotalShards: len(shards),
	}, nil
}
