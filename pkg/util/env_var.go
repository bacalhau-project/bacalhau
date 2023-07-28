package util

import (
	"fmt"
	"os"
)

type EnvParserFunc[T any] func(string) (T, error)

func GetEnvAs[T any](envVar string, deflt T, parser EnvParserFunc[T]) T {
	v := os.Getenv(envVar)
	if v != "" {
		r, e := parser(v)
		if e == nil {
			return r
		}
	}

	return deflt
}

func GetEnv(envVar string, deflt string) string {
	v := os.Getenv(envVar)
	if v != "" {
		return v
	}

	return deflt
}

// FlattenMap is used for flattening an env var map
// for use with docker runtime
func FlattenMap(m map[string]string) []string {
	s := make([]string, 0, len(m))

	for k, v := range m {
		s = append(s, fmt.Sprintf("%s=%s", k, v))
	}

	return s
}
