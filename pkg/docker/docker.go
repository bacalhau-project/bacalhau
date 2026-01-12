package docker

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/registry"
	dockerclient "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	pkgerrors "github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.ptx.dk/multierrgroup"
	"golang.org/x/exp/slices"

	"github.com/bacalhau-project/bacalhau/pkg/docker/tracing"
	"github.com/bacalhau-project/bacalhau/pkg/util/closer"
)

const unknownPlatform = "unknown"

type Client struct {
	tracing.TracedClient
	hostPlatform v1.Platform
	platformSet  bool
	platformMu   sync.Mutex
	credentials  Credentials
}

func NewDockerClient() (*Client, error) {
	client, err := tracing.NewTracedClient()
	if err != nil {
		return nil, NewDockerError(err)
	}
	return &Client{
		TracedClient: client,
		credentials:  GetDockerCredentials(),
	}, nil
}

func (c *Client) IsInstalled(ctx context.Context) bool {
	_, err := c.Info(ctx)
	return err == nil
}

func (c *Client) HostGatewayIP(ctx context.Context) (net.IP, error) {
	response, err := c.NetworkInspect(ctx, "bridge", network.InspectOptions{})
	if err != nil {
		return net.IP{}, NewDockerError(err)
	}
	if configs := response.IPAM.Config; len(configs) < 1 {
		return net.IP{}, NewCustomDockerError(BridgeNetworkUnattached, "bridge network unattached")
	} else {
		return net.ParseIP(configs[0].Gateway), nil
	}
}

func (c *Client) removeContainers(ctx context.Context, filterz filters.Args) error {
	containers, err := c.ContainerList(ctx, container.ListOptions{All: true, Filters: filterz})
	if err != nil {
		return NewDockerError(err)
	}

	wg := multierrgroup.Group{}
	for _, cont := range containers {
		containerID := cont.ID
		wg.Go(func() error {
			return c.RemoveContainer(ctx, containerID)
		})
	}
	return wg.Wait()
}

func (c *Client) removeNetworks(ctx context.Context, filterz filters.Args) error {
	networks, err := c.NetworkList(ctx, network.ListOptions{Filters: filterz})
	if err != nil {
		return NewDockerError(err)
	}

	wg := multierrgroup.Group{}
	for _, n := range networks {
		networkID := n.ID
		wg.Go(func() error {
			log.Ctx(ctx).Debug().Str("Network", networkID).Msg("Network Stop")
			return c.NetworkRemove(ctx, networkID)
		})
	}
	return wg.Wait()
}

func (c *Client) RemoveObjectsWithLabel(ctx context.Context, labelName, labelValue string) error {
	filterz := filters.NewArgs(
		filters.Arg("label", fmt.Sprintf("%s=%s", labelName, labelValue)),
	)

	containerErr := c.removeContainers(ctx, filterz)
	networkErr := c.removeNetworks(ctx, filterz)
	return errors.Join(containerErr, networkErr)
}

func (c *Client) FindContainer(ctx context.Context, label string, value string) (string, error) {
	containers, err := c.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return "", NewDockerError(err)
	}

	for _, ctr := range containers {
		if ctr.Labels[label] == value {
			return ctr.ID, nil
		}
	}

	return "", NewCustomDockerError(ContainerNotFound, fmt.Sprintf("unable to find container for %s=%s", label, value))
}

func (c *Client) FollowLogs(ctx context.Context, id string) (stdout, stderr io.Reader, err error) {
	cont, err := c.ContainerInspect(ctx, id)
	if err != nil {
		return nil, nil, NewDockerError(err)
	}

	logOptions := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
	}

	ctx = log.Ctx(ctx).With().Str("ContainerID", cont.ID).Str("Image", cont.Image).Logger().WithContext(ctx)
	logsReader, err := c.ContainerLogs(ctx, cont.ID, logOptions)
	if err != nil {
		return nil, nil, NewDockerError(err)
	}

	stdoutReader, stdoutWriter := io.Pipe()
	stderrReader, stderrWriter := io.Pipe()
	go func() {
		stdoutBuffer := bufio.NewWriter(stdoutWriter)
		stderrBuffer := bufio.NewWriter(stderrWriter)
		defer closer.CloseWithLogOnError("stderrWriter", stderrWriter)
		defer closer.CloseWithLogOnError("stdoutWriter", stdoutWriter)
		defer stderrBuffer.Flush()
		defer stdoutBuffer.Flush()
		defer closer.CloseWithLogOnError("logsReader", logsReader)

		_, err = stdcopy.StdCopy(stdoutBuffer, stderrBuffer, logsReader)
		if err != nil && !errors.Is(err, context.Canceled) {
			log.Ctx(ctx).Err(err).Msg("error reading container logs")
		}
	}()

	return stdoutReader, stderrReader, nil
}

