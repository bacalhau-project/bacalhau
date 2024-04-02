package challenge

// StringMarshaller is a struct that implements the encoding.BinaryMarshaler interface for strings.
// It holds a string value that can be marshaled into a byte slice.
type StringMarshaller struct {
	Input string
}

// NewStringMarshaller returns a pointer to a new StringMarshaller initialized with the given input string.
// This function is typically used to prepare a string for binary marshaling.
func NewStringMarshaller(input string) *StringMarshaller {
	return &StringMarshaller{Input: input}
}

// MarshalBinary implements the encoding.BinaryMarshaler interface.
// It converts the string held by StringMarshaller into a slice of bytes.
// As string to byte conversion in Go is straightforward and error-free, this method returns nil for the error.
func (m *StringMarshaller) MarshalBinary() ([]byte, error) {
	return []byte(m.Input), nil
}
