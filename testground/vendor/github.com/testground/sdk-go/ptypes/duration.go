package ptypes

import (
	"encoding/json"
	"errors"
	"time"
)

// Duration wraps a time.Duration and provides JSON marshal logic.
type Duration struct {
	time.Duration
}

var (
	_ json.Marshaler   = Duration{}
	_ json.Unmarshaler = &Duration{}
)

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case float64:
		d.Duration = time.Duration(value)
		return nil
	case string:
		var err error
		d.Duration, err = time.ParseDuration(value)
		if err != nil {
			return err
		}
		return nil
	default:
		return errors.New("invalid duration")
	}
}