func (c *Client) GetOutputStream(
	ctx context.Context,
	containerID string,
	since *time.Time,
	follow, timestamps bool,
) (io.ReadCloser, error) {
	cont, err := c.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, NewDockerError(err)
	}

	// As long as the container exists we can get logs from it

	logOptions := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     follow,
		Timestamps: timestamps,
	}
	if since != nil {
		logOptions.Since = since.Format(time.RFC3339Nano)
	}

	ctx = log.Ctx(ctx).With().Str("ContainerID", cont.ID).Str("Image", cont.Image).Logger().WithContext(ctx)
	logsReader, err := c.ContainerLogs(ctx, cont.ID, logOptions)
	if err != nil {
		return nil, NewDockerError(err)
	}

	return logsReader, nil
}

func (c *Client) RemoveContainer(ctx context.Context, id string) error {
	log.Ctx(ctx).Debug().Str("id", id).Msgf("Container Stop")
	// ContainerRemove kills and removes a container from the docker host.
	err := c.ContainerRemove(ctx, id, container.RemoveOptions{
		RemoveVolumes: true,
		Force:         true,
	})
	if err != nil {
		return pkgerrors.WithStack(err)
	}
	return nil
}

func (c *Client) getHostPlatform(ctx context.Context) (v1.Platform, error) {
	c.platformMu.Lock()
	defer c.platformMu.Unlock()

	// Return cached platform if available
	if c.platformSet {
		return c.hostPlatform, nil
	}

	// Try to get the platform info
	version, err := c.ServerVersion(ctx)
	if err != nil {
		return v1.Platform{}, fmt.Errorf("failed to get host platform: %w", err)
	}

	// Find the Engine component
	engineIdx := slices.IndexFunc(version.Components, func(v types.ComponentVersion) bool {
		return v.Name == "Engine"
	})
	if engineIdx == -1 {
		return v1.Platform{}, fmt.Errorf("docker engine component not found")
	}

	// Extract platform details
	engine := version.Components[engineIdx].Details
	if engine["Os"] == "" || engine["Arch"] == "" {
		return v1.Platform{}, fmt.Errorf("incomplete platform information from docker engine")
	}

	// Note that 'Os' is linux on Darwin/Windows platforms that are running Linux VMs
	c.hostPlatform = v1.Platform{
		Architecture: engine["Arch"],
		OS:           engine["Os"],
	}
	c.platformSet = true
	return c.hostPlatform, nil
}

func (c *Client) SupportedPlatforms(ctx context.Context) ([]v1.Platform, error) {
	platform, err := c.getHostPlatform(ctx)
	if err != nil {
		return nil, err
	}
	return []v1.Platform{platform}, nil
}

func (c *Client) isPlatformCompatible(info types.ImageInspect, hostPlatform v1.Platform) bool { //nolint:staticcheck // TODO: migrate to image.InspectResponse
	// If any fields are "unknown", the platform info is not reliable
	if info.Os == unknownPlatform || info.Architecture == unknownPlatform {
		log.Debug().
			Str("image_os", info.Os).
			Str("image_arch", info.Architecture).
			Msg("Image has unknown platform values")
		return false
	}

	// Check if platforms match
	return info.Os == hostPlatform.OS && info.Architecture == hostPlatform.Architecture
}

