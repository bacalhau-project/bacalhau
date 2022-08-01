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

func GetTotalJobShards(job executor.Job) uint {
	shardCount := job.ExecutionPlan.TotalShards
	if shardCount == 0 {
		shardCount = 1
	}
	return job.Deal.Concurrency * shardCount
}

// we explode each sharded volume and calculate the batch size
func ProcessJobSharding(
	ctx context.Context,
	job executor.Job,
	storageProviders map[storage.StorageSourceType]storage.StorageProvider,
) (executor.Job, error) {
	config := job.Spec.Sharding
	if config.GlobPattern == "" {
		job.ExecutionPlan = executor.JobExecutionPlan{
			TotalShards: 1,
		}
		return job, nil
	}

	// this is an exploded list of all storage inodes
	// once the storage driver has expanded the volume
	// we will filter these using the glob pattern
	// and then group them based on batch size
	allVolumes := []storage.StorageSpec{}
	for _, volume := range job.Spec.Inputs {
		storageProvider, ok := storageProviders[volume.Engine]
		if !ok {
			return job, fmt.Errorf("storage provider not found for engine %s", volume.Engine)
		}
		explodedVolumes, err := storageProvider.Explode(ctx, volume)
		if !ok {
			return job, err
		}
		allVolumes = append(allVolumes, explodedVolumes...)
	}

	// let's filter all of the combined volumes down using the glob pattern
	filteredVolumes, err := ApplyGlobPattern(allVolumes, config.GlobPattern, "")
	if err != nil {
		return job, err
	}
	fmt.Printf("filteredVolumes --------------------------------------\n")
	spew.Dump(filteredVolumes)
	return job, nil
}
