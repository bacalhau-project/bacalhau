//go:build unit || !integration

package system

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParsingGPUsWithMany(t *testing.T) {
	output := strings.Join([]string{
		"0, Tesla T4, 15360",
		"1, Tesla T1, 12345",
	}, "\n")

	gpus, err := parseNvidiaCliOutput(strings.NewReader(output))
	require.NoError(t, err)
	require.Len(t, gpus, 2)
	require.Equal(t, uint64(0), gpus[0].Index)
	require.Equal(t, "Tesla T4", gpus[0].Name)
	require.Equal(t, uint64(15360), gpus[0].Memory)
	require.Equal(t, uint64(1), gpus[1].Index)
	require.Equal(t, "Tesla T1", gpus[1].Name)
	require.Equal(t, uint64(12345), gpus[1].Memory)
}

func TestParsingGPUsWithOne(t *testing.T) {
	output := strings.Join([]string{
		"0, Tesla T4, 15360",
	}, "\n")

	gpus, err := parseNvidiaCliOutput(strings.NewReader(output))
	require.NoError(t, err)
	require.Len(t, gpus, 1)
	require.Equal(t, uint64(0), gpus[0].Index)
	require.Equal(t, "Tesla T4", gpus[0].Name)
	require.Equal(t, uint64(15360), gpus[0].Memory)
}

func TestParsingGPUsWithNone(t *testing.T) {
	output := strings.Join([]string{}, "\n")

	gpus, err := parseNvidiaCliOutput(strings.NewReader(output))
	require.NoError(t, err)
	require.Len(t, gpus, 0)
}
