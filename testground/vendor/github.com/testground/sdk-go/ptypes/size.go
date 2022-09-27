package ptypes

import (
	"encoding/json"
	"errors"

	"github.com/dustin/go-humanize"
)

// Size is a type that unmarshals human-readable binary sizes like "100 KB"
// into an uint64, where the unit is bytes.
type Size uint64

var (
	_ json.Marshaler   = Size(0)
	_ json.Unmarshaler = (*Size)(nil)
)

func (s Size) MarshalJSON() ([]byte, error) {
	return nil, nil
}

func (s *Size) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	str, ok := v.(string)
	if !ok {
		return errors.New("invalid size param, must be string")
	}
	n, err := humanize.ParseBytes(str)
	if err != nil {
		return err
	}
	*s = Size(n)
	return nil
}
