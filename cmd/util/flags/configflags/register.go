package configflags

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/bacalhau-project/bacalhau/cmd/util/flags"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/models"
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
	Deprecated           bool
	DeprecatedMessage    string
}

func BindFlags(v *viper.Viper, register map[string][]Definition) error {
	seen := make(map[string]Definition)
	for _, defs := range register {
		for _, def := range defs {
			// sanity check to ensure we are not binding a config key on more than one flag.
			if dup, ok := seen[def.ConfigPath]; ok && !def.Deprecated {
				return fmt.Errorf("DEVELOPER ERROR: duplicate registration of config key %s for flag %s"+
					" previously registered on on flag %s", def.ConfigPath, def.FlagName, dup.FlagName)
			}
			if !def.Deprecated {
				seen[def.ConfigPath] = def
			}
			flagDefs := viper.Get(cliflags.RootCommandConfigFlags)
			if flagDefs == nil {
				flagDefs = make([]Definition, 0)
			}
			flagsConfigs := flagDefs.([]Definition)
			flagsConfigs = append(flagsConfigs, def)
			v.Set(cliflags.RootCommandConfigFlags, flagsConfigs)
		}
	}
	return nil
}

// PreRun returns a run hook that binds the passed flag sets onto the command.
func PreRun(v *viper.Viper, flags map[string][]Definition) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		err := BindFlags(v, flags)
		if err != nil {
			return err
		}
		return err
	}
}

// RegisterFlags adds flags to the command based on provided definitions.
// This method should be called before the command runs to register flags accordingly.
func RegisterFlags(cmd *cobra.Command, register map[string][]Definition) error {
	for name, defs := range register {
		flagSet := pflag.NewFlagSet(name, pflag.ContinueOnError)
		// Determine the type of the default value
		for _, def := range defs {
			switch v := def.DefaultValue.(type) {
			case int:
				flagSet.Int(def.FlagName, v, def.Description)
			case uint64:
				flagSet.Uint64(def.FlagName, v, def.Description)
			case bool:
				flagSet.Bool(def.FlagName, v, def.Description)
			case string:
				flagSet.String(def.FlagName, v, def.Description)
			case []string:
				flagSet.StringSlice(def.FlagName, v, def.Description)
			case map[string]string:
				flagSet.StringToString(def.FlagName, v, def.Description)
			case models.JobSelectionDataLocality:
				flagSet.Var(flags.DataLocalityFlag(&v), def.FlagName, def.Description)
			case logger.LogMode:
				flagSet.Var(flags.LoggingFlag(&v), def.FlagName, def.Description)
			case time.Duration:
				flagSet.DurationVar(&v, def.FlagName, v, def.Description)
			case types.Duration:
				flagSet.DurationVar((*time.Duration)(&v), def.FlagName, time.Duration(v), def.Description)
			case types.ResourceType:
				flagSet.String(def.FlagName, string(v), def.Description)
			default:
				return fmt.Errorf("unhandled type: %T for flag %s", v, def.FlagName)
			}

			if def.Deprecated {
				flag := flagSet.Lookup(def.FlagName)
				flag.Deprecated = def.DeprecatedMessage
				flag.Hidden = true
			}
		}
		cmd.PersistentFlags().AddFlagSet(flagSet)
	}
	return nil
}
