package test

import (
	"testing"

	_ "github.com/filecoin-project/bacalhau/pkg/logger"
)

func TestHasCidLocally(t *testing.T) {

	stack, cancelFunction := SetupTest(
		t,
		3,
		0,
	)

	defer TeardownTest(stack, cancelFunction)

}
