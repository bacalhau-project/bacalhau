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
	assert.EqualValues(t, []models.GPU{{}}, out.GPUs)
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
	assert.EqualValues(t, []models.GPU{{}, {}, {}, {}}, out.GPUs)
}

func TestScaleGPU(t *testing.T) {
	t.Run("scaling single GPU results in 1 GPU unit", func(t *testing.T) {
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

		// ensure 50% of a GPU is still 1 gpu
		assert.EqualValues(t, 1, out.GPU)
		assert.EqualValues(t, []models.GPU{{}}, out.GPUs)
	})

	t.Run("scaling zero GPU results in 0 GPU unit", func(t *testing.T) {
		systemCapacity := models.Resources{
			CPU:    10,
			Memory: 10,
			Disk:   10,
			GPU:    0,
		}

		scaler := ResourceScaler{
			CPU:    "80%",
			Memory: "70%",
			Disk:   "60%",
			GPU:    "50%",
		}

		out, err := scaler.ToResource(systemCapacity)
		require.NoError(t, err)

		// ensure 50% of a GPU is still 1 gpu
		assert.EqualValues(t, 0, out.GPU)
		assert.Empty(t, out.GPUs)
	})

	t.Run("scaling 2 GPU to 50% results in 1 GPU unit", func(t *testing.T) {
		systemCapacity := models.Resources{
			CPU:    10,
			Memory: 10,
			Disk:   10,
			GPU:    2,
		}

		scaler := ResourceScaler{
			CPU:    "80%",
			Memory: "70%",
			Disk:   "60%",
			GPU:    "50%",
		}

		out, err := scaler.ToResource(systemCapacity)
		require.NoError(t, err)

		// ensure 50% of a GPU is still 1 gpu
		assert.EqualValues(t, 1, out.GPU)
		assert.EqualValues(t, []models.GPU{{}}, out.GPUs)
	})

	t.Run("scaling 2 GPU to 1% results in 1 GPU unit", func(t *testing.T) {
		systemCapacity := models.Resources{
			CPU:    10,
			Memory: 10,
			Disk:   10,
			GPU:    2,
		}

		scaler := ResourceScaler{
			CPU:    "80%",
			Memory: "70%",
			Disk:   "60%",
			GPU:    "1%",
		}

		out, err := scaler.ToResource(systemCapacity)
		require.NoError(t, err)

		// ensure 50% of a GPU is still 1 gpu
		assert.EqualValues(t, 1, out.GPU)
		assert.EqualValues(t, []models.GPU{{}}, out.GPUs)
	})

	t.Run("scaling 4 GPU to 75% results in 3 GPU unit", func(t *testing.T) {
		systemCapacity := models.Resources{
			CPU:    10,
			Memory: 10,
			Disk:   10,
			GPU:    4,
		}

		scaler := ResourceScaler{
			CPU:    "80%",
			Memory: "70%",
			Disk:   "60%",
			GPU:    "75%",
		}

		out, err := scaler.ToResource(systemCapacity)
		require.NoError(t, err)

		// ensure 50% of a GPU is still 1 gpu
		assert.EqualValues(t, 3, out.GPU)
		assert.EqualValues(t, []models.GPU{{}, {}, {}}, out.GPUs)
	})

	t.Run("scaling defined GPU types", func(t *testing.T) {
		systemCapacity := models.Resources{
			CPU:    10,
			Memory: 10,
			Disk:   10,
			GPU:    3,
			GPUs: []models.GPU{
				{Vendor: "nvidia"},
				{Vendor: "amd"},
				{Vendor: "intel"},
			},
		}

		scaler := ResourceScaler{
			CPU:    "80%",
			Memory: "70%",
			Disk:   "60%",
			GPU:    "70%",
		}

		out, err := scaler.ToResource(systemCapacity)
		require.NoError(t, err)

		// ensure 70% of a GPU is 2 gpu
		assert.EqualValues(t, 2, out.GPU)
		assert.EqualValues(t, []models.GPU{
			{Vendor: "nvidia"},
			{Vendor: "amd"},
		}, out.GPUs)
	})

}
