package marshaller

import (
	"encoding/json"
	"fmt"
)

// JSONMarshaller uses JSON encoding for marshaling.
type JSONMarshaller struct{}

// NewJSONMarshaller initializes and returns a new JSONMarshaller.
func NewJSONMarshaller() *JSONMarshaller {
	return &JSONMarshaller{}
}

// Marshal converts the given object into a JSON byte slice.
func (JSONMarshaller) Marshal(obj interface{}) ([]byte, error) {
	return json.Marshal(obj)
}

// Unmarshal decodes JSON data into the given object and normalizes it if applicable.
func (JSONMarshaller) Unmarshal(data []byte, obj interface{}) error {
	err := json.Unmarshal(data, obj)
	if err != nil {
		return fmt.Errorf("json unmarshal: %w", err)
	}

	normalizeIfApplicable(obj)
	return nil
}

// compile-time check that JSONMarshaller implements Marshaller
var _ Marshaller = JSONMarshaller{}
