package marshaller

// Marshaller defines methods for marshaling and unmarshaling data, and normalizing it if applicable.
type Marshaller interface {
	// Marshal converts the given object into a byte slice.
	Marshal(interface{}) ([]byte, error)
	// Unmarshal decodes data into the given object, and normalizes it if applicable.
	Unmarshal([]byte, interface{}) error
}
