//nolint:all
package shared

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/localdb"
	"github.com/filecoin-project/bacalhau/pkg/logger"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	model "github.com/filecoin-project/bacalhau/pkg/model/v1beta1"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type GenericSQLSuite struct {
	suite.Suite
	SetupHandler    func() *GenericSQLDatastore
	TeardownHandler func()
	datastore       *GenericSQLDatastore
	db              *sql.DB
}

func (suite *GenericSQLSuite) SetupTest() {
	suite.Require().NoError(system.InitConfigForTesting(suite.T()))
	logger.ConfigureTestLogging(suite.T())
	datastore := suite.SetupHandler()
	suite.datastore = datastore
	suite.db = datastore.GetDB()
}

func (suite *GenericSQLSuite) TearDownSuite() {
	if suite.TeardownHandler != nil {
		suite.TeardownHandler()
	}
}

func (suite *GenericSQLSuite) TestSQLiteMigrations() {
	_, err := suite.db.Exec(`
insert into job (id) values ('123');
`)
	require.NoError(suite.T(), err)
	var id string
	rows, err := suite.db.Query(`
select id from job;
`)
	require.NoError(suite.T(), err)
	defer rows.Close()
	for rows.Next() {
		err = rows.Scan(&id)
		if err != nil {
			log.Fatal(err)
		}
		log.Println(id)
	}
	err = rows.Err()
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), id, "123")
}

func (suite *GenericSQLSuite) TestRoundtripJob() {
	job := &model.Job{
		Metadata: model.Metadata{
			ID: "hellojob",
		},
	}
	err := suite.datastore.AddJob(context.Background(), job)
	require.NoError(suite.T(), err)
	loadedJob, err := suite.datastore.GetJob(context.Background(), job.Metadata.ID)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), job.Metadata.ID, loadedJob.Metadata.ID)
}

func (suite *GenericSQLSuite) TestAddingTwoJobs() {
	err := suite.datastore.AddJob(context.Background(), &model.Job{
		Metadata: model.Metadata{
			ID: "hellojob1",
		},
	})
	require.NoError(suite.T(), err)
	err = suite.datastore.AddJob(context.Background(), &model.Job{
		Metadata: model.Metadata{
			ID: "hellojob2",
		},
	})
	require.NoError(suite.T(), err)
}

//nolint:funlen
func (suite *GenericSQLSuite) TestGetJobs() {
	jobCount := 100
	dateString := "2021-11-22"
	date, err := time.Parse("2006-01-02", dateString)
	require.NoError(suite.T(), err)

	for i := 0; i < jobCount; i++ {
		date = date.Add(time.Hour * 1)
		annotations := []string{"apples", fmt.Sprintf("oranges%d", i)}
		if i < jobCount/2 {
			annotations = append(annotations, "bananas")
		}
		job := &model.Job{
			Metadata: model.Metadata{
				ID:        fmt.Sprintf("hellojob%d", i),
				CreatedAt: date,
				ClientID:  fmt.Sprintf("testclient%d", i),
			},
			Spec: model.Spec{
				Annotations: annotations,
			},
		}
		err = suite.datastore.AddJob(context.Background(), job)
		require.NoError(suite.T(), err)
	}

	// sanity check that we can see all jobs
	allJobs, err := suite.datastore.GetJobs(context.Background(), localdb.JobQuery{})
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), jobCount, len(allJobs))

	// sort by date asc and check first id
	sortedAsc, err := suite.datastore.GetJobs(context.Background(), localdb.JobQuery{
		SortBy:      "created_at",
		SortReverse: false,
	})
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), jobCount, len(sortedAsc))
	require.Equal(suite.T(), "hellojob0", sortedAsc[0].Metadata.ID)

	// sort by date desc and check first id
	sortedDesc, err := suite.datastore.GetJobs(context.Background(), localdb.JobQuery{
		SortBy:      "created_at",
		SortReverse: true,
	})
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), jobCount, len(sortedDesc))
	require.Equal(suite.T(), "hellojob99", sortedDesc[0].Metadata.ID)

	// check basic limit
	const limit = 10
	tenJobs, err := suite.datastore.GetJobs(context.Background(), localdb.JobQuery{
		Limit: limit,
	})
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), limit, len(tenJobs))

	// pagination
	tenJobsSecondPage, err := suite.datastore.GetJobs(context.Background(), localdb.JobQuery{
		Limit:  limit,
		Offset: 10,
	})
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), limit, len(tenJobsSecondPage))
	require.Equal(suite.T(), "hellojob10", tenJobsSecondPage[0].Metadata.ID)

	loadedJobCount, err := suite.datastore.GetJobsCount(context.Background(), localdb.JobQuery{
		IncludeTags: []model.IncludedTag{"bananas"},
		Limit:       1,
		SortBy:      "created_at",
		SortReverse: true,
	})
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), jobCount/2, loadedJobCount)

	sortedWithLimit, err := suite.datastore.GetJobs(context.Background(), localdb.JobQuery{
		SortBy:      "created_at",
		SortReverse: true,
		Limit:       limit,
	})
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), limit, len(sortedWithLimit))

	// a label they all have
	withAppleLabel, err := suite.datastore.GetJobs(context.Background(), localdb.JobQuery{
		IncludeTags: []model.IncludedTag{"apples"},
	})
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), jobCount, len(withAppleLabel))

	// a label only half of them have
	withBananasLabel, err := suite.datastore.GetJobs(context.Background(), localdb.JobQuery{
		IncludeTags: []model.IncludedTag{"bananas"},
	})
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), jobCount/2, len(withBananasLabel))

	// a label only half of them have plus a second label
	withAppleAndBananaLabel, err := suite.datastore.GetJobs(context.Background(), localdb.JobQuery{
		IncludeTags: []model.IncludedTag{"apples", "bananas"},
	})
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), jobCount/2, len(withAppleAndBananaLabel))

	// combine three labels - only 1 result
	withAppleAndBananaAndIDLabel, err := suite.datastore.GetJobs(context.Background(), localdb.JobQuery{
		IncludeTags: []model.IncludedTag{"apples", "bananas", "oranges17"},
	})
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), 1, len(withAppleAndBananaAndIDLabel))

	// exclude with a single id
	basicExclude, err := suite.datastore.GetJobs(context.Background(), localdb.JobQuery{
		ExcludeTags: []model.ExcludedTag{"oranges17"},
	})
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), jobCount-1, len(basicExclude))

	// exclude half
	halfExclude, err := suite.datastore.GetJobs(context.Background(), localdb.JobQuery{
		ExcludeTags: []model.ExcludedTag{"bananas"},
	})
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), jobCount/2, len(halfExclude))

	// include and exclude
	includeAndExclude, err := suite.datastore.GetJobs(context.Background(), localdb.JobQuery{
		IncludeTags: []model.IncludedTag{"apples", "bananas"},
		ExcludeTags: []model.ExcludedTag{"oranges17"},
	})
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), jobCount/2-1, len(includeAndExclude))

	// load jobs from one client
	singleClient, err := suite.datastore.GetJobs(context.Background(), localdb.JobQuery{
		ClientID: "testclient17",
	})
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), 1, len(singleClient))
}

