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

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

const endpoint = "http://update.bacalhau.org/version"

type UpdateCheckResponse struct {
	Version *models.BuildVersionInfo `json:"version"`
	Message string                   `json:"message"`
}

func CheckForUpdate(
	ctx context.Context,
	currentClientVersion, currentServerVersion *models.BuildVersionInfo,
	clientID string,
	InstallationID string,
) (*UpdateCheckResponse, error) {
	u, err := url.Parse(endpoint)
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
	q.Set("clientID", clientID)
	q.Set("InstallationID", InstallationID)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build HTTP request for update check")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch the latest version from the server")
	}
	defer resp.Body.Close()

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

// RunUpdateChecker starts a goroutine that will periodically make an update
// check according to the configured update frequency and check skipping. The
// goroutine is ended by canceling the passed context. `getServerVersion` is
// allowed to return a nil version if there is no server to communicate with
// (e.g. because the node running the update check is the server).
func RunUpdateChecker(
	ctx context.Context,
	getServerVersion func(context.Context) (*models.BuildVersionInfo, error),
	responseCallback func(context.Context, *UpdateCheckResponse),
) {
	updateFrequency := config.GetUpdateCheckFrequency()
	if updateFrequency <= 0 {
		log.Ctx(ctx).Warn().Dur(types.UpdateCheckFrequency, updateFrequency).Msg("Update frequency is zero or less so no update checks will run")
		return
	}

	clientVersion := Get()
	clientID, err := config.GetClientID()
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Failed to read client ID")
		return
	}
	userID, err := config.GetInstallationUserID()
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Failed to read user ID")
		return
	}

	runUpdateCheck := func() {
		// Check this every time, to handle programmatic changes to config that
		// may switch this on or off.
		if skip, err := config.Get[bool](types.UpdateSkipChecks); skip || err != nil {
			log.Ctx(ctx).Debug().Err(err).Bool(types.UpdateSkipChecks, skip).Msg("Skipping update check due to config")
			return
		}

		// The server may update itself between checks, so always ask the server
		// for its current version.
		serverVersion, err := getServerVersion(ctx)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("Failed to read server version")
			serverVersion = nil
		}

		updateResponse, err := CheckForUpdate(ctx, clientVersion, serverVersion, clientID, userID)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("Failed to perform update check")
		}

		if err == nil {
			responseCallback(ctx, updateResponse)
			err = writeNewLastCheckTime()
			log.Ctx(ctx).WithLevel(logger.ErrOrDebug(err)).Err(err).Msg("Completed update check")
		}
	}

	lastCheck, err := readLastCheckTime()
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).Msg("Error reading update check state – will perform check anyway")
		lastCheck = time.UnixMilli(0)
	}

	// Count down the remaining time between the last check and the next check,
	// and then reset the ticker to start doing regular periodic checks. This is fine
	// because the ticker will not have fired before the initial timer.
	initialPeriod := time.Until(lastCheck.Add(updateFrequency))
	initialTimer := time.NewTimer(initialPeriod)
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

type updateState struct {
	LastCheck time.Time
}

func readLastCheckTime() (time.Time, error) {
	path, err := config.Get[string](types.UpdateCheckStatePath)
	if err != nil {
		return time.Now(), errors.Wrap(err, "error getting repo path")
	}

	file, err := os.Open(path)
	if err != nil {
		return time.Now(), errors.Wrap(err, "error opening update state file")
	}
	defer file.Close()

	var state updateState
	err = json.NewDecoder(file).Decode(&state)
	if err != nil {
		return time.Now(), errors.Wrap(err, "error reading update state")
	}

	return state.LastCheck, nil
}

func writeNewLastCheckTime() error {
	path, err := config.Get[string](types.UpdateCheckStatePath)
	if err != nil {
		return errors.Wrap(err, "error getting repo path")
	}

	file, err := os.Create(path)
	if err != nil {
		return errors.Wrap(err, "error creating update state file")
	}
	defer file.Close()

	state := updateState{LastCheck: time.Now()}
	err = json.NewEncoder(file).Encode(&state)
	if err != nil {
		return errors.Wrap(err, "error writing update state")
	}

	return nil
}
