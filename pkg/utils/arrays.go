package utils

import (
	"fmt"
	"strings"
)

func ContainsString(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func ContainsInt(s []int, e int) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func JoinMapKeysValues(s map[string]string) (string, error) {
	values := make([]string, 0, len(s))
	for k, v := range s {
		values = append(values, fmt.Sprintf("%v='%v'", k, v))
	}
	return strings.Join(values, ", "), nil
}

func AppendIfMissing(slice []string, s string) []string {
	if ContainsString(slice, s) {
		return slice
	}
	return append(slice, s)
}

func RemoveIndex(s []int, index int) []int {
	ret := make([]int, 0)
	ret = append(ret, s[:index]...)
	return append(ret, s[index+1:]...)
}

func ValueOrDefault(s string, d string) string {
	if s != "" {
		return s
	} else {
		return d
	}
}