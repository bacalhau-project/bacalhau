package job

import (
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
func ApplyGlobPattern(files []storage.StorageSpec, pattern string) ([]storage.StorageSpec, error) {
	var result []storage.StorageSpec
	for _, file := range files {
		matches, err := doublestar.Match(pattern, file.Path)
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
