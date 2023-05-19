package test

import (
	"context"
	"fmt"
	"time"

	"github.com/bacalhau-project/bacalhau/dashboard/api/pkg/model"
	"github.com/bacalhau-project/bacalhau/dashboard/api/pkg/store"
	"github.com/bacalhau-project/bacalhau/dashboard/api/pkg/types"
	"github.com/bacalhau-project/bacalhau/pkg/docker"
	"github.com/bacalhau-project/bacalhau/pkg/localdb"
	"github.com/bacalhau-project/bacalhau/pkg/localdb/postgres"
	"github.com/bacalhau-project/bacalhau/pkg/localdb/shared"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/stretchr/testify/suite"
)

const SpinUpWaitTime = 200 * time.Millisecond

type DashboardTestSuite struct {
	suite.Suite

	client *docker.Client
	opts   *model.ModelOptions
	api    *model.ModelAPI
	user   *types.User

	localDB localdb.LocalDB
	store   *store.PostgresStore

	ctx context.Context
}

var _ suite.SetupAllSuite = (*DashboardTestSuite)(nil)
var _ suite.SetupTestSuite = (*DashboardTestSuite)(nil)
var _ suite.TearDownTestSuite = (*DashboardTestSuite)(nil)

func (s *DashboardTestSuite) SetupSuite() {
	docker.MustHaveDocker(s.T())

	system.InitConfigForTesting(s.T())

	s.ctx = context.Background()

	var err error
	s.client, err = docker.NewDockerClient()
	s.NoError(err)

	s.opts = &model.ModelOptions{
		Libp2pHost:       nil,
		PostgresHost:     "localhost",
		PostgresPort:     5432,
		PostgresDatabase: "postgres",
		PostgresUser:     "postgres",
		PostgresPassword: "postgres",
	}
	// Option to make dashboard hold test jobs for moderation.
	s.opts.SelectionPolicy.RejectStatelessJobs = true

	port, err := nat.NewPort("tcp", fmt.Sprint(s.opts.PostgresPort))
	s.NoError(err)

	container, err := s.client.ContainerCreate(s.ctx, &container.Config{
		Tty: true,
		Env: []string{
			fmt.Sprintf("POSTGRES_DB=%s", s.opts.PostgresDatabase),
			fmt.Sprintf("POSTGRES_USER=%s", s.opts.PostgresUser),
			fmt.Sprintf("POSTGRES_PASSWORD=%s", s.opts.PostgresPassword),
		},
		Image:           "postgres",
		NetworkDisabled: false,
	}, &container.HostConfig{
		NetworkMode: "bridge",
		PortBindings: map[nat.Port][]nat.PortBinding{
			port: {{HostIP: "0.0.0.0", HostPort: fmt.Sprint(s.opts.PostgresPort)}},
		},
		AutoRemove: true,
	}, nil, nil, fmt.Sprintf("postgres-%s", s.T().Name()))
	s.NoError(err)

	err = s.client.ContainerStart(s.ctx, container.ID, dockertypes.ContainerStartOptions{})
	s.NoError(err)

	s.T().Cleanup(func() {
		err := s.client.ContainerStop(context.Background(), container.ID, time.Second)
		s.NoError(err)
	})
}

// SetupTest implements suite.SetupTestSuite
func (s *DashboardTestSuite) SetupTest() {
	var cancel context.CancelFunc
	const maxTestDuration = time.Minute
	s.ctx, cancel = context.WithTimeout(context.Background(), maxTestDuration)
	s.T().Cleanup(cancel)

	var err error
	for {
		s.localDB, err = postgres.NewPostgresDatastore(
			s.opts.PostgresHost,
			s.opts.PostgresPort,
			s.opts.PostgresDatabase,
			s.opts.PostgresUser,
			s.opts.PostgresPassword,
			true,
		)

		if err != nil {
			s.T().Log(err.Error())
			time.Sleep(SpinUpWaitTime)
			continue
		}

		s.store, err = store.NewPostgresStore(
			s.opts.PostgresHost,
			s.opts.PostgresPort,
			s.opts.PostgresDatabase,
			s.opts.PostgresUser,
			s.opts.PostgresPassword,
			true,
		)

		if err != nil {
			s.T().Log(err.Error())
			time.Sleep(SpinUpWaitTime)
			continue
		}

		s.api, err = model.NewModelAPI(*s.opts)
		if err != nil {
			s.T().Log(err.Error())
			time.Sleep(SpinUpWaitTime)
			continue
		}

		break
	}

	s.user, err = s.api.AddUser(s.ctx, "test", "password")
	s.NoError(err)
}

// TearDownTest implements suite.TearDownTestSuite
func (s *DashboardTestSuite) TearDownTest() {
	s.NoError(s.store.MigrateDown())

	postgresDB := s.localDB.(*shared.GenericSQLDatastore)
	s.NoError(postgresDB.MigrateDown())
}
