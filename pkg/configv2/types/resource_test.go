//go:build unit || !integration

package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

func TestScaler(t *testing.T) {
	systemCapacity := models.Resources{
		CPU:    10,
		Memory: 10,
		Disk:   10,
		GPU:    1,
	}

	scaler := ResourceScaler{
		CPU:    "80%",
		Memory: "70%",
		Disk:   "60%",
		GPU:    "50%",
	}

	out, err := scaler.ToResource(systemCapacity)
	require.NoError(t, err)

	assert.EqualValues(t, 8, out.CPU)
	assert.EqualValues(t, 7, out.Memory)
	assert.EqualValues(t, 6, out.Disk)
	assert.EqualValues(t, 1, out.GPU)
}

func TestResources(t *testing.T) {
	systemCapacity := models.Resources{
		CPU:    10,
		Memory: 10,
		Disk:   10,
		GPU:    1,
	}

	scaler := ResourceScaler{
		CPU:    "10000m",
		Memory: "10GB",
		Disk:   "100GB",
		GPU:    "4",
	}

	out, err := scaler.ToResource(systemCapacity)
	require.NoError(t, err)

	assert.EqualValues(t, 10, out.CPU)
	assert.EqualValues(t, 10_000_000_000, out.Memory)
	assert.EqualValues(t, 100_000_000_000, out.Disk)
	assert.EqualValues(t, 4, out.GPU)
}
