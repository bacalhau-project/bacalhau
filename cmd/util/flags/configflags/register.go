package configflags

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/bacalhau-project/bacalhau/cmd/util/flags"
	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/semantic"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/model"
)

// Definition serves as a bridge between Cobra's command-line flags
// and Viper's configuration management. Each instance of `Definition` maps a flag
// (as presented to the user via the CLI) to its corresponding configuration setting
// in Viper. Here's a breakdown:
//   - FlagName: The name of the flag as it appears on the command line.
//   - ConfigPath: The path/key used by Viper to store and retrieve the flag's value.
//     This path can represent nested configuration settings. It is also the environment variable (replace '.' with '_')
//   - DefaultValue: The default value for the flag, used both in Cobra (when the flag
//     is not explicitly provided) and in Viper (as the initial configuration value).
//   - Description: A human-readable description of the flag's purpose, shown in help
//     messages and documentation.
//
// By defining flags in this manner, we establish a clear and consistent pattern for
// integrating Cobra and Viper, ensuring that command-line interactions seamlessly
// reflect and influence the underlying configuration state.
type Definition struct {
	FlagName             string
	ConfigPath           string
	DefaultValue         interface{}
	Description          string
	EnvironmentVariables []string
}

// BindFlags binds flags from a command to Viper using the provided definitions.
// This method should be called in command `PreRun`
func BindFlags(cmd *cobra.Command, register map[string][]Definition) error {
	for _, defs := range register {
		for _, def := range defs {
			// set the default value
			viper.SetDefault(def.ConfigPath, def.DefaultValue)

			// bind the flag to viper
			if err := viper.BindPFlag(def.ConfigPath, cmd.Flags().Lookup(def.FlagName)); err != nil {
				return err
			}

			// Bind environment variables to viper
			if len(def.EnvironmentVariables) > 0 {
				if err := viper.BindEnv(append([]string{def.ConfigPath}, def.EnvironmentVariables...)...); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// RegisterFlags adds flags to the command based on provided definitions.
// This method should be called before the command runs to register flags accordingly.
func RegisterFlags(cmd *cobra.Command, register map[string][]Definition) error {
	for name, defs := range register {
		fset := pflag.NewFlagSet(name, pflag.ContinueOnError)
		// Determine the type of the default value
		for _, def := range defs {
			switch v := def.DefaultValue.(type) {
			case int:
				fset.Int(def.FlagName, v, def.Description)
			case bool:
				fset.Bool(def.FlagName, v, def.Description)
			case string:
				fset.String(def.FlagName, v, def.Description)
			case []string:
				fset.StringSlice(def.FlagName, v, def.Description)
			case map[string]string:
				fset.StringToString(def.FlagName, v, def.Description)
			case model.JobSelectionDataLocality:
				fset.Var(flags.DataLocalityFlag((*semantic.JobSelectionDataLocality)(&v)), def.FlagName, def.Description)
			case logger.LogMode:
				fset.Var(flags.LoggingFlag(&v), def.FlagName, def.Description)
			case time.Duration:
				fset.DurationVar(&v, def.FlagName, v, def.Description)
			case types.Duration:
				fset.DurationVar((*time.Duration)(&v), def.FlagName, time.Duration(v), def.Description)
			case types.StorageType:
				fset.Var(flags.StorageTypeFlag(&v), def.FlagName, def.Description)
			default:
				return fmt.Errorf("unhandled type: %T", v)
			}
		}
		cmd.PersistentFlags().AddFlagSet(fset)
	}
	return nil
}
