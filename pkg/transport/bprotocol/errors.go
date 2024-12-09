package bprotocol

import (
	"fmt"
)

// ErrUpgradeAvailable indicates that the orchestrator supports the NCLv1 protocol
var ErrUpgradeAvailable = fmt.Errorf("node supports NCLv1 protocol - legacy protocol disabled")
