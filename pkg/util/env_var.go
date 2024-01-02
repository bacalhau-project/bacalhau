package util

import "os"

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
