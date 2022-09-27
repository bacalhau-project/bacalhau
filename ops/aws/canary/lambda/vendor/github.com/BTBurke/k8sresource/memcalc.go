// Package k8sresource provides utility functions for working with Kubernetes-style memory and cpu resources
// which are expressed as strings such as 256Mi or 2Gi for memory and 100m or 0.1 for CPU.  Methods are provided for converting
// between string representations and numeric values, and for math operations.
package k8sresource

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

const (
	_ = iota
	_
	// Mi = Megabytes
	Mi float64 = 1 << (10 * iota)
	// Gi = Gigabytes
	Gi
)

// Memory allows converstion between string and float representations and basic math operations on memory types
type Memory struct {
	m float64
}

// NewMem returns a new memory instance initialized at 0
func NewMem() Memory {
	return Memory{0}
}

// NewMemFromString parses a Kubernetes-style memory string (e.g., 256Mi, 1Gi)
func NewMemFromString(m string) (Memory, error) {
	f, err := memToFloat64(m)
	if err != nil {
		return Memory{}, err
	}
	return Memory{f}, nil
}

// NewMemFromFloat creates a new memory instance initialized to m
func NewMemFromFloat(m float64) Memory {
	return Memory{m}
}

// Add will parse the memory expressed as a string and return a new memory instance
// equal to the sum of the current instance plus m
func (s Memory) Add(m string) (Memory, error) {
	f, err := memToFloat64(m)
	if err != nil {
		return Memory{}, err
	}
	return Memory{s.m + f}, nil
}

// Sub will parse the memory expressed as a string and return a new memory instance
// equal to the current instance minus m
func (s Memory) Sub(m string) (Memory, error) {
	f, err := memToFloat64(m)
	if err != nil {
		return Memory{}, err
	}
	return Memory{s.m - f}, nil
}

// AddF will return a new memory instance equal to the sum of the current instance plus m
func (s Memory) AddF(m float64) Memory {
	return Memory{s.m + m}
}

// SubF will return a new memory instance equal to the current instance minus m
func (s Memory) SubF(m float64) Memory {
	return Memory{s.m - m}
}

// ToString returns the Kubernetes-style memory value as a string rounded up to the nearest
// megabyte.  Values over 1Gi will still be returned as an equivalent Mi value.
func (s Memory) ToString() string {
	return float64ToMi(s.m)
}

// ToFloat64 returns the memory value as a float
func (s Memory) ToFloat64() float64 {
	return s.m
}

func memToFloat64(s string) (float64, error) {
	switch {
	case strings.HasSuffix(s, "Mi"):
		mem, err := strconv.ParseFloat(strings.TrimSuffix(s, "Mi"), 64)
		if err != nil {
			return 0, fmt.Errorf("failed to convert memory string %s to float", s)
		}
		return mem * Mi, nil
	case strings.HasSuffix(s, "Gi"):
		mem, err := strconv.ParseFloat(strings.TrimSuffix(s, "Gi"), 64)
		if err != nil {
			return 0, fmt.Errorf("failed to convert memory string %s to float", s)
		}
		return mem * Gi, nil
	default:
		return 0, fmt.Errorf("failed to convert memory string %s to float, unknown units", s)
	}
}

func float64ToMi(m float64) string {
	return fmt.Sprintf("%dMi", int(math.Ceil(m/Mi)))
}
