//go:build unit || !integration

package marshaller

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Mock a Normalizable type
type MockNormalizable struct {
	Value string
}

func (m *MockNormalizable) Normalize() {
	m.Value = "normalized"
}

// Mock a NonNormalizable type
type MockNonNormalizable struct {
	Value string
}

func TestMarshaller(t *testing.T) {
	marshallers := []Marshaller{
		NewJSONMarshaller(),
		NewBinaryMarshaller(),
	}

	for _, m := range marshallers {
		t.Run(fmt.Sprintf("TestMarshal:%T", m), func(t *testing.T) {
			testStruct := MockNormalizable{Value: "test"}
			bytes, err := m.Marshal(testStruct)
			assert.NoError(t, err)

			var unmarshaled MockNormalizable
			err = m.Unmarshal(bytes, &unmarshaled)
			assert.NoError(t, err)
			assert.Equal(t, "normalized", unmarshaled.Value)
		})

		t.Run(fmt.Sprintf("TestUnmarshal:%T", m), func(t *testing.T) {
			normalizable := MockNormalizable{}
			nonNormalizable := MockNonNormalizable{}

			// Initially serialize the objects
			dataNormalizable, _ := m.Marshal(normalizable)
			dataNonNormalizable, _ := m.Marshal(nonNormalizable)

			// Unmarshal and check for normalization
			err := m.Unmarshal(dataNormalizable, &normalizable)
			assert.NoError(t, err)
			assert.Equal(t, "normalized", normalizable.Value)

			err = m.Unmarshal(dataNonNormalizable, &nonNormalizable)
			assert.NoError(t, err)
			assert.Equal(t, "", nonNormalizable.Value)
		})
	}
}