// ImageDistribution fetches the details for the specified image by asking
// docker to fetch the distribution manifest from the remote registry. This
// manifest will contain information on the digest along with the details
// of the platform that the image supports.
//
// It is worth noting that if the call is made to the docker hub, the digest
// retrieved may not appear accurate when compared to the hub website but
// this is expected as the non-platform-specific digest is not displayed
// on the docker hub. This digest is safe however as both manual and
// programmatic pulls do the correct thing in retrieving the correct image
// for the platform.
//
// cf:
//   - https://github.com/moby/moby/issues/40636)
//   - https://github.com/docker/roadmap/issues/262
//
// When a docker image is available on only a single platform, the digest
// shown will be the digest pointing directly at the manifest for that image
// on that platform (as shown by the docker hub).  Where multiple platforms
// are available, the digest is pointing to a top level document describing
// all of the different platform manifests available.
//
// In either case, `docker pull` will do the correct thing and download the
// image for your platform. For example:
//
// $ docker manifest inspect  bitnami/rabbitmq@sha256:0be0d2a2 ...
//
//	"manifests": [ {
//		  "digest": "sha256:959a02013e8ab5538167f9....",
//		  "platform": { "architecture": "amd64", "os": "linux" }
//		},
//		{
//		  "digest": "sha256:11ee2c7e9e69e3a8311a19....",
//		  "platform": { "architecture": "arm64", "os": "linux"}
//		}]
//
// $ docker pull bitnami/rabbitmq@sha256:0be0d2a2 ...
// $ docker image ls
// bitnami/rabbitmq ... 48603925e10c
//
// The digest 486039 can be found in manifest sha256:11ee2c7e which is the manifest for
// the current authors machine.
//
// $ docker manifest inspect bitnami/rabbitmq@sha256:11ee2c7e
//
//	  "config": {
//		   "size": 7383,
//		   "digest": "sha256:48603925e10c01936ea4258f...."
//	  }
//
// This is the image that will finally be installed.
func (c *Client) ImageDistribution(
	ctx context.Context, image string) (*ImageManifest, error) {
	hostPlatform, err := c.getHostPlatform(ctx)
	if err != nil {
		return nil, err
	}

	// Check local image first
	info, _, err := c.ImageInspectWithRaw(ctx, image)
	if err == nil {
		// If image matches our platform and has a digest, we can trust it
		if c.isPlatformCompatible(info, hostPlatform) {
			repos := info.RepoDigests
			if len(repos) >= 1 {
				// We only want the digest part of the name, otherwise we would have
				// to go through supporting two different values in the returned
				// ImageManifest (fully qualified IDs and also just digests)
				digestParts := strings.Split(repos[0], "@")
				digest, err := digest.Parse(digestParts[1])
				if err != nil {
					return nil, NewCustomDockerError(ImageDigestMismatch, "image digest mismatch")
				}

				return &ImageManifest{
					Digest: digest,
					Platforms: []v1.Platform{
						{
							Architecture: info.Architecture,
							OS:           info.Os,
							OSVersion:    info.OsVersion,
						},
					},
				}, nil
			}
		} else {
			log.Ctx(ctx).Debug().
				Str("image", image).
				Str("local_platform", fmt.Sprintf("%s/%s", info.Os, info.Architecture)).
				Str("host_platform", platformString(hostPlatform)).
				Msg("Local image platform mismatch, checking registry")
		}
	} else if !dockerclient.IsErrNotFound(err) { //nolint:staticcheck // TODO: migrate to cerrdefs.IsNotFound
		return nil, NewDockerImageError(err, image)
	}

	// Try registry
	authToken := getAuthToken(ctx, image, c.credentials)
	dist, err := c.DistributionInspect(ctx, image, authToken)
	if err != nil {
		return nil, NewDockerImageError(err, image)
	}

	// Filter out unknown platforms
	var platforms []v1.Platform
	for _, p := range dist.Platforms {
		if p.OS != unknownPlatform && p.Architecture != unknownPlatform {
			platforms = append(platforms, p)
		}
	}

	return &ImageManifest{
		Digest:    dist.Descriptor.Digest,
		Platforms: platforms,
	}, nil
}

