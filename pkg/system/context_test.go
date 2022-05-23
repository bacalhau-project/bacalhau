package system

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCancelContext(t *testing.T) {
	seenHandler := false
	cancelContext := GetCancelContext()
	cancelContext.AddShutdownHandler(func() {
		seenHandler = true
	})
	cancelContext.Stop()
	assert.True(t, seenHandler, "cancel context handler not called")
}
