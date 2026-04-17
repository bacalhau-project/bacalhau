package marshaller

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/c2h5oh/datasize"
	"sigs.k8s.io/yaml"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type KeyString string
type KeyInt int

const MaxSerializedStringInput = int(10 * datasize.MB)

// Arbitrarily choosing 1000 jobs to serialize - this is a pretty high
const MaxNumberOfObjectsToSerialize = 1000

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
	switch marshalType {
	case jsonMarshal:
		return json.Marshal(t)
	case jsonMarshalIndent:
		return json.MarshalIndent(t, "", strings.Repeat(" ", indentSpaces))
	case yamlMarshal:
		return yaml.Marshal(t)
	default:
		return nil, fmt.Errorf("unknown marshal type %d", marshalType)
	}
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
	switch unmarshalType {
	case jsonUnmarshal:
		return json.Unmarshal(b, t)
	case yamlUnmarshal:
		// Our format requires that we use the 	"sigs.k8s.io/yaml" library
		return yaml.Unmarshal(b, t)
	default:
		return fmt.Errorf("unknown unmarshal type")
	}
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

// normalizeIfApplicable attempts to normalize the object if it implements the Normalizable interface.
func normalizeIfApplicable(obj interface{}) {
	if normalizable, ok := obj.(models.Normalizable); ok {
		normalizable.Normalize()
	}
}

// UnmarshalJob unmarshalled `in` into a models.Job. It returns an error if:
// - `in` cannot be marshaled to json.
// - `in` contains an un-settable or unknown field.
// - `in` cannot be marshaled into a models.Job.
func UnmarshalJob(in []byte) (*models.Job, error) {
	// json is a subset of yaml, if `in` is already json this is a noop.
	jsonBytes, err := yaml.YAMLToJSON(in)
	if err != nil {
		return nil, fmt.Errorf("converting yaml to json: %w", err)
	}

	// unmarshal the job into a generic map
	var raw map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &raw); err != nil {
		return nil, err
	}
	// error(s) if un-settable or unknown fields are provided.
	if err := validateRawJob(raw); err != nil {
		return nil, err
	}

	var out *models.Job
	if err := json.Unmarshal(jsonBytes, &out); err != nil {
		switch v := err.(type) {
		// json.UnmarshalTypeError describes a JSON value that as not appropriate for the value of the specified Go type.
		// e.g. if the Count field was not an int, or the Name field was not a string
		case *json.UnmarshalTypeError:
			return nil, fmt.Errorf("field: '%s' in '%s' is invalid type: '%s' expected: '%s'", v.Field, v.Struct, v.Value, v.Type.String())
		default:
			return nil, err
		}
	}
	return out, nil
}

// validateRawJob returns an error if `jobSpec` contains a job field that may not be set by the user or if unknown
// fields were included in `jobSpec`.
func validateRawJob(jobSpec map[string]interface{}) error {
	var mErr error
	// first check if any un-settable fields were provided.
	for key := range jobSpec {
		specKey := strings.ToLower(key)
		if _, found := disallowedKeys[specKey]; found {
			mErr = errors.Join(mErr, fmt.Errorf("field: '%s' in 'Job' is not allowed", key))
		}
	}
	// remove all known fields from the provided spec.
	for key := range jobSpec {
		specKey := strings.ToLower(key)
		if _, ok := disallowedKeys[specKey]; ok {
			delete(jobSpec, key)
		}
		if _, ok := allowedKeys[specKey]; ok {
			delete(jobSpec, key)
		}
	}
	// any fields that remain are considered unknown
	for key := range jobSpec {
		mErr = errors.Join(mErr, fmt.Errorf("unknown field: '%s' in 'Job'", key))
	}
	return mErr
}

var (
	// disallowedKeys contains system fields that users may not set when providing a job spec.
	disallowedKeys = map[string]struct{}{
		"id":         {},
		"state":      {},
		"version":    {},
		"revision":   {},
		"createtime": {},
		"modifytime": {},
	}
	// allowedKeys contains fields users are permitted to set when providing a job spec.
	allowedKeys = map[string]struct{}{
		"name":        {},
		"namespace":   {},
		"type":        {},
		"priority":    {},
		"count":       {},
		"constraints": {},
		"meta":        {},
		"labels":      {},
		"tasks":       {},
	}
)
