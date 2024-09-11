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

/*
AsTimeDuration returns Duration as a time.Duration type.
It is only called in the code gen to set viper defaults.
This method exists for the following reasons:
 1. our config file is yaml, and we want to use durations for configuring things, printed in a human-readable format.
 2. time.Duration does not, and will not implement MarshalText/UnmarshalText
    - https://github.com/golang/go/issues/16039
 3. viper.GetDuration doesn't return an error, and instead returns `0` if it fails to cast
    - https://github.com/spf13/viper/blob/master/viper.go#L1048
    - https://github.com/spf13/cast/blob/master/caste.go#L61
    viper.GetDuration is unable to cast a types.Duration to a time.Duration (see second link), so it returns 0.

To meet the 3 constraints above we write durations into viper as a time.Duration, but cast them
to a types.Duration when they are added to our config structure. This allows us to:
1. Have human-readable times in out config file.
2. Have type safety on out cli, i.e. config set <duration_key> 10s
3. Return durations instead of `0` from viper.GetDuration.
*/
func (dur Duration) AsTimeDuration() time.Duration {
	return time.Duration(dur)
}
