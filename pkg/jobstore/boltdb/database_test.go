//go:build unit || !integration

package boltjobstore

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/benbjohnson/clock"
	"github.com/stretchr/testify/suite"
	bolt "go.etcd.io/bbolt"
)

type DatabaseTestSuite struct {
	suite.Suite
	store  *BoltJobStore
	dbFile string
	clock  *clock.Mock
	ctx    context.Context
}

func TestDatabaseTestSuite(t *testing.T) {
	suite.Run(t, new(DatabaseTestSuite))
}

func (s *DatabaseTestSuite) SetupTest() {
	s.clock = clock.NewMock()
	s.ctx = context.Background()

	dir, _ := os.MkdirTemp("", "bacalhau-jobstore-test")
	s.dbFile = filepath.Join(dir, "testing.db")

	s.store, _ = NewBoltJobStore(s.dbFile, WithClock(s.clock))
}

func (s *DatabaseTestSuite) TearDownTest() {
	s.store.Close(s.ctx)
	os.Remove(s.dbFile)
}

func (s *DatabaseTestSuite) TestGetDatabaseBad() {
	_, err := GetDatabase("")
	s.Error(err)
}

func (s *DatabaseTestSuite) TestGetBucketDataErr() {
	_ = s.store.database.View(func(tx *bolt.Tx) (err error) {
		data := GetBucketData(tx, NewBucketPath("non-existent"), []byte("nope"))
		s.Nil(data)
		return nil
	})
}

func (s *DatabaseTestSuite) TestBucketPartialSearch() {
	err := s.store.database.Update(func(tx *bolt.Tx) error {
		_, err := NewBucketPath("root/notbucket-000").Get(tx, true)
		s.NoError(err)

		_, err = NewBucketPath("root/bucket-123").Get(tx, true)
		s.NoError(err)

		_, err = NewBucketPath("root/bucket-456").Get(tx, true)
		s.NoError(err)

		root := tx.Bucket([]byte("root"))
		s.NotNil(root)

		keys, err := GetBucketsByPrefix(tx, root, []byte("bucket"))
		s.NoError(err)

		s.Equal(2, len(keys))
		s.Equal("bucket-123", string(keys[0]))
		s.Equal("bucket-456", string(keys[1]))

		return nil
	})
	s.NoError(err)
}

func (s *DatabaseTestSuite) TestDeadJobs() {

	type testcase struct {
		JobID      string
		IsTerminal bool
		Type       string
	}

	testcases := []testcase{
		{
			JobID:      "1",
			IsTerminal: true,
			Type:       "batch",
		},
		{
			JobID:      "2",
			IsTerminal: false,
			Type:       "batch",
		},
		{
			JobID:      "3",
			IsTerminal: false,
			Type:       "service",
		},
	}

	lifetimes := make(map[string]time.Duration)
	lifetimes["batch"] = time.Duration(1) * time.Second

	for _, tc := range testcases {
		job := makeJob(
			model.EngineDocker,
			model.PublisherNoop,
			[]string{"bash", "-c", "echo hello"})
		job.Metadata.ID = tc.JobID
		job.Metadata.ClientID = tc.JobID

		err := s.store.CreateJob(s.ctx, *job)
		s.Require().NoError(err)

		if tc.IsTerminal {
			// Make sure job is in terminal state
			s.store.UpdateJobState(s.ctx, jobstore.UpdateJobStateRequest{
				JobID:    job.Metadata.ID,
				NewState: model.JobStateCancelled,
			})
		}
		var identifiers []string
		err = s.store.database.View(func(tx *bolt.Tx) (err error) {
			s.clock.Add(1 * time.Hour)
			identifiers, err = FindDeadJobs(tx, s.clock.Now().UTC(), lifetimes)
			return err
		})
		s.Require().NoError(err)
		s.Require().Equal(identifiers, []string{"1"})
	}
}
