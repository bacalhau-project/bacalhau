package system

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCleanupManager(t *testing.T) {
	clean := false

	cm := NewCleanupManager()
	cm.RegisterCallback(func() error {
		clean = true
		return nil
	})

	cm.Cleanup()
	assert.True(t, clean, "cleanup handler failed to run registered functions")
}
