package devstack

import (
	"fmt"
	"os"
	"testing"

	_ "github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/stretchr/testify/require"
)

func prepareFolderWithFiles(t *testing.T, fileCount int) string { //nolint:unused
	basePath := t.TempDir()
	for i := 0; i < fileCount; i++ {
		err := os.WriteFile(
			fmt.Sprintf("%s/%d.txt", basePath, i),
			[]byte(fmt.Sprintf("hello %d", i)),
			os.ModePerm,
		)
		require.NoError(t, err)
	}
	return basePath
}
