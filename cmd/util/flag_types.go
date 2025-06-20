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

// ============================================================================
// BoolValue - Custom bool flag type
// ============================================================================

// BoolValue implements pflag.Value for bool flags with custom error messages
type BoolValue struct {
	value *bool
}

func NewBoolValue(val bool, p *bool) *BoolValue {
	*p = val
	return &BoolValue{value: p}
}

func (b *BoolValue) Set(s string) error {
	v, err := strconv.ParseBool(s)
	if err != nil {
		return fmt.Errorf("'%s' is not a valid boolean: please provide 'true' or 'false'", s)
	}
	*b.value = v
	return nil
}

func (b *BoolValue) Type() string {
	return "bool"
}

func (b *BoolValue) String() string {
	return strconv.FormatBool(*b.value)
}
