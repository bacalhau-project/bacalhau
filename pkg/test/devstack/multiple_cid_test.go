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
func (suite *MultipleCIDSuite) SetupAllSuite() {

}

// Before each test
func (suite *MultipleCIDSuite) SetupTest() {
	err := system.InitConfigForTesting()
	require.NoError(suite.T(), err)
}

func (suite *MultipleCIDSuite) TearDownTest() {
}

func (suite *MultipleCIDSuite) TearDownAllSuite() {

}

func (suite *MultipleCIDSuite) TestMultipleCIDs() {
	ctx := context.Background()

	stack, cm := SetupTest(
		ctx,
		suite.T(),
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
	require.NoError(suite.T(), err)

	fileCid2, err := devstack.AddTextToNodes(ctx, []byte("file2"), devstack.ToIPFSClients(stack.Nodes[:1])...)
	require.NoError(suite.T(), err)

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
			Path:          "/hello-cid-1.txt",
		},
		{
			StorageSource: model.StorageSourceIPFS,
			CID:           fileCid2,
			Path:          "/hello-cid-2.txt",
		},
	}
	j.Deal = model.Deal{Concurrency: 1}

	submittedJob, err := apiClient.Submit(ctx, j, nil)
	require.NoError(suite.T(), err)

	resolver := apiClient.GetJobStateResolver()

	err = resolver.Wait(
		ctx,
		submittedJob.ID,
		1,
		job.WaitThrowErrors([]model.JobStateType{
			model.JobStateCancelled,
			model.JobStateError,
		}),
		job.WaitForJobStates(map[model.JobStateType]int{
			model.JobStateCompleted: 1,
		}),
	)
	require.NoError(suite.T(), err)

	shards, err := resolver.GetShards(ctx, submittedJob.ID)
	require.NoError(suite.T(), err)

	shard := shards[0]

	node, err := stack.GetNode(ctx, shard.NodeID)
	require.NoError(suite.T(), err)

	outputDir, err := ioutil.TempDir("", "bacalhau-ipfs-multiple-cid-test")
	require.NoError(suite.T(), err)
	require.NotEmpty(suite.T(), shard.PublishedResult.CID)

	outputPath := filepath.Join(outputDir, shard.PublishedResult.CID)
	err = node.IPFSClient.Get(ctx, shard.PublishedResult.CID, outputPath)
	require.NoError(suite.T(), err)

	stdout, err := os.ReadFile(fmt.Sprintf("%s/stdout", outputPath))
	require.NoError(suite.T(), err)

	// check that the stdout string containts the text hello-cid-1.txt and hello-cid-2.txt
	require.Contains(suite.T(), string(stdout), "hello-cid-1.txt")
	require.Contains(suite.T(), string(stdout), "hello-cid-2.txt")
}

