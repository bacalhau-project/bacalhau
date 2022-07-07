package system

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCleanupManager(t *testing.T) {
	clean := false

	cm := NewCleanupManager()
	cm.RegisterCallback(func() error {
		clean = true
		return nil
	})

	cm.Cleanup()
	require.True(t, clean, "cleanup handler failed to run registered functions")
}
