package configflags

import (
	"fmt"
)

const FeatureDeprecatedMessage = "This feature has been deprecated and is no longer " +
	"functional. The flag has no effect and can be safely removed."

func makeDeprecationMessage(key string) string {
	return fmt.Sprintf("Use %s to set this configuration", makeConfigFlagDeprecationCommand(key))
}

func makeConfigFlagDeprecationCommand(key string) string {
	return fmt.Sprintf("--config %s=<value>", key)
}