func (suite *GenericSQLSuite) TestJobEvents() {
	eventCount := 5
	job := &model.Job{
		Metadata: model.Metadata{
			ID: "hellojob",
		},
	}
	err := suite.datastore.AddJob(context.Background(), job)
	require.NoError(suite.T(), err)

	dateString := "2021-11-22"
	date, err := time.Parse("2006-01-02", dateString)
	require.NoError(suite.T(), err)

	for i := 0; i < eventCount; i++ {
		date = date.Add(time.Minute * 1)
		eventName := model.JobEventBid
		localEventName := model.JobLocalEventBid
		if i == 0 {
			eventName = model.JobEventCreated
			localEventName = model.JobLocalEventSelected
		}
		ev := model.JobEvent{
			JobID:     job.Metadata.ID,
			EventName: eventName,
			EventTime: date,
		}
		err = suite.datastore.AddEvent(context.Background(), job.Metadata.ID, ev)
		require.NoError(suite.T(), err)
		localEvent := model.JobLocalEvent{
			JobID:     job.Metadata.ID,
			EventName: localEventName,
		}
		err = suite.datastore.AddLocalEvent(context.Background(), job.Metadata.ID, localEvent)
		require.NoError(suite.T(), err)
	}

	jobEvents, err := suite.datastore.GetJobEvents(context.Background(), job.Metadata.ID)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), eventCount, len(jobEvents))
	// check that the EventName of the first event is the created one
	require.Equal(suite.T(), model.JobEventCreated, jobEvents[0].EventName)

	// repeat the same event block above but for local events
	localEvents, err := suite.datastore.GetJobLocalEvents(context.Background(), job.Metadata.ID)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), eventCount, len(localEvents))
	// check that the EventName of the first event is the created one
	require.Equal(suite.T(), model.JobLocalEventSelected, localEvents[0].EventName)
}

func (suite *GenericSQLSuite) TestJobState() {
	job := &model.Job{
		Metadata: model.Metadata{
			ID: "hellojob",
		},
	}
	err := suite.datastore.AddJob(context.Background(), job)
	require.NoError(suite.T(), err)
	err = suite.datastore.UpdateShardState(
		context.Background(),
		job.Metadata.ID,
		"node-test",
		0,
		model.JobShardState{
			NodeID:     "node-test",
			ShardIndex: 0,
			State:      model.JobStateRunning,
		},
	)
	require.NoError(suite.T(), err)
	state, err := suite.datastore.GetJobState(context.Background(), job.Metadata.ID)
	require.NoError(suite.T(), err)
	node, ok := state.Nodes["node-test"]
	require.True(suite.T(), ok)
	shard, ok := node.Shards[0]
	require.True(suite.T(), ok)
	require.Equal(suite.T(), "node-test", shard.NodeID)
	require.Equal(suite.T(), model.JobStateRunning, shard.State)
}

func (suite *GenericSQLSuite) TestUpdateDeal() {
	job := &model.Job{
		Metadata: model.Metadata{
			ID: "hellojob",
		},
		Spec: model.Spec{
			Deal: model.Deal{
				Concurrency: 1,
			},
		},
	}
	err := suite.datastore.AddJob(context.Background(), job)
	require.NoError(suite.T(), err)
	err = suite.datastore.UpdateJobDeal(
		context.Background(),
		job.Metadata.ID,
		model.Deal{
			Concurrency: 3,
		},
	)
	require.NoError(suite.T(), err)
	updatedJob, err := suite.datastore.GetJob(context.Background(), job.Metadata.ID)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), 3, updatedJob.Spec.Deal.Concurrency)
}
