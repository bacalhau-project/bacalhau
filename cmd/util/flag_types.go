package util

import (
	"fmt"
	"strconv"
)

// ============================================================================
// UintValue - Custom uint64 flag type
// ============================================================================

// UintValue implements pflag.Value for uint64 flags with custom error messages
type UintValue struct {
	value *uint64
}

func NewUintValue(val uint64, p *uint64) *UintValue {
	*p = val
	return &UintValue{value: p}
}

func (u *UintValue) Set(s string) error {
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return fmt.Errorf("'%s' is not a valid number: please provide a positive integer", s)
	}
	*u.value = v
	return nil
}

func (u *UintValue) Type() string {
	return "uint"
}

func (u *UintValue) String() string {
	return strconv.FormatUint(*u.value, 10)
}
