package sharding

import (
	doublestar "github.com/bmatcuk/doublestar/v4"
)

/*
  ipfs resolver will explode a CID into a flat list of file paths
*/
func ExplodeCid(files []string, pattern string) ([]string, error) {
	var result []string
	for _, file := range files {
		matches, err := doublestar.Match(pattern, file)
		if err != nil {
			return result, err
		}
		if matches {
			result = append(result, file)
		}
	}
	return result, nil
}
