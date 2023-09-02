package types

import "time"

// Duration is a wrapper type for time.Duration
// for decoding and encoding from/to YAML, etc.
type Duration time.Duration

// UnmarshalText implements interface for YAML decoding
func (dur *Duration) UnmarshalText(text []byte) error {
	d, err := time.ParseDuration(string(text))
	if err != nil {
		return err
	}
	*dur = Duration(d)
	return err
}

func (dur Duration) MarshalText() ([]byte, error) {
	d := time.Duration(dur)
	return []byte(d.String()), nil
}

func (dur Duration) String() string {
	return time.Duration(dur).String()
}
