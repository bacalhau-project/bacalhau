package utils

import (
	"encoding/json"
	"fmt"

	"github.com/PaesslerAG/jsonpath"
)

type DynamicJSON struct {
	data interface{}
}

func ParseToDynamicJSON(jsonStr string) (*DynamicJSON, error) {
	var data interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}
	return &DynamicJSON{data: data}, nil
}

// Query executes a JSONPath query and returns the result
// Example queries:
// $.store.book[0].title
// $.store.book[?(@.price < 10)]
// $.store.book[*].author
func (d *DynamicJSON) Query(path string) (*JSONValue, error) {
	val, err := jsonpath.Get(path, d.data)
	if err != nil {
		return &JSONValue{}, err
	}
	return &JSONValue{value: val}, nil
}

// JSONValue wraps a value and provides type-safe access
type JSONValue struct {
	value interface{}
}

// String returns the string value or empty string if not a string
func (v *JSONValue) String() string {
	if str, ok := v.value.(string); ok {
		return str
	}
	panic("value not a string")
}

// Float returns the float64 value or 0 if not a number
func (v *JSONValue) Float() float64 {
	switch n := v.value.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	default:
		panic("value not a float")
	}
}

// Int returns the int value or 0 if not a number
func (v *JSONValue) Int() int {
	return int(v.Float())
}

// Bool returns the boolean value or false if not a boolean
func (v *JSONValue) Bool() bool {
	if b, ok := v.value.(bool); ok {
		return b
	}
	panic("value not a boolean")
}

// Array returns the array value or nil if not an array
func (v *JSONValue) Array() []interface{} {
	if arr, ok := v.value.([]interface{}); ok {
		return arr
	}
	panic("value not an array")
}

// Map returns the map value or nil if not a map
func (v *JSONValue) Map() map[string]interface{} {
	if m, ok := v.value.(map[string]interface{}); ok {
		return m
	}
	panic("value not a map")
}

// Raw returns the underlying value
func (v *JSONValue) Raw() interface{} {
	return v.value
}
