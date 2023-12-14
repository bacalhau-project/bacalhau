package util

import "fmt"

// InterfaceToStringArray converts an interface{} that we know is a []string
// to that []string via []interface{}. This is useful when we have a map[string]interface{}
// and we want to get the []string{} out of it.
func InterfaceToStringArray(source interface{}) ([]string, error) {
	if source == nil {
		return nil, nil
	}

	sourceArray, ok := source.([]interface{})
	if !ok {
		return nil, fmt.Errorf("expected []interface{} but got %T", source)
	}

	result := make([]string, len(sourceArray))
	for i, v := range sourceArray {
		result[i] = fmt.Sprint(v)
	}

	return result, nil
}
