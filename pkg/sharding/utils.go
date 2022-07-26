package sharding

import (
	doublestar "github.com/bmatcuk/doublestar/v4"
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
func Group(files []string, pattern string) ([]string, error) {
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
