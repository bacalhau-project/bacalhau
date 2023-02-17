//go:build integration

package postgres

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/filecoin-project/bacalhau/pkg/docker"
	"github.com/filecoin-project/bacalhau/pkg/localdb/shared"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestPostgresSuite(t *testing.T) {
	docker.MustHaveDocker(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	client, err := docker.NewDockerClient()
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, client.Close())
	})

	require.NoError(t, client.PullImage(ctx, "postgres"))
	c, err := client.ContainerCreate(ctx, &container.Config{
		Image:        "postgres",
		ExposedPorts: map[nat.Port]struct{}{},
		Env:          []string{"POSTGRES_DB=postgres", "POSTGRES_USER=postgres", "POSTGRES_PASSWORD=postgres"},
	}, &container.HostConfig{
		PortBindings: map[nat.Port][]nat.PortBinding{
			"5432/tcp": {{}},
		},
	}, nil, nil, fmt.Sprintf("postgres-%s", t.Name()))
	require.NoError(t, err)

	t.Cleanup(func() {
		assert.NoError(t, client.RemoveContainer(context.Background(), c.ID))
	})

	require.NoError(t, client.ContainerStart(ctx, c.ID, dockertypes.ContainerStartOptions{}))

	var status dockertypes.ContainerJSON
	for {
		status, err = client.ContainerInspect(ctx, c.ID)
		require.NoError(t, err)

		if status.State.Status == "running" {
			break
		}
		time.Sleep(1 * time.Second)
	}

	port, err := strconv.Atoi(status.NetworkSettings.Ports["5432/tcp"][0].HostPort)
	require.NoError(t, err)

	var datastore *shared.GenericSQLDatastore
	testingSuite := new(shared.GenericSQLSuite)
	testingSuite.SetupHandler = func() *shared.GenericSQLDatastore {
		if datastore == nil {
			for {
				datastore, err = NewPostgresDatastore(
					"localhost",
					port,
					"postgres",
					"postgres",
					"postgres",
					true,
				)
				if err != nil {
					time.Sleep(1 * time.Second)
				} else {
					break
				}
			}
		} else {
			err := datastore.MigrateDown()
			require.NoError(t, err)
			err = datastore.MigrateUp()
			require.NoError(t, err)
		}
		return datastore
	}

	suite.Run(t, testingSuite)
}
