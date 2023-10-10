package marshaller

import (
	"bytes"
	"encoding/gob"
	"fmt"
)

// BinaryMarshaller uses gob encoding for marshaling.
type BinaryMarshaller struct{}

// NewBinaryMarshaller initializes and returns a new BinaryMarshaller.
func NewBinaryMarshaller() *BinaryMarshaller {
	return &BinaryMarshaller{}
}

// Marshal converts the given object into a gob-encoded byte slice.
func (BinaryMarshaller) Marshal(obj interface{}) ([]byte, error) {
	var b bytes.Buffer
	e := gob.NewEncoder(&b)

	if err := e.Encode(obj); err != nil {
		return nil, fmt.Errorf("gob encode: %w", err)
	}

	return b.Bytes(), nil
}

// Unmarshal decodes gob data into the given object.
func (BinaryMarshaller) Unmarshal(data []byte, obj interface{}) error {
	b := bytes.NewBuffer(data)
	d := gob.NewDecoder(b)

	if err := d.Decode(obj); err != nil {
		return fmt.Errorf("gob decode: %w", err)
	}

	normalizeIfApplicable(obj)
	return nil
}

// compile-time check that BinaryMarshaller implements Marshaller
var _ Marshaller = BinaryMarshaller{}
