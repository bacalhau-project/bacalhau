package util

import (
	"context"
	"fmt"

	"github.com/Masterminds/semver"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/client"

	"github.com/bacalhau-project/bacalhau/pkg/version"
)

type Versions struct {
	ClientVersion *models.BuildVersionInfo `json:"clientVersion,omitempty"`
	ServerVersion *models.BuildVersionInfo `json:"serverVersion,omitempty"`
	LatestVersion *models.BuildVersionInfo `json:"latestVersion,omitempty"`
	UpdateMessage string                   `json:"updateMessage,omitempty"`
}

func CheckVersion(cmd *cobra.Command, args []string) error {
	// the client will not be known until the root persistent pre run logic is executed which
	// sets up the repo and config
	ctx := cmd.Context()
	client := GetAPIClient(ctx)

	// Check that the server version is compatible with the client version
	serverVersion, _ := client.Version(ctx) // Ok if this fails, version validation will skip
	if err := EnsureValidVersion(ctx, version.Get(), serverVersion); err != nil {
		return fmt.Errorf("version validation failed: %s", err)
	}

	return nil
}

func GetAllVersions(ctx context.Context) (Versions, error) {
	var err error
	versions := Versions{ClientVersion: version.Get()}
	versions.ServerVersion, err = client.NewAPIClient(config.ClientAPIHost(), config.ClientAPIPort()).Version(ctx)
	if err != nil {
		return versions, errors.Wrap(err, "error running version command")
	}

	clientID, err := config.GetClientID()
	if err != nil {
		return versions, errors.Wrap(err, "error getting client ID")
	}

	updateCheck, err := version.CheckForUpdate(ctx, versions.ClientVersion, versions.ServerVersion, clientID)
	if err != nil {
		return versions, errors.Wrap(err, "failed to get latest version")
	} else {
		versions.UpdateMessage = updateCheck.Message
		versions.LatestVersion = updateCheck.Version
	}

	return versions, nil
}

var printMessage *string = nil

// StartUpdateCheck is a Cobra pre run hook to run an update check in the
// background. There should be no output if the check fails or the context is
// cancelled before the check can complete.
func StartUpdateCheck(cmd *cobra.Command, args []string) {
	go func(ctx context.Context) {
		if skip, err := config.Get[bool](types.SkipUpdateCheck); skip || err != nil {
			log.Ctx(ctx).Debug().Err(err).Bool(types.SkipUpdateCheck, skip).Msg("Skipping update check")
			return
		}

		versions, err := GetAllVersions(ctx)
		if err == nil {
			printMessage = &versions.UpdateMessage
		}
	}(cmd.Context())
}

// PrintUpdateCheck is a Cobra post run hook to print the results of an update
// check. The message will be a non-nil pointer only if the update check
// succeeds and should only have visible output if the message is non-empty.
func PrintUpdateCheck(cmd *cobra.Command, args []string) {
	if printMessage != nil && *printMessage != "" {
		fmt.Fprintln(cmd.ErrOrStderr())
		fmt.Fprintln(cmd.ErrOrStderr(), *printMessage)
	}
}

func EnsureValidVersion(ctx context.Context, clientVersion, serverVersion *models.BuildVersionInfo) error {
	if clientVersion == nil {
		log.Ctx(ctx).Warn().Msg("Unable to parse nil client version, skipping version check")
		return nil
	}
	if clientVersion.GitVersion == version.DevelopmentGitVersion {
		log.Ctx(ctx).Debug().Msg("Development client version, skipping version check")
		return nil
	}
	if serverVersion == nil {
		log.Ctx(ctx).Warn().Msg("Unable to parse nil server version, skipping version check")
		return nil
	}
	if serverVersion.GitVersion == version.DevelopmentGitVersion {
		log.Ctx(ctx).Debug().Msg("Development server version, skipping version check")
		return nil
	}
	c, err := semver.NewVersion(clientVersion.GitVersion)
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).Msg("Unable to parse client version, skipping version check")
		return nil
	}
	s, err := semver.NewVersion(serverVersion.GitVersion)
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).Msg("Unable to parse server version, skipping version check")
		return nil
	}
	if s.GreaterThan(c) {
		return fmt.Errorf(`the server version %s is newer than client version %s, please upgrade your client with the following command:
curl -sL https://get.bacalhau.org/install.sh | bash`,
			serverVersion.GitVersion,
			clientVersion.GitVersion,
		)
	}
	if c.GreaterThan(s) {
		return fmt.Errorf(
			"client version %s is newer than server version %s, please ask your network administrator to update Bacalhau",
			clientVersion.GitVersion,
			serverVersion.GitVersion,
		)
	}
	return nil
}
