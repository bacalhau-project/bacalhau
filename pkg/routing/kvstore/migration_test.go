//go:build unit || !integration

package kvstore_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/nats-io/nats-server/v2/server"
	natsserver "github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/routing/kvstore"
)

type KVMigrationSuite struct {
	suite.Suite
	nats   *server.Server
	client *nats.Conn
	js     jetstream.JetStream
}

func (s *KVMigrationSuite) SetupTest() {
	opts := &natsserver.DefaultTestOptions
	opts.Port = TEST_PORT
	opts.JetStream = true
	opts.StoreDir = s.T().TempDir()

	s.nats = natsserver.RunServer(opts)
	var err error
	s.client, err = nats.Connect(s.nats.Addr().String())
	s.Require().NoError(err)

	s.js, err = jetstream.New(s.client)
	s.Require().NoError(err)
}

func (s *KVMigrationSuite) TearDownTest() {
	s.nats.Shutdown()
	s.client.Close()
}

func TestKVMigrationSuite(t *testing.T) {
	suite.Run(t, new(KVMigrationSuite))
}

func (s *KVMigrationSuite) TestMigrationFromNodeInfoToNodeState() {
	ctx := context.Background()

	// Create 'from' bucket and populate it, simulating a requester on v130 with state to migrate.
	fromKV, err := s.js.CreateKeyValue(ctx, jetstream.KeyValueConfig{Bucket: kvstore.BucketNameV0})
	s.Require().NoError(err)

	nodeInfos := []models.NodeInfo{
		generateNodeInfo("node1", models.EngineDocker),
		generateNodeInfo("node2", models.EngineWasm),
		generateNodeInfo("node3", models.EngineDocker, models.EngineWasm),
	}

	// populate bucket with models.NodeInfo, these will be migrated to models.NodeState
	for _, n := range nodeInfos {
		data, err := json.Marshal(n)
		s.Require().NoError(err)
		_, err = fromKV.Put(ctx, n.ID(), data)
		s.Require().NoError(err)
	}

	fromBucket := kvstore.BucketNameV0
	toBucket := kvstore.BucketNameCurrent

	// Open a NodeStore to trigger migration
	ns, err := kvstore.NewNodeStore(ctx, kvstore.NodeStoreParams{
		BucketName: toBucket,
		Client:     s.client,
	})
	s.Require().NoError(err)

	// Assert the migrated data is correct
	for _, ni := range nodeInfos {
		ns, err := ns.Get(ctx, ni.ID())
		s.Require().NoError(err)
		s.Equal(models.NodeStates.DISCONNECTED, ns.Connection)
		s.Equal(models.NodeMembership.PENDING, ns.Membership)
		s.Equal(ni, ns.Info)
	}

	// Assert the from bucket has been cleaned up
	_, err = s.js.KeyValue(ctx, fromBucket)
	s.Require().Equal(jetstream.ErrBucketNotFound, err)
}

func (s *KVMigrationSuite) TestMigrationStoreEmpty() {
	ctx := context.Background()

	// Create an empty 'from' bucket
	_, err := s.js.CreateKeyValue(ctx, jetstream.KeyValueConfig{Bucket: kvstore.BucketNameV0})
	s.Require().NoError(err)

	fromBucket := kvstore.BucketNameV0
	toBucket := kvstore.BucketNameCurrent

	// Open a NodeStore to trigger migration, in this case there is a from bucket, but it's empty.
	ns, err := kvstore.NewNodeStore(ctx, kvstore.NodeStoreParams{
		BucketName: toBucket,
		Client:     s.client,
	})
	s.Require().NoError(err)

	// Assert the from bucket has been cleaned up
	_, err = s.js.KeyValue(ctx, fromBucket)
	s.Require().Contains(err.Error(), "bucket not found")

	// Assert that no data was migrated since the from bucket was empty
	resp, err := ns.List(ctx)
	s.Require().NoError(err)
	s.Require().Len(resp, 0)
}

func (s *KVMigrationSuite) TestMigrationStoreDNE() {
	ctx := context.Background()

	fromBucket := kvstore.BucketNameV0
	toBucket := kvstore.BucketNameCurrent

	// Open a NodeStore to trigger migration, in this case there isn't a from bucket to migrate from.
	ns, err := kvstore.NewNodeStore(ctx, kvstore.NodeStoreParams{
		BucketName: toBucket,
		Client:     s.client,
	})
	s.Require().NoError(err)

	// Assert the from bucket has been cleaned up
	_, err = s.js.KeyValue(ctx, fromBucket)
	s.Require().Contains(err.Error(), "bucket not found")

	// Assert that no data was migrated since the from bucket DNE (does not exist)
	resp, err := ns.List(ctx)
	s.Require().NoError(err)
	s.Require().Len(resp, 0)
}
