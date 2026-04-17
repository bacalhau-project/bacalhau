//go:build unit || !integration

package marshaller

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnmarshalJob(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		var jobWithUnknownFields = `
{
  "Name": "A Simple Docker Job",
  "Type": "batch",
  "Count": 1,
  "Tasks": [
    {
      "Name": "My main task",
      "Engine": {
        "Type": "docker",
        "Params": {
          "Image": "busybox:1.37.0",
          "Entrypoint": [
            "/bin/sh"
          ],
          "Parameters": [
            "-c",
            "echo Hello Bacalhau!"
          ]
        }
      }
    }
  ]
}
`
		j, err := UnmarshalJob([]byte(jobWithUnknownFields))
		assert.NotNil(t, j)
		assert.NoError(t, err)
	})
	t.Run("job with unknown field", func(t *testing.T) {
		var jobWithUnknownFields = `
{
  "Name": "A Simple Docker Job",
  "Type": "batch",
  "Selector": "nope",
  "Count": 1,
  "Tasks": [
    {
      "Name": "My main task",
      "Engine": {
        "Type": "docker",
        "Params": {
          "Image": "busybox:1.37.0",
          "Entrypoint": [
            "/bin/sh"
          ],
          "Parameters": [
            "-c",
            "echo Hello Bacalhau!"
          ]
        }
      }
    }
  ]
}
`
		j, err := UnmarshalJob([]byte(jobWithUnknownFields))
		assert.Nil(t, j)
		assert.Error(t, err)
		assert.ErrorContains(t, err, "unknown field: 'Selector' in 'Job'")
	})
	t.Run("job with un-settable field", func(t *testing.T) {
		var jobWithUnknownFields = `
{
  "Name": "A Simple Docker Job",
  "Type": "batch",
  "Version": 1,
  "Count": 1,
  "Tasks": [
    {
      "Name": "My main task",
      "Engine": {
        "Type": "docker",
        "Params": {
          "Image": "busybox:1.37.0",
          "Entrypoint": [
            "/bin/sh"
          ],
          "Parameters": [
            "-c",
            "echo Hello Bacalhau!"
          ]
        }
      }
    }
  ]
}
`
		j, err := UnmarshalJob([]byte(jobWithUnknownFields))
		assert.Nil(t, j)
		assert.Error(t, err)
		assert.ErrorContains(t, err, "field: 'Version' in 'Job' is not allowed")
	})
	t.Run("job with invalid field type field", func(t *testing.T) {
		var jobWithUnknownFields = `
{
  "Name": "A Simple Docker Job",
  "Type": "batch",
  "Count": "1",
  "Tasks": [
    {
      "Name": "My main task",
      "Engine": {
        "Type": "docker",
        "Params": {
          "Image": "busybox:1.37.0",
          "Entrypoint": [
            "/bin/sh"
          ],
          "Parameters": [
            "-c",
            "echo Hello Bacalhau!"
          ]
        }
      }
    }
  ]
}
`
		j, err := UnmarshalJob([]byte(jobWithUnknownFields))
		assert.Nil(t, j)
		assert.Error(t, err)
		assert.ErrorContains(t, err, "field: 'Count' in 'job' is invalid type: 'string' expected: 'int'")
	})

}
