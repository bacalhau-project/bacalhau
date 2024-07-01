package deprecated

import (
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/publicapi/endpoint/requester"
)

const migrationURL = requester.MigrationGuideURL

var migrationMessageSuffix = fmt.Sprintf(`See the migration guide at %s for more information.`, migrationURL)