func (suite *MultipleCIDSuite) TestMultipleURLs() {

	files := map[string]string{
		"/file1.txt": "Before you marry a person, you should first make them use a computer with slow Internet to see who they really are.\n",
		"/file2.txt": "I walk around like everything’s fine, but deep down, inside my shoe, my sock is sliding off.\n",
	}

	ctx := context.Background()

	stack, cm := SetupTest(
		ctx,
		suite.T(),
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

	j := &model.Job{}
	j.Spec = model.Spec{
		Engine:    model.EngineDocker,
		Verifier:  model.VerifierNoop,
		Publisher: model.PublisherIpfs,
		Docker: model.JobSpecDocker{
			Image: "ubuntu",
			Entrypoint: []string{
				"bash", "-c",
				"cat /inputs/hello-url-1.txt && cat /inputs/hello-url-2.txt",
			},
		},
	}
	j.Spec.Inputs = []model.StorageSpec{
		{
			StorageSource: model.StorageSourceURLDownload,
			URL:           fmt.Sprintf("%s/file1.txt", svr.URL),
			Path:          "/inputs/hello-url-1.txt",
		},
		{
			StorageSource: model.StorageSourceURLDownload,
			URL:           fmt.Sprintf("%s/file2.txt", svr.URL),
			Path:          "/inputs/hello-url-2.txt",
		},
	}
	j.Deal = model.Deal{Concurrency: 1}

	submittedJob, err := apiClient.Submit(ctx, j, nil)
	require.NoError(suite.T(), err)

	resolver := apiClient.GetJobStateResolver()

	err = resolver.Wait(
		ctx,
		submittedJob.ID,
		1,
		job.WaitThrowErrors([]model.JobStateType{
			model.JobStateCancelled,
			model.JobStateError,
		}),
		job.WaitForJobStates(map[model.JobStateType]int{
			model.JobStateCompleted: 1,
		}),
	)
	require.NoError(suite.T(), err)

	shards, err := resolver.GetShards(ctx, submittedJob.ID)
	require.NoError(suite.T(), err)

	shard := shards[0]

	node, err := stack.GetNode(ctx, shard.NodeID)
	require.NoError(suite.T(), err)

	outputDir, err := ioutil.TempDir("", "bacalhau-ipfs-multiple-url-test")
	require.NoError(suite.T(), err)
	require.NotEmpty(suite.T(), shard.PublishedResult.CID)

	outputPath := filepath.Join(outputDir, shard.PublishedResult.CID)
	err = node.IPFSClient.Get(ctx, shard.PublishedResult.CID, outputPath)
	require.NoError(suite.T(), err)

	stdout, err := os.ReadFile(fmt.Sprintf("%s/stdout", outputPath))
	require.NoError(suite.T(), err)

	require.Equal(suite.T(), files["/file1.txt"]+files["/file2.txt"], string(stdout))
}

func (suite *MultipleCIDSuite) TestIPFSURLCombo() {
	URLContent := "Common sense is like deodorant. The people who need it most never use it.\n"
	IPFSContent := "Truth hurts. Maybe not as much as jumping on a bicycle with a seat missing, but it hurts.\n"

	ctx := context.Background()

	stack, cm := SetupTest(
		ctx,
		suite.T(),
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

	cid, err := devstack.AddTextToNodes(ctx, []byte(IPFSContent), devstack.ToIPFSClients(stack.Nodes[:1])...)
	require.NoError(suite.T(), err)

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
				"cat /inputs/hello-url.txt && cat /inputs/hello-ipfs.txt",
			},
		},
	}
	j.Spec.Inputs = []model.StorageSpec{
		{
			StorageSource: model.StorageSourceURLDownload,
			URL:           svr.URL,
			Path:          "/inputs/hello-url.txt",
		},
		{
			StorageSource: model.StorageSourceIPFS,
			CID:           cid,
			Path:          "/inputs/hello-ipfs.txt",
		},
	}
	j.Deal = model.Deal{Concurrency: 1}

	submittedJob, err := apiClient.Submit(ctx, j, nil)
	require.NoError(suite.T(), err)

	resolver := apiClient.GetJobStateResolver()

	err = resolver.Wait(
		ctx,
		submittedJob.ID,
		1,
		job.WaitThrowErrors([]model.JobStateType{
			model.JobStateCancelled,
			model.JobStateError,
		}),
		job.WaitForJobStates(map[model.JobStateType]int{
			model.JobStateCompleted: 1,
		}),
	)
	require.NoError(suite.T(), err)

	shards, err := resolver.GetShards(ctx, submittedJob.ID)
	require.NoError(suite.T(), err)

	shard := shards[0]

	node, err := stack.GetNode(ctx, shard.NodeID)
	require.NoError(suite.T(), err)

	outputDir, err := ioutil.TempDir("", "bacalhau-ipfs-multiple-url-test")
	require.NoError(suite.T(), err)
	require.NotEmpty(suite.T(), shard.PublishedResult.CID)

	outputPath := filepath.Join(outputDir, shard.PublishedResult.CID)
	err = node.IPFSClient.Get(ctx, shard.PublishedResult.CID, outputPath)
	require.NoError(suite.T(), err)

	stdout, err := os.ReadFile(fmt.Sprintf("%s/stdout", outputPath))
	require.NoError(suite.T(), err)

	require.Equal(suite.T(), URLContent+IPFSContent, string(stdout))
}
