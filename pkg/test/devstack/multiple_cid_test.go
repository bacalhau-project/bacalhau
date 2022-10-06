//go:build !(windows && unit)

package devstack

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/computenode"
	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/job"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type MultipleCIDSuite struct {
	suite.Suite
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestMultipleCIDSuite(t *testing.T) {
	suite.Run(t, new(MultipleCIDSuite))
}

// Before all suite
func (s *MultipleCIDSuite) SetupAllSuite() {

}

// Before each test
func (s *MultipleCIDSuite) SetupTest() {
	err := system.InitConfigForTesting()
	require.NoError(s.T(), err)
}

func (suite *MultipleCIDSuite) TearDownTest() {
}

func (s *MultipleCIDSuite) TearDownAllSuite() {

}

func (s *MultipleCIDSuite) TestMultipleCIDs() {
	ctx := context.Background()

	stack, cm := SetupTest(
		ctx,
		s.T(),
		1,
		0,
		computenode.NewDefaultComputeNodeConfig(),
	)
	defer TeardownTest(stack, cm)

	t := system.GetTracer()
	ctx, rootSpan := system.NewRootSpan(ctx, t, "pkg/test/devstack/multiple_cid_test/testmultiplecids")
	defer rootSpan.End()
	cm.RegisterCallback(system.CleanupTraceProvider)

	fileCid1, err := devstack.AddTextToNodes(ctx, []byte("file1"), devstack.ToIPFSClients(stack.Nodes[:1])...)
	require.NoError(s.T(), err)

	fileCid2, err := devstack.AddTextToNodes(ctx, []byte("file2"), devstack.ToIPFSClients(stack.Nodes[:1])...)
	require.NoError(s.T(), err)

	apiUri := stack.Nodes[0].APIServer.GetURI()
	apiClient := publicapi.NewAPIClient(apiUri)

	j := &model.Job{}
	j.Spec = model.Spec{
		Engine:    model.EngineDocker,
		Verifier:  model.VerifierNoop,
		Publisher: model.PublisherIpfs,
		Docker: model.JobSpecDocker{
			Image: "ubuntu",
			Entrypoint: []string{
				"ls",
			},
		},
	}
	j.Spec.Inputs = []model.StorageSpec{
		{
			StorageSource: model.StorageSourceIPFS,
			CID:           fileCid1,
			MountPath:     "/inputs-1",
		},
		{
			StorageSource: model.StorageSourceIPFS,
			CID:           fileCid2,
			MountPath:     "/inputs-2",
		},
	}
	j.Deal = model.Deal{Concurrency: 1}

	submittedJob, err := apiClient.Submit(ctx, j, nil)
	require.NoError(s.T(), err)

	resolver := apiClient.GetJobStateResolver()

	err = resolver.Wait(
		ctx,
		submittedJob.ID,
		1,
		job.WaitThrowErrors([]model.JobStateType{
			model.JobStateError,
		}),
		job.WaitForJobStates(map[model.JobStateType]int{
			model.JobStateCompleted: 1,
		}),
	)
	require.NoError(s.T(), err)

	shards, err := resolver.GetShards(ctx, submittedJob.ID)
	require.NoError(s.T(), err)

	shard := shards[0]

	node, err := stack.GetNode(ctx, shard.NodeID)
	require.NoError(s.T(), err)

	outputDir, err := ioutil.TempDir("", "bacalhau-ipfs-multiple-cid-test")
	require.NoError(s.T(), err)
	require.NotEmpty(s.T(), shard.PublishedResult.CID)

	outputPath := filepath.Join(outputDir, shard.PublishedResult.CID)
	err = node.IPFSClient.Get(ctx, shard.PublishedResult.CID, outputPath)
	require.NoError(s.T(), err)

	stdout, err := os.ReadFile(fmt.Sprintf("%s/stdout", outputPath))
	require.NoError(s.T(), err)

	// check that the stdout string containts the text hello-cid-1.txt and hello-cid-2.txt
	require.Contains(s.T(), string(stdout), "hello-cid-1.txt")
	require.Contains(s.T(), string(stdout), "hello-cid-2.txt")
}

