package v1beta1

import (
	"testing"

	"github.com/stretchr/testify/require"
)

var millicoreTestCases map[Millicores]string = map[Millicores]string{
	Millicore:        "1m",
	Core:             "1",
	100 * Millicore:  "100m",
	900 * Millicore:  "900m",
	3 * Core:         "3",
	3500 * Millicore: "3500m",
}

func TestMillicoresString(t *testing.T) {
	for input, expectedString := range millicoreTestCases {
		t.Run(expectedString, func(t *testing.T) {
			require.Equal(t, expectedString, input.String())
		})
	}
}
