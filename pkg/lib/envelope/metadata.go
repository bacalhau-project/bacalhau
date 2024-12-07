package envelope

import (
	"strconv"
	"time"
)

// Metadata keys
const (
	KeyMessageType     = "Bacalhau-Type"
	KeyPayloadEncoding = "Bacalhau-PayloadEncoding"
)

// Metadata contains metadata about the message
type Metadata map[string]string

// ToMap returns the Metadata as a regular map[string]string
func (m Metadata) ToMap() map[string]string {
	return m
}

// ToHeaders returns the Metadata as a map[string][]string
func (m Metadata) ToHeaders() map[string][]string {
	headers := make(map[string][]string, len(m))
	for k, v := range m {
		headers[k] = []string{v}
	}
	return headers
}

// NewMetadataFromMap creates a new shallow copy Metadata object from a map.
// Changes to the map will be reflected in the Metadata object, but more efficient than NewMetadataFromMapCopy
func NewMetadataFromMap(m map[string]string) *Metadata {
	if m == nil {
		return &Metadata{}
	}
	metadata := Metadata(m)
	return &metadata
}

// NewMetadataFromMapCopy creates a new deepcopy Metadata object from a map.
// Changes to the map will not be reflected in the Metadata object
func NewMetadataFromMapCopy(m map[string]string) *Metadata {
	metadata := make(Metadata, len(m))
	for k, v := range m {
		metadata[k] = v
	}
	return &metadata
}

// Get returns the value for a given key, or an empty string if the key doesn't exist
func (m Metadata) Get(key string) string {
	return m[key]
}

// Has checks if a key exists in the metadata
func (m Metadata) Has(key string) bool {
	_, ok := m[key]
	return ok
}

// Set sets the value for a given key
func (m Metadata) Set(key, value string) {
	m[key] = value
}

// SetInt sets the value for a given key as an int
func (m Metadata) SetInt(key string, value int) {
	m[key] = strconv.Itoa(value)
}

// SetInt64 sets the value for a given key as an int64
func (m Metadata) SetInt64(key string, value int64) {
	m[key] = strconv.FormatInt(value, 10)
}

// SetTime sets the value for a given key as a time.Time
func (m Metadata) SetTime(key string, value time.Time) {
	m.SetInt64(key, value.UnixNano())
}

// GetInt gets the value as an int, returning 0 if the key doesn't exist or the value isn't a valid int
func (m Metadata) GetInt(key string) int {
	if val, ok := m[key]; ok {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return 0
}

// GetInt64 gets the value as an int64, returning 0 if the key doesn't exist or the value isn't a valid int64
func (m Metadata) GetInt64(key string) int64 {
	if val, ok := m[key]; ok {
		if i, err := strconv.ParseInt(val, 10, 64); err == nil {
			return i
		}
	}
	return 0
}

// GetUint64 gets the value as a uint64, returning 0 if the key doesn't exist or the value isn't a valid uint64
func (m Metadata) GetUint64(key string) uint64 {
	if val, ok := m[key]; ok {
		if i, err := strconv.ParseUint(val, 10, 64); err == nil {
			return i
		}
	}
	return 0
}

// GetTime gets the value as a time.Time, returning the zero time if the key doesn't exist or the value isn't a valid time
func (m Metadata) GetTime(key string) time.Time {
	if val, ok := m[key]; ok {
		if i, err := strconv.ParseInt(val, 10, 64); err == nil {
			return time.Unix(0, i)
		}
	}
	return time.Time{}
}
