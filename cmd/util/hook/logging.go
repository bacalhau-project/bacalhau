package hook

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// ApplyPorcelainLogLevel sets the log level of loggers running on user-facing
// "porcelain" commands to be zerolog.FatalLevel to reduce noise shown to users,
// unless the user has specifically set a valid log level with an env var.
func ApplyPorcelainLogLevel(cmd *cobra.Command, _ []string) {
	if lvl, err := zerolog.ParseLevel(os.Getenv("LOG_LEVEL")); lvl != zerolog.NoLevel && err == nil {
		return
	}

	ctx := cmd.Context()
	ctx = log.Ctx(ctx).Level(zerolog.FatalLevel).WithContext(ctx)
	cmd.SetContext(ctx)
}
