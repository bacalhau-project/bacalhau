package version

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/pkg/errors"
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
) (*UpdateCheckResponse, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse URL for update host")
	}

	q := u.Query()
	q.Set("clientVersion", currentClientVersion.GitVersion)
	if currentServerVersion.GitVersion != "" {
		q.Set("serverVersion", currentServerVersion.GitVersion)
	}
	q.Set("operatingSystem", currentClientVersion.GOOS)
	q.Set("architecture", currentClientVersion.GOARCH)
	q.Set("userID", clientID)

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