func (c *Client) PullImage(ctx context.Context, img string) error {
	hostPlatform, err := c.getHostPlatform(ctx)
	if err != nil {
		return err
	}

	// Check if we already have this image locally and it matches our platform
	info, _, err := c.ImageInspectWithRaw(ctx, img)
	if err == nil {
		if c.isPlatformCompatible(info, hostPlatform) {
			return nil
		}
	} else if !dockerclient.IsErrNotFound(err) { //nolint:staticcheck // TODO: migrate to cerrdefs.IsNotFound
		return NewDockerImageError(err, img)
	} else {
		log.Ctx(ctx).Debug().Str("image", img).Msg("Pulling image as it wasn't found")
	}

	// Set platform in pull options
	pullOptions := image.PullOptions{
		RegistryAuth: getAuthToken(ctx, img, c.credentials),
		Platform:     platformString(hostPlatform),
	}

	output, err := c.ImagePull(ctx, img, pullOptions)
	if err != nil {
		return NewDockerError(err)
	}

	defer closer.CloseWithLogOnError("image-pull", output)

	stop := make(chan struct{}, 1)
	defer func() {
		stop <- struct{}{}
	}()
	t := time.NewTicker(3 * time.Second)
	defer t.Stop()

	layers := &sync.Map{}
	go func() {
		for {
			select {
			case <-stop:
				return
			case <-t.C:
				logImagePullStatus(ctx, layers)
			}
		}
	}()

	dec := json.NewDecoder(output)
	for {
		var mess jsonmessage.JSONMessage
		if err := dec.Decode(&mess); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if mess.Aux != nil {
			continue
		}
		if mess.Error != nil {
			return mess.Error
		}
		layers.Store(mess.ID, mess)
	}
}

func logImagePullStatus(ctx context.Context, m *sync.Map) {
	withUnits := map[string]*zerolog.Event{}
	withoutUnits := map[string][]string{}
	m.Range(func(_, value any) bool {
		mess := value.(jsonmessage.JSONMessage)

		if mess.Progress == nil || mess.Progress.Current <= 0 {
			withoutUnits[mess.Status] = append(withoutUnits[mess.Status], mess.ID)
		} else {
			var status string
			if mess.Progress.Total <= 0 {
				status = fmt.Sprintf("%d %s", mess.Progress.Total, mess.Progress.Units)
			} else {
				status = fmt.Sprintf("%.3f%%", float64(mess.Progress.Current)/float64(mess.Progress.Total)*100) //nolint:mnd
			}

			if _, ok := withUnits[mess.Status]; !ok {
				withUnits[mess.Status] = zerolog.Dict()
			}

			withUnits[mess.Status].Str(mess.ID, status)
		}

		return true
	})
	e := log.Ctx(ctx).Debug()
	for s, l := range withUnits {
		e = e.Dict(s, l)
	}
	for s, l := range withoutUnits {
		sort.Strings(l)
		e = e.Strs(s, l)
	}

	e.Msg("Pulling layers")
}

func getAuthToken(ctx context.Context, image string, dockerCreds Credentials) string {
	if dockerCreds.IsValid() {
		// We only currently support auth for the default registry, so any
		// pulls for `image` or `user/image` should be okay, anything trying
		// to pull `repo/user/image` should not.
		if strings.Count(image, "/") < 2 {
			authConfig := registry.AuthConfig{
				Username: dockerCreds.Username,
				Password: dockerCreds.Password,
			}

			encodedJSON, err := json.Marshal(authConfig)
			if err != nil {
				log.Ctx(ctx).Err(err).Msg("failed to encode docker credentials")
			} else {
				log.Ctx(ctx).
					Info().
					Str("Image", image).
					Msg("authenticated inspect from docker registry")
				return base64.URLEncoding.EncodeToString(encodedJSON)
			}
		} else {
			log.Ctx(ctx).Info().Msg("cannot authenticate for custom registry")
		}
	}

	return ""
}

func platformString(platform v1.Platform) string {
	return fmt.Sprintf("%s/%s", platform.OS, platform.Architecture)
}

const (
	UsernameEnvVar = "DOCKER_USERNAME"
	PasswordEnvVar = "DOCKER_PASSWORD"
)

type Credentials struct {
	Username string
	Password string
}

func (d *Credentials) IsValid() bool {
	return d.Username != "" && d.Password != ""
}

func GetDockerCredentials() Credentials {
	return Credentials{
		Username: os.Getenv(UsernameEnvVar),
		Password: os.Getenv(PasswordEnvVar),
	}
}
