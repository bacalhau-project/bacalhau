package v1alpha1

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/c2h5oh/datasize"
	"sigs.k8s.io/yaml"
)

type KeyString string
type KeyInt int

const MaxSerializedStringInput = int(10 * datasize.MB)
const MaxSerializedStringOutput = int(10 * datasize.MB)

// Arbitrarily choosing 1000 jobs to serialize - this is a pretty high
const MaxNumberOfObjectsToSerialize = 1000

const JSONIndentSpaceNumber = 4

const ShortIDLength = 8

func equal(a, b string) bool {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	return strings.EqualFold(a, b)
}

const (
	jsonMarshal = iota
	jsonMarshalIndent
	yamlMarshal
	jsonUnmarshal
	yamlUnmarshal
)

func JSONMarshalWithMax[T any](t T) ([]byte, error) {
	return genericMarshalWithMax(t, jsonMarshal, 0)
}

func JSONMarshalIndentWithMax[T any](t T, indentSpaces int) ([]byte, error) {
	return genericMarshalWithMax(t, jsonMarshalIndent, indentSpaces)
}

func YAMLMarshalWithMax[T any](t T) ([]byte, error) {
	return genericMarshalWithMax(t, yamlMarshal, -1)
}

// Create function to take generic and marshall func and return []byte and error
func genericMarshalWithMax[T any](t T, marshalType int, indentSpaces int) ([]byte, error) {
	err := ConfirmMaxSliceSize(t, MaxNumberOfObjectsToSerialize)
	if err != nil {
		return nil, fmt.Errorf("cannot serialize more than %d %s",
			MaxNumberOfObjectsToSerialize,
			reflect.TypeOf(t).String())
	}
	if marshalType == jsonMarshal {
		return json.Marshal(t)
	} else if marshalType == jsonMarshalIndent {
		return json.MarshalIndent(t, "", strings.Repeat(" ", indentSpaces))
	} else if marshalType == yamlMarshal {
		return yaml.Marshal(t)
	}

	return nil, fmt.Errorf("unknown marshal type %d", marshalType)
}

func JSONUnmarshalWithMax[T any](b []byte, t *T) error {
	return genericUnmarshalWithMax(b, t, jsonUnmarshal)
}

func YAMLUnmarshalWithMax[T any](b []byte, t *T) error {
	return genericUnmarshalWithMax(b, t, yamlUnmarshal)
}

func genericUnmarshalWithMax[T any](b []byte, t *T, unmarshalType int) error {
	if len(b) > MaxSerializedStringInput {
		return fmt.Errorf("size of bytes to unmarshal (%d) larger than maximum allowed (%d)",
			len(b),
			MaxSerializedStringInput)
	}
	if unmarshalType == jsonUnmarshal {
		return json.Unmarshal(b, t)
	} else if unmarshalType == yamlUnmarshal {
		// Our format requires that we use the 	"sigs.k8s.io/yaml" library
		return yaml.Unmarshal(b, t)
	}
	return fmt.Errorf("unknown unmarshal type")
}

func ConfirmMaxSliceSize[T any](t T, maxSize int) error {
	if _, isSlice := any(t).([]T); isSlice {
		tt := any(t).([]T)
		if len(tt) > maxSize {
			return fmt.Errorf("number of objects (%d) more than max (%d)", len(tt), maxSize)
		}
	}
	return nil
}

func GetShardID(jobID string, shardIndex int) string {
	return fmt.Sprintf("%s:%d", jobID, shardIndex)
}
