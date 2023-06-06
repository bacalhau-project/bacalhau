//go:build integration || !unit

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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	shared2 "github.com/bacalhau-project/bacalhau/dashboard/api/pkg/localdb/shared"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/docker"
	_ "github.com/bacalhau-project/bacalhau/pkg/logger"
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

	require.NoError(t, client.PullImage(ctx, "postgres", config.GetDockerCredentials()))
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

	var datastore *shared2.GenericSQLDatastore
	testingSuite := new(shared2.GenericSQLSuite)
	testingSuite.SetupHandler = func() *shared2.GenericSQLDatastore {
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
