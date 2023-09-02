package configflags

import (
	"github.com/bacalhau-project/bacalhau/pkg/config/configenv"
)

// Default is the default value used by flags. It will be overridden by any values in the config file.
// We can configure the default flag values by setting `Default` to a different config environment.
var Default = configenv.Production
