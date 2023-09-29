package util

import (
	"context"
	"fmt"

	"github.com/Masterminds/semver"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/pkg/models"

	"github.com/bacalhau-project/bacalhau/pkg/version"
)

func CheckVersion(cmd *cobra.Command, args []string) error {
	// corba doesn't do PersistentPreRun{,E} chaining yet
	// https://github.com/spf13/cobra/issues/252
	root := cmd
	for ; root.HasParent(); root = root.Parent() {
	}
	root.PersistentPreRun(cmd, args)

	// the client will not be known until the root persisten pre run logic is executed which
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
