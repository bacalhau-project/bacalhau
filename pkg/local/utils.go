package local

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	dockerclient "github.com/docker/docker/client"
	"github.com/filecoin-project/bacalhau/pkg/config"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	executor_util "github.com/filecoin-project/bacalhau/pkg/executor/util"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/transport/libp2p"
	"github.com/moby/moby/pkg/stdcopy"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/rs/zerolog/log"
)

// ErrContainerMarkedForRemoval indicates that the docker daemon is about to
// delete, or has already deleted, the given container.
var ErrContainerMarkedForRemoval = fmt.Errorf("docker container marked for removal")

var DefaultBootstrapAddresses = []string{
	"/ip4/35.245.115.191/tcp/1235/p2p/QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL",
	"/ip4/35.245.61.251/tcp/1235/p2p/QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF",
	"/ip4/35.245.251.239/tcp/1235/p2p/QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3",
}
var DefaultSwarmPort = 1235

func NewDockerClient() (*dockerclient.Client, error) {
	return dockerclient.NewClientWithOpts(dockerclient.FromEnv, dockerclient.WithAPIVersionNegotiation())
}

func IsInstalled(dockerClient *dockerclient.Client) bool {
	_, err := dockerClient.Info(context.Background())
	return err == nil
}

func GetContainer(dockerClient *dockerclient.Client, nameOrID string) (*types.Container, error) {
	containers, err := dockerClient.ContainerList(context.Background(), types.ContainerListOptions{
		All: true,
	})
	if err != nil {
		return nil, err
	}

	// TODO: #287 Fix if when we care about optimization of memory (224 bytes copied per loop)
	// nolint:gocritic // will fix when we care
	for _, container := range containers {
		if container.ID == nameOrID {
			return &container, nil
		}

		if container.ID[0:11] == nameOrID {
			return &container, nil
		}

		for _, containerName := range container.Names {
			if containerName == fmt.Sprintf("/%s", nameOrID) {
				return &container, nil
			}
		}
	}

	return nil, nil
}

func GetContainersWithLabel(dockerClient *dockerclient.Client, labelName, labelValue string) ([]types.Container, error) {
	results := []types.Container{}
	containers, err := dockerClient.ContainerList(context.Background(), types.ContainerListOptions{
		All: true,
	})

	if err != nil {
		return nil, err
	}
	// TODO: #287 Fix if when we care about optimization of memory (224 bytes copied per loop)
	// nolint:gocritic // will fix when we care
	for _, container := range containers {
		value, ok := container.Labels[labelName]
		if !ok {
			continue
		}
		if value == labelValue {
			results = append(results, container)
		}
	}
	return results, nil
}

func GetLogs(dockerClient *dockerclient.Client, nameOrID string) (stdout, stderr string, err error) {
	container, err := GetContainer(dockerClient, nameOrID)
	if err != nil {
		return "", "", fmt.Errorf("failed to get container: %w", err)
	}
	if container == nil {
		return "", "", fmt.Errorf("no container found: %s", nameOrID)
	}

	logsReader, err := dockerClient.ContainerLogs(context.Background(), container.ID, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	})
	if err != nil {
		// String checking is unfortunately the best we have, as errors are
		// returned by the docker server as strings, and aren't strongly typed.
		if strings.Contains(err.Error(), "can not get logs from container which is dead or marked for removal") {
			return "", "", ErrContainerMarkedForRemoval
		}

		return "", "", fmt.Errorf("failed to get container logs: %w", err)
	}

	stdoutBuffer := bytes.NewBuffer([]byte{})
	stderrBuffer := bytes.NewBuffer([]byte{})
	_, err = stdcopy.StdCopy(stdoutBuffer, stderrBuffer, logsReader)
	if err != nil {
		return "", "", err
	}

	return stdoutBuffer.String(), stderrBuffer.String(), nil
}

func RemoveContainer(dockerClient *dockerclient.Client, nameOrID string) error {
	ctx := context.Background()

	container, err := GetContainer(dockerClient, nameOrID)
	if err != nil {
		return err
	}
	if container == nil {
		return nil
	}
	log.Debug().Msgf("Container Stop: %s", container.ID)
	timeout := time.Millisecond * 100
	err = dockerClient.ContainerStop(ctx, container.ID, &timeout)
	if err != nil {
		return err
	}
	err = dockerClient.ContainerRemove(ctx, container.ID, types.ContainerRemoveOptions{
		RemoveVolumes: true,
		Force:         true,
	})
	if err != nil {
		return err
	}
	return nil
}

func WaitForContainer(client *dockerclient.Client, id string, maxAttempts int, delay time.Duration) error {
	waiter := &system.FunctionWaiter{
		Name:        fmt.Sprintf("wait for container to be running: %s", id),
		MaxAttempts: maxAttempts,
		Delay:       delay,
		Handler: func() (bool, error) {
			container, err := GetContainer(client, id)
			if err != nil {
				return false, err
			}
			if container == nil {
				return false, nil
			}
			return container.State == "running", nil
		},
	}
	return waiter.Wait()
}

func WaitForContainerLogs(client *dockerclient.Client, id string, maxAttempts int, delay time.Duration, findString string) (string, error) {
	lastLogs := ""
	waiter := &system.FunctionWaiter{
		Name:        fmt.Sprintf("wait for container to be running: %s", id),
		MaxAttempts: maxAttempts,
		Delay:       delay,
		Handler: func() (bool, error) {
			container, err := GetContainer(client, id)
			if err != nil {
				return false, err
			}
			if container == nil {
				return false, nil
			}
			if container.State != "running" {
				return false, nil
			}
			stdout, stderr, err := GetLogs(client, id)
			if err != nil {
				return false, err
			}
			lastLogs = stdout + "\n" + stderr
			return strings.Contains(stdout, findString) || strings.Contains(stderr, findString), nil
		},
	}
	err := waiter.Wait()
	return lastLogs, err
}

func PullImage(dockerClient *dockerclient.Client, image string) error {
	imagePullStream, err := dockerClient.ImagePull(
		context.Background(),
		image,
		types.ImagePullOptions{},
	)

	if err != nil {
		return err
	}

	if config.IsDebug() {
		_, err = io.Copy(os.Stdout, imagePullStream)
		if err != nil {
			return err
		}
	}

	return imagePullStream.Close()
}

func RunJobLocally(ctx context.Context, jobspec executor.JobSpec) (string, error) {
	log.Debug().Msgf("in your local docker executor!")

	cm := system.NewCleanupManager()
	cm.RegisterCallback(system.CleanupTracer)
	defer cm.Cleanup()

	peers := DefaultBootstrapAddresses // Default to connecting to defaults
	log.Debug().Msgf("libp2p connecting to: %s", strings.Join(peers, ", "))

	hostPort := DefaultSwarmPort
	transport, err := libp2p.NewTransport(cm, hostPort, peers)
	if err != nil {
		fmt.Printf("error is : %v", err)
	}
	hostID, err := transport.HostID(context.Background())
	if err != nil {
		fmt.Printf("error is : %v", err)
	}

	addrStr := "/ip4/0.0.0.0/tcp/8080"
	addr, err := ma.NewMultiaddr(addrStr)
	if err != nil {
		fmt.Printf("error is : %v", err)
	}
	e, _ := executor_util.NewLocalStandardExecutors(cm, addr.String(), fmt.Sprintf("bacalhau-%s", hostID))
	return e.RunJobLocally(ctx, jobspec)
}
