package deprecated

import (
	"fmt"
)

const MigrationURL = "https://docs.bacalhau.org/v/v.1.4.0/references/cli-reference/command-migration"

var migrationMessageSuffix = fmt.Sprintf(`See the migration guide at %s for more information.`, MigrationURL)
