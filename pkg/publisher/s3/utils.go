package s3

import (
	"strings"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

func ParsePublishedKey(key string, execution *models.Execution, archive bool) string {
	if archive && !strings.HasSuffix(key, ".tar.gz") {
		key = key + ".tar.gz"
	}
	if !archive && !strings.HasSuffix(key, "/") {
		key = key + "/"
	}

	key = strings.ReplaceAll(key, "{nodeID}", execution.NodeID)
	key = strings.ReplaceAll(key, "{executionID}", execution.ID)
	key = strings.ReplaceAll(key, "{jobID}", execution.JobID)
	key = strings.ReplaceAll(key, "{date}", time.Now().Format("20060102"))
	key = strings.ReplaceAll(key, "{time}", time.Now().Format("150405"))
	return key
}