func (s *MultipleCIDSuite) TestMultipleURLs() {
	file1 := "hello-cid-1.txt"
	file2 := "hello-cid-2.txt"
	mount1 := "/inputs-1"
	mount2 := "/inputs-2"

	files := map[string]string{
		fmt.Sprintf("/%s", file1): "Before you marry a person, you should first make them use a computer with slow Internet to see who they really are.\n",
		fmt.Sprintf("/%s", file2): "I walk around like everything's fine, but deep down, inside my shoe, my sock is sliding off.\n",
	}

	ctx := context.Background()

	stack, cm := SetupTest(
		ctx,
		s.T(),
		1,
		0,
		computenode.ComputeNodeConfig{
			JobSelectionPolicy: computenode.JobSelectionPolicy{
				Locality: computenode.Anywhere,
			},
		},
	)
	defer TeardownTest(stack, cm)

	t := system.GetTracer()
	ctx, rootSpan := system.NewRootSpan(ctx, t, "pkg/test/devstack/multiple_cid_test/testmultipleurls")
	defer rootSpan.End()
	cm.RegisterCallback(system.CleanupTraceProvider)

	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		content, ok := files[r.URL.Path]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("not found"))
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(content))
		}
	}))
	defer svr.Close()

	apiUri := stack.Nodes[0].APIServer.GetURI()
	apiClient := publicapi.NewAPIClient(apiUri)

	j := model.NewJob()
	j.Spec = model.Spec{
		Engine:    model.EngineDocker,
		Verifier:  model.VerifierNoop,
		Publisher: model.PublisherIpfs,
		Docker: model.JobSpecDocker{
			Image: "ubuntu",
			Entrypoint: []string{
				"bash", "-c",
				fmt.Sprintf("cat /%s/%s && cat /%s/%s",
					mount1, file1,
					mount2, file2),
			},
		},
	}
	j.Spec.Inputs = []model.StorageSpec{
		{
			StorageSource: model.StorageSourceURLDownload,
			URL:           fmt.Sprintf("%s/%s", svr.URL, file1),
			MountPath:     fmt.Sprintf("/%s", mount1),
		},
		{
			StorageSource: model.StorageSourceURLDownload,
			URL:           fmt.Sprintf("%s/%s", svr.URL, file2),
			MountPath:     fmt.Sprintf("/%s", mount2),
		},
	}
	j.Deal = model.Deal{Concurrency: 1}

	submittedJob, err := apiClient.Submit(ctx, j, nil)
	require.NoError(s.T(), err)

	resolver := apiClient.GetJobStateResolver()

	err = resolver.Wait(
		ctx,
		submittedJob.ID,
		1,
		job.WaitThrowErrors([]model.JobStateType{
			model.JobStateError,
		}),
		job.WaitForJobStates(map[model.JobStateType]int{
			model.JobStateCompleted: 1,
		}),
	)
	require.NoError(s.T(), err)

	outputDir, err := ioutil.TempDir("", "bacalhau-ipfs-multiple-url-test")
	require.NoError(s.T(), err)

	shards, err := resolver.GetShards(ctx, submittedJob.ID)
	require.NoError(s.T(), err)
	require.True(s.T(), len(shards) > 0, "No shards created during submit job.")

	jobEvents, err := apiClient.GetEvents(ctx, submittedJob.ID)
	require.NoError(s.T(), err, "Could not get job events.")
	fmt.Printf("=========== JOB EVENTS =========")
	for _, e := range jobEvents {
		fmt.Printf("Event: %+v\n", e.EventName)
	}

	shard := shards[0]
	require.NotEmpty(s.T(), shard.PublishedResult.CID)

	node, err := stack.GetNode(ctx, shard.NodeID)
	require.NoError(s.T(), err)

	outputPath := filepath.Join(outputDir, shard.PublishedResult.CID)
	err = node.IPFSClient.Get(ctx, shard.PublishedResult.CID, outputPath)
	require.NoError(s.T(), err)
	require.FileExists(s.T(), fmt.Sprintf("%s/stdout", outputPath))

	stdout, err := os.ReadFile(fmt.Sprintf("%s/stdout", outputPath))
	log.Debug().Str("stdout", string(stdout)).Msg("stdout")
	require.NoError(s.T(), err)

	require.Equal(s.T(), files[fmt.Sprintf("/%s", file1)]+
		files[fmt.Sprintf("/%s", file2)],
		string(stdout))
}

