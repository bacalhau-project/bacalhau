package ptypes

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// Rate is a param that's parsed as "quantity/interval", where `quantity` is a
// float and `interval` is a string parsable by time.ParseDuration, e.g. "1s".
//
// You can omit the numeric component of the interval to default to 1, e.g.
// "100/s" is the same as "100/1s".
//
// Examples of valid Rate strings include: "100/s", "0.5/m", "500/5m".
type Rate struct {
	Quantity float64
	Interval time.Duration
}

var (
	_ json.Marshaler   = Duration{}
	_ json.Unmarshaler = &Duration{}
)

func (r Rate) MarshalJSON() ([]byte, error) {
	return nil, nil
}
func (r *Rate) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}

	str, ok := v.(string)
	if !ok {
		return errors.New("invalid rate param, must be string")
	}

	strs := strings.Split(str, "/")
	if len(strs) != 2 {
		return errors.New("invalid rate param. Must be in format  'quantity / interval'")
	}

	q, err := strconv.ParseFloat(strs[0], 64)
	if err != nil {
		return fmt.Errorf("error parsing quantity portion of rate: %s", err)
	}
	intervalStr := strings.TrimSpace(strs[1])
	if !unicode.IsDigit(rune(intervalStr[0])) {
		intervalStr = "1" + intervalStr
	}

	i, err := time.ParseDuration(intervalStr)
	if err != nil {
		return fmt.Errorf("error parsing interval portion of rate: %s", err)
	}

	r.Quantity = q
	r.Interval = i
	return nil
}
