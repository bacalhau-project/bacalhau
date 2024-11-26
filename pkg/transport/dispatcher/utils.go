package dispatcher

import (
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
)

// generateMsgID creates an idempotent unique message ID for the given event
// that can be used to deduplicate messages.
func generateMsgID(event watcher.Event) string {
	return fmt.Sprintf("seq-%d", event.SeqNum)
}
