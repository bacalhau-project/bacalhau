package flags

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/model"
)

type FlagDefinition struct {
	FlagName     string
	ConfigPath   string
	DefaultValue interface{}
	Description  string
}

// Generic function to define a flag, set a default value, and bind it to Viper
func RegisterFlags(cmd *cobra.Command, register map[string][]FlagDefinition) error {
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
				fset.Var(DataLocalityFlag(&v), def.FlagName, def.Description)
			case logger.LogMode:
				fset.Var(LoggingFlag(&v), def.FlagName, def.Description)
			default:
				return fmt.Errorf("unhandled type: %T", v)
			}
			viper.SetDefault(def.ConfigPath, def.DefaultValue)
			if err := viper.BindPFlag(def.ConfigPath, fset.Lookup(def.FlagName)); err != nil {
				return err
			}
		}
		cmd.PersistentFlags().AddFlagSet(fset)
	}
	return nil
}
