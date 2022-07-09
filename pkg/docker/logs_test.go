package docker

import (
	"context"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/stretchr/testify/require"
)

func TestStreamLogs(t *testing.T) {

	// we need to run this first or we get a "no such image" error
	_, err := system.RunCommandGetResults( // nolint:govet // shadowing ok
		"docker",
		[]string{"pull", "ubuntu"},
	)
	require.NoError(t, err)

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(10*time.Second))
	defer cancel()

	cl, err := NewDockerClient()
	require.NoError(t, err)

	if !IsInstalled(cl) {
		t.Skip("docker is not installed")
	}

	cfg := container.Config{
		Image: "ubuntu",
		Tty:   false,
		Cmd: []string{
			"/bin/bash",
			"-c",
			"echo 'hello stdout' && >&2 echo 'hello stderr'",
		},
		NetworkDisabled: true,
	}

	c, err := cl.ContainerCreate(ctx, &cfg, nil, nil, nil, "")
	require.NoError(t, err)

	defer func() {
		if err := RemoveContainer(cl, c.ID); err != nil {
			t.Fatal(err.Error())
		}
	}()

	err = cl.ContainerStart(ctx, c.ID, types.ContainerStartOptions{})
	require.NoError(t, err)

	ls, err := StreamLogs(ctx, cl, c.ID)
	require.NoError(t, err)
	defer ls.Close()

	statusCh, errCh := cl.ContainerWait(ctx, c.ID, container.WaitConditionNotRunning)
	select {
	case err = <-errCh:
		t.Fatal(err)
	case status := <-statusCh:
		if status.StatusCode != 0 {
			t.Fatal("container exited with non-zero status")
		}
		if status.Error != nil {
			t.Fatal(status.Error.Message)
		}
	}

	// Let's see what the logs look like...
	stdout, stderr, err := ls.Logs()
	require.NoError(t, err)
	require.Equal(t, "hello stdout\n", stdout)
	require.Equal(t, "hello stderr\n", stderr)
}
