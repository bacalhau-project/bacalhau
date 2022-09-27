package k8sresource

import (
	"fmt"
	"strconv"
	"strings"
)

// CPU allows converstion between string and int representations of equivalent millicores
// and basic math operations on cpu types
type CPU struct {
	millicores int
}

// NewCPU returns a new CPU instance initialized at 0
func NewCPU() CPU {
	return CPU{0}
}

// NewCPUFromString parses a Kubernetes-style cpu string (e.g., 100m, 0.1)
func NewCPUFromString(c string) (CPU, error) {
	cpu, err := cpuIntFromString(c)
	if err != nil {
		return CPU{}, err
	}
	return CPU{cpu}, nil
}

// NewCPUFromFloat creates a new CPU instance initialized to an equivalent number of
// millicores (e.g., 0.5)
func NewCPUFromFloat(c float64) CPU {
	return CPU{int(c * 1000)}
}

// Add will parse the CPUexpressed as a string and return a new CPU instance
// equal to the sum of the current instance plus m
func (cpu CPU) Add(c string) (CPU, error) {
	ci, err := cpuIntFromString(c)
	if err != nil {
		return CPU{}, err
	}
	return CPU{cpu.millicores + ci}, nil
}

// Sub will parse the CPU expressed as a string and return a new CPU instance
// equal to the current instance minus m
func (cpu CPU) Sub(c string) (CPU, error) {
	ci, err := cpuIntFromString(c)
	if err != nil {
		return CPU{}, err
	}
	return CPU{cpu.millicores - ci}, nil
}

// AddF will return a new CPU instance equal to the sum of the current instance plus m
func (cpu CPU) AddF(c float64) CPU {
	return CPU{cpu.millicores + int(c*1000)}
}

// SubF will return a new CPU instance equal to the current instance minus m
func (cpu CPU) SubF(c float64) CPU {
	return CPU{cpu.millicores - int(c*1000)}
}

// ToString returns the Kubernetes-style CPU value as a string rounded to the nearest
// millicore
func (cpu CPU) ToString() string {
	return fmt.Sprintf("%dm", cpu.millicores)
}

// ToFloat64 returns the CPU value as a float representing fractions of a core
// (e.g., 500m = 0.5)
func (cpu CPU) ToFloat64() float64 {
	return float64(cpu.millicores) / 1000
}

// ToMillicores returns the CPU value as an int representing total millicores
func (cpu CPU) ToMillicores() int {
	return cpu.millicores
}

func cpuIntFromString(s string) (int, error) {
	switch {
	case strings.HasSuffix(s, "m"):
		i, err := strconv.Atoi(strings.TrimSuffix(s, "m"))
		if err != nil {
			return 0, fmt.Errorf("unknown cpu format: %s", s)
		}
		return i, nil
	default:
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0, fmt.Errorf("unknown cpu format: %s", s)
		}
		return int(f * 1000), nil
	}
}
