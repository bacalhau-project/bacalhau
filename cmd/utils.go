package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Masterminds/semver"
	"github.com/rs/zerolog/log"
	"github.com/spf13/pflag"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/requester/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/version"
)

const (
	JSONFormat                         string = "json"
	YAMLFormat                         string = "yaml"
	DefaultDockerRunWaitSeconds               = 600
	PrintoutCanceledButRunningNormally string = "printout canceled but running normally"
	// AutoDownloadFolderPerm is what permissions we give to a folder we create when downloading results
	AutoDownloadFolderPerm       = 0755
	HowFrequentlyToUpdateTicker  = 50 * time.Millisecond
	DefaultSpinnerFormatDuration = 30 * time.Millisecond
	DefaultTimeout               = 30 * time.Minute
)

var eventsWorthPrinting = map[model.JobStateType]eventStruct{
	model.JobStateNew:        {Message: "Creating job for submission", IsTerminal: false, PrintDownload: false, IsError: false},
	model.JobStateQueued:     {Message: "Job waiting to be scheduled", PrintDownload: true, IsTerminal: false, IsError: false},
	model.JobStateInProgress: {Message: "Job in progress", PrintDownload: true, IsTerminal: false, IsError: false},
	model.JobStateError:      {Message: "Error while executing the job", PrintDownload: false, IsTerminal: true, IsError: true},
	model.JobStateCancelled:  {Message: "Job canceled", PrintDownload: false, IsTerminal: true, IsError: false},
	model.JobStateCompleted:  {Message: "Job finished", PrintDownload: false, IsTerminal: true, IsError: false},
	model.JobStateCompletedPartially: {
		Message:       "Job partially complete",
		PrintDownload: false,
		IsTerminal:    true,
		IsError:       true,
	},
}

type eventStruct struct {
	Message       string
	IsTerminal    bool
	PrintDownload bool
	IsError       bool
}

func shortenTime(outputWide bool, t time.Time) string { //nolint:unused // Useful function, holding here
	if outputWide {
		return t.Format("06-01-02-15:04:05")
	}

	return t.Format("15:04:05")
}

var DefaultShortenStringLength = 20

func shortenString(outputWide bool, st string) string {
	if outputWide {
		return st
	}

	if len(st) < DefaultShortenStringLength {
		return st
	}

	return st[:20] + "..."
}

func shortID(outputWide bool, id string) string {
	if outputWide {
		return id
	}
	if len(id) < model.ShortIDLength {
		return id
	}
	return id[:model.ShortIDLength]
}

func GetAPIHostAndPort() string {
	return fmt.Sprintf("%s:%d", apiHost, apiPort)
}

func GetAPIClient() *publicapi.RequesterAPIClient {
	return publicapi.NewRequesterAPIClient(apiHost, apiPort)
}

// ensureValidVersion checks that the server version is the same or less than the client version
func EnsureValidVersion(ctx context.Context, clientVersion, serverVersion *model.BuildVersionInfo) error {
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

func NewIPFSDownloadFlags(settings *model.DownloaderSettings) *pflag.FlagSet {
	flags := pflag.NewFlagSet("IPFS Download flags", pflag.ContinueOnError)
	flags.BoolVar(&settings.Raw, "raw",
		settings.Raw, "Download raw result CIDs instead of merging multiple CIDs into a single result")
	flags.DurationVar(&settings.Timeout, "download-timeout-secs",
		settings.Timeout, "Timeout duration for IPFS downloads.")
	flags.StringVar(&settings.OutputDir, "output-dir",
		settings.OutputDir, "Directory to write the output to.")
	flags.StringVar(&settings.IPFSSwarmAddrs, "ipfs-swarm-addrs",
		settings.IPFSSwarmAddrs, "Comma-separated list of IPFS nodes to connect to.")
	return flags
}

func getDefaultJobFolder(jobID string) string {
	return fmt.Sprintf("job-%s", system.GetShortID(jobID))
}

// if the user does not supply a value for "download results to here"
// then we default to making a folder in the current directory
func ensureDefaultDownloadLocation(jobID string) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	downloadDir := filepath.Join(cwd, getDefaultJobFolder(jobID))
	err = os.MkdirAll(downloadDir, AutoDownloadFolderPerm)
	if err != nil {
		return "", err
	}
	return downloadDir, nil
}

