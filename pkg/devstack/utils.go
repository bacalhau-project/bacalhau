package devstack

import (
	"os"
)

func DevstackEnvFile() string {
	return os.Getenv("DEVSTACK_ENV_FILE")
}
