package version

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

const (
	// the default endpoint used to check for bacalhau version updates.
	defaultUpdateCheckerEndpoint = "https://get.bacalhau.org/version"
	// when this environment variable is set bacalhau will use the endpoint specified by it to check for updates.
	envVarUpdateCheckerEndpoint = "BACALHAU_UPDATE_CHECKER_ENDPOINT"
)

func getUpdateCheckerEndpoint() string {
	maybeEp := os.Getenv(envVarUpdateCheckerEndpoint)
	if maybeEp != "" {
		return maybeEp
	}
	return defaultUpdateCheckerEndpoint
}

type UpdateCheckResponse struct {
	Version *models.BuildVersionInfo `json:"version"`
	Message string                   `json:"message"`
}

func CheckForUpdate(
	ctx context.Context,
	currentClientVersion, currentServerVersion *models.BuildVersionInfo,
	instanceID string,
) (*UpdateCheckResponse, error) {
	u, err := url.Parse(getUpdateCheckerEndpoint())
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse URL for update host")
	}

	q := u.Query()
	if currentClientVersion != nil && currentClientVersion.GitVersion != "" {
		q.Set("clientVersion", currentClientVersion.GitVersion)
		q.Set("operatingSystem", currentClientVersion.GOOS)
		q.Set("architecture", currentClientVersion.GOARCH)
	}
	if currentServerVersion != nil && currentServerVersion.GitVersion != "" {
		q.Set("serverVersion", currentServerVersion.GitVersion)
	}
	if instanceID != "" {
		q.Set("instanceID", instanceID)
	}
	if installationID := system.InstallationID(); installationID != "" {
		q.Set("InstallationID", installationID)
	}

	// The BACALHAU_UPDATE_CHECKER_TEST is an env variable a user can set so that we can track
	// when the binary is being run by a non-user, to enable easier filtering of queries
	// to their update server for internal/CI.
	if os.Getenv("BACALHAU_UPDATE_CHECKER_TEST") != "" {
		q.Set("bacalhau_update_checker_test", "true")
	}

	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build HTTP request for update check")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch the latest version from the server")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("failed to fetch the latest version from the server: %s", resp.Status)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response body")
	}

	var updateCheck UpdateCheckResponse
	err = json.Unmarshal(body, &updateCheck)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal the server response when checking for updates")
	}

	return &updateCheck, nil
}

func LogUpdateResponse(ctx context.Context, ucr *UpdateCheckResponse) {
	if ucr != nil && ucr.Message != "" {
		log.Ctx(ctx).Info().Str("NewVersion", ucr.Version.GitVersion).Msg(strings.ReplaceAll(ucr.Message, "\n", " "))
	}
}

type UpdateStore interface {
	ReadLastUpdateCheck() (time.Time, error)
	WriteLastUpdateCheck(time.Time) error
	InstanceID() string
}

// RunUpdateChecker starts a goroutine that will periodically make an update
// check according to the configured update frequency and check skipping. The
// goroutine is ended by canceling the passed context. `getServerVersion` is
// allowed to return a nil version if there is no server to communicate with
// (e.g. because the node running the update check is the server).
func RunUpdateChecker(
	ctx context.Context,
	cfg types.Bacalhau,
	store UpdateStore,
	getServerVersion func(context.Context) (*models.BuildVersionInfo, error),
	responseCallback func(context.Context, *UpdateCheckResponse),
) {
	updateFrequency := time.Duration(cfg.UpdateConfig.Interval)
	if updateFrequency <= 0 {
		log.Ctx(ctx).Debug().Dur("interval", updateFrequency).Msg("Update frequency is zero or less so no update checks will run")
		return
	}

	clientVersion := Get()
	instanceID := store.InstanceID()

	runUpdateCheck := func() {
		// The server may update itself between checks, so always ask the server
		// for its current version.
		// TODO(forrest): [correctness] this should be a local only call to get the version of the binary running this code
		// otherwise I am going to get a message telling me there is a new version of bacalhau because my server
		// is out of data even when my client is up to date which is really confusing and not what we want.
		serverVersion, err := getServerVersion(ctx)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("Failed to read server version")
			serverVersion = nil
		}

		updateResponse, err := CheckForUpdate(ctx, clientVersion, serverVersion, instanceID)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("Failed to perform update check")
		}

		if err == nil {
			responseCallback(ctx, updateResponse)
			err = store.WriteLastUpdateCheck(time.Now())
			log.Ctx(ctx).WithLevel(logger.ErrOrDebug(err)).Err(err).Msg("Completed update check")
		}
	}

	lastCheck, err := store.ReadLastUpdateCheck()
	if err != nil {
		// Only log if the error is not about a missing update.json
		if !os.IsNotExist(err) {
			log.Ctx(ctx).Warn().Err(err).Msg("Error reading update check state – will perform check anyway")
		} else {
			log.Ctx(ctx).Debug().Msg("No update check state found – will perform check")
		}
		lastCheck = time.UnixMilli(0)
	}

	// Count down the remaining time between the last check and the next check,
	// and then reset the ticker to start doing regular periodic checks. This is fine
	// because the ticker will not have fired before the initial timer.
	initialPeriod := time.Until(lastCheck.Add(updateFrequency))
	// TODO(forrest) [simplify]: we can make this simpler. Use one timer else we
	//  will be checking for updates more than the configured value
	// this time ticks the last time we performed a check + the config default value
	initialTimer := time.NewTimer(initialPeriod)
	// by default this time ticks based on the config value, e.g. 24 hours
	updateTicker := time.NewTicker(updateFrequency)

	go func() {
		for {
			select {
			case <-initialTimer.C:
				runUpdateCheck()
				updateTicker.Reset(updateFrequency)
			case <-updateTicker.C:
				runUpdateCheck()
			case <-ctx.Done():
				initialTimer.Stop()
				updateTicker.Stop()
				return
			}
		}
	}()
}