func processDownloadSettings(settings model.DownloaderSettings, jobID string) (model.DownloaderSettings, error) {
	if settings.OutputDir == "" {
		dir, err := ensureDefaultDownloadLocation(jobID)
		if err != nil {
			return settings, err
		}
		settings.OutputDir = dir
	}
	return settings, nil
}

type RunTimeSettings struct {
	AutoDownloadResults   bool // Automatically download the results after finishing
	IPFSGetTimeOut        int  // Timeout for IPFS in seconds
	IsLocal               bool // Job should be executed locally
	WaitForJobToFinish    bool // Wait for the job to finish before returning
	WaitForJobTimeoutSecs int  // Timeout for waiting for the job to finish
	PrintJobIDOnly        bool // Only print the Job ID as output
	PrintNodeDetails      bool // Print the node details as output
	Follow                bool // Follow along with the output of the job
	SkipSyntaxChecking    bool // Skip having 'shellchecker' verify syntax of the command.
	DryRun                bool // iff true do not submit the job, but instead print out what will be submitted.
}

func NewRunTimeSettings() *RunTimeSettings {
	return &RunTimeSettings{
		AutoDownloadResults:   false,
		WaitForJobToFinish:    true,
		WaitForJobTimeoutSecs: DefaultDockerRunWaitSeconds,
		IPFSGetTimeOut:        10,
		IsLocal:               false,
		PrintJobIDOnly:        false,
		PrintNodeDetails:      false,
		Follow:                false,
		SkipSyntaxChecking:    false,
		DryRun:                false,
	}
}

func NewRunTimeSettingsFlags(settings *RunTimeSettings) *pflag.FlagSet {
	flags := pflag.NewFlagSet("Runtime settings", pflag.ContinueOnError)
	flags.IntVarP(&settings.IPFSGetTimeOut, "gettimeout", "g", settings.IPFSGetTimeOut,
		`Timeout for getting the results of a job in --wait`)
	flags.BoolVar(&settings.IsLocal, "local", settings.IsLocal,
		`Run the job locally. Docker is required`)
	flags.BoolVar(&settings.WaitForJobToFinish, "wait", settings.WaitForJobToFinish,
		`Wait for the job to finish.`)
	flags.IntVar(&settings.WaitForJobTimeoutSecs, "wait-timeout-secs", settings.WaitForJobTimeoutSecs,
		`When using --wait, how many seconds to wait for the job to complete before giving up.`)
	flags.BoolVar(&settings.PrintJobIDOnly, "id-only", settings.PrintJobIDOnly,
		`Print out only the Job ID on successful submission.`)
	flags.BoolVar(&settings.PrintNodeDetails, "node-details", settings.PrintNodeDetails,
		`Print out details of all nodes (overridden by --id-only).`)
	flags.BoolVar(&settings.AutoDownloadResults, "download", settings.AutoDownloadResults,
		`Should we download the results once the job is complete?`)
	flags.BoolVarP(&settings.Follow, "follow", "f", settings.Follow,
		`When specified will follow the output from the job as it runs`)
	flags.BoolVar(
		&settings.SkipSyntaxChecking, "skip-syntax-checking", settings.SkipSyntaxChecking,
		`Skip having 'shellchecker' verify syntax of the command`)
	flags.BoolVar(
		&settings.DryRun, "dry-run", settings.DryRun,
		`Do not submit the job, but instead print out what will be submitted`)

	return flags
}

func GetCommandLineExecutable() string {
	return os.Args[0]
}

// DockerImageContainsTag checks if the image contains a tag or a digest
func DockerImageContainsTag(image string) bool {
	if strings.Contains(image, ":") {
		return true
	}
	if strings.Contains(image, "@") {
		return true
	}
	return false
}