func (s *MultipleCIDSuite) TestIPFSURLCombo() {
	ipfsfile := "hello-ipfs.txt"
	urlfile := "hello-url.txt"
	ipfsmount := "/inputs-1"
	urlmount := "/inputs-2"

	URLContent := "Common sense is like deodorant. The people who need it most never use it.\n"
	IPFSContent := "Truth hurts. Maybe not as much as jumping on a bicycle with a seat missing, but it hurts.\n"

	ctx := context.Background()

	stack, cm := SetupTest(
		ctx,
		s.T(),
		1,
		0,
		computenode.ComputeNodeConfig{
			JobSelectionPolicy: computenode.JobSelectionPolicy{
				Locality: computenode.Anywhere,
			},
		},
	)
	defer TeardownTest(stack, cm)

	t := system.GetTracer()
	ctx, rootSpan := system.NewRootSpan(ctx, t, "pkg/test/devstack/multiple_cid_test/testmultipleurls")
	defer rootSpan.End()
	cm.RegisterCallback(system.CleanupTraceProvider)

	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(URLContent))
	}))
	defer svr.Close()

	cid, err := devstack.AddTextToNodes(ctx,
		[]byte(IPFSContent),
		devstack.ToIPFSClients(stack.Nodes[:1])...)
	require.NoError(s.T(), err)

	apiUri := stack.Nodes[0].APIServer.GetURI()
	apiClient := publicapi.NewAPIClient(apiUri)

	j := &model.Job{}
	j.Spec = model.Spec{
		Engine:    model.EngineDocker,
		Verifier:  model.VerifierNoop,
		Publisher: model.PublisherIpfs,
		Docker: model.JobSpecDocker{
			Image: "ubuntu",
			Entrypoint: []string{
				"bash", "-c",
				fmt.Sprintf("cat %s/%s", ipfsmount, ipfsfile),
				// fmt.Sprintf("cat %s/%s && cat %s/%s",
				// 	ipfsmount, ipfsfile,
				// 	urlmount, urlfile),
			},
		},
	}
	j.Spec.Inputs = []model.StorageSpec{
		{
			StorageSource: model.StorageSourceURLDownload,
			URL:           fmt.Sprintf("%s/%s", svr.URL, urlfile),
			MountPath:     urlmount,
		},
		{
			StorageSource: model.StorageSourceIPFS,
			CID:           cid,
			MountPath:     ipfsmount,
		},
	}
	j.Deal = model.Deal{Concurrency: 1}

	submittedJob, err := apiClient.Submit(ctx, j, nil)
	require.NoError(s.T(), err)

	resolver := apiClient.GetJobStateResolver()

	err = resolver.Wait(
		ctx,
		submittedJob.ID,
		1,
		job.WaitThrowErrors([]model.JobStateType{
			model.JobStateError,
		}),
		job.WaitForJobStates(map[model.JobStateType]int{
			model.JobStateCompleted: 1,
		}),
	)
	require.NoError(s.T(), err)

	shards, err := resolver.GetShards(ctx, submittedJob.ID)
	require.NoError(s.T(), err)

	shard := shards[0]

	node, err := stack.GetNode(ctx, shard.NodeID)
	require.NoError(s.T(), err)

	outputDir, err := ioutil.TempDir("", "bacalhau-ipfs-multiple-url-test")
	require.NoError(s.T(), err)
	require.NotEmpty(s.T(), shard.PublishedResult.CID)

	outputPath := filepath.Join(outputDir, shard.PublishedResult.CID)
	err = node.IPFSClient.Get(ctx, shard.PublishedResult.CID, outputPath)
	require.NoError(s.T(), err)

	stdout, err := os.ReadFile(fmt.Sprintf("%s/stdout", outputPath))
	require.NoError(s.T(), err)

	require.Equal(s.T(), URLContent+IPFSContent, string(stdout))
}
