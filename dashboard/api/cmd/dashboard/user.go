package dashboard

import (
	"fmt"

	"github.com/davecgh/go-spew/spew"
	"github.com/filecoin-project/bacalhau/dashboard/api/pkg/model"
	"github.com/spf13/cobra"
)

type userOptionsType struct {
	username string
	password string
}

func setupUserOptions(cmd *cobra.Command, opts *userOptionsType) {
	cmd.PersistentFlags().StringVar(
		&opts.username, "username", opts.username,
		`The username for the user.`,
	)
	cmd.PersistentFlags().StringVar(
		&opts.password, "password", opts.password,
		`The password for the user.`,
	)
}

func newUserOptions() userOptionsType {
	return userOptionsType{
		username: "",
		password: "",
	}
}

func newUserCmd() *cobra.Command {
	userCmd := &cobra.Command{
		Use:   "user",
		Short: "Commands to manage users",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return fmt.Errorf("please run a subcommand")
		},
	}
	userCmd.AddCommand(newAddUserCmd())
	userCmd.AddCommand(newSetPasswordCmd())
	return userCmd
}

func newAddUserCmd() *cobra.Command {
	modelOptions := newModelOptions()
	userOptions := newUserOptions()
	addUserCmd := &cobra.Command{
		Use:   "add",
		Short: "Add a new user",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return adduser(cmd, modelOptions, userOptions)
		},
	}
	setupModelOptions(addUserCmd, &modelOptions)
	setupUserOptions(addUserCmd, &userOptions)
	return addUserCmd
}

func newSetPasswordCmd() *cobra.Command {
	modelOptions := newModelOptions()
	userOptions := newUserOptions()
	setPasswordCmd := &cobra.Command{
		Use:   "password",
		Short: "Set the password for an existing user",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return setpassword(cmd, modelOptions, userOptions)
		},
	}
	setupModelOptions(setPasswordCmd, &modelOptions)
	setupUserOptions(setPasswordCmd, &userOptions)
	return setPasswordCmd
}

func adduser(cmd *cobra.Command, modelOptions model.ModelOptions, userOptions userOptionsType) error {
	model, err := model.NewModelAPI(modelOptions)
	if err != nil {
		return err
	}
	user, err := model.AddUser(cmd.Context(), userOptions.username, userOptions.password)
	if err != nil {
		return err
	}
	spew.Dump(user)
	return nil
}

func setpassword(cmd *cobra.Command, modelOptions model.ModelOptions, userOptions userOptionsType) error {
	model, err := model.NewModelAPI(modelOptions)
	if err != nil {
		return err
	}
	user, err := model.UpdateUserPassword(cmd.Context(), userOptions.username, userOptions.password)
	if err != nil {
		return err
	}
	spew.Dump(user)
	return nil
}
