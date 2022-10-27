//go:build !(unit && (windows || darwin))

package devstack

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"sync"
	"testing"
	"time"

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
func (s *MultipleCIDSuite) SetupSuite() {

}

// Before each test
func (s *MultipleCIDSuite) SetupTest() {
	err := system.InitConfigForTesting()
	require.NoError(s.T(), err)
}

func (suite *MultipleCIDSuite) TearDownTest() {
}

func (s *MultipleCIDSuite) TearDownSuite() {

}

func (s *MultipleCIDSuite) TestMultipleCIDs() {
	ctx := context.Background()

	dirCID1 := "/input-1"
	dirCID2 := "/input-2"

	fileName1 := "hello-cid-1.txt"
	fileName2 := "hello-cid-2.txt"

	stack, cm := SetupTest(
		ctx,
		s.T(),
		1,
		0,
		false,
		computenode.NewDefaultComputeNodeConfig(),
	)

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
				"bash",
				"-c",
				fmt.Sprintf("ls && ls %s && ls %s", dirCID1, dirCID2),
			},
		},
	}
	j.Spec.Inputs = []model.StorageSpec{
		{
			StorageSource: model.StorageSourceIPFS,
			CID:           fileCid1,
			Path:          path.Join(dirCID1, fileName1),
		},
		{
			StorageSource: model.StorageSourceIPFS,
			CID:           fileCid2,
			Path:          path.Join(dirCID2, fileName2),
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

	outputDir := s.T().TempDir()
	require.NotEmpty(s.T(), shard.PublishedResult.CID)

	outputPath := filepath.Join(outputDir, shard.PublishedResult.CID)
	err = node.IPFSClient.Get(ctx, shard.PublishedResult.CID, outputPath)
	require.NoError(s.T(), err)

	stdout, err := os.ReadFile(fmt.Sprintf("%s/stdout", outputPath))
	require.NoError(s.T(), err)

	// check that the stdout string containts the text hello-cid-1.txt and hello-cid-2.txt
	require.Contains(s.T(), string(stdout), fileName1)
	require.Contains(s.T(), string(stdout), fileName2)
}

type URLBasedTestCase struct {
	file1  string
	file2  string
	mount1 string
	mount2 string
	files  map[string]string
}

func runURLTest(
	t *testing.T,
	handler func(w http.ResponseWriter, r *http.Request),
	testCase URLBasedTestCase,
) {
	ctx := context.Background()

	stack, cm := SetupTest(
		ctx,
		t,
		1,
		0,
		false,
		computenode.ComputeNodeConfig{
			JobSelectionPolicy: computenode.JobSelectionPolicy{
				Locality: computenode.Anywhere,
			},
		},
	)

	ctx, rootSpan := system.NewRootSpan(ctx, system.GetTracer(), "pkg/test/devstack/multiple_cid_test/testmultipleurls")
	defer rootSpan.End()
	cm.RegisterCallback(system.CleanupTraceProvider)

	svr := httptest.NewServer(http.HandlerFunc(handler))
	defer svr.Close()

	apiUri := stack.Nodes[0].APIServer.GetURI()
	apiClient := publicapi.NewAPIClient(apiUri)

	entrypoint := []string{
		"bash", "-c",
		fmt.Sprintf("cat %s/%s && cat %s/%s",
			testCase.mount1, testCase.file1,
			testCase.mount2, testCase.file2),
	}
	j := model.NewJob()
	j.Spec = model.Spec{
		Engine:    model.EngineDocker,
		Verifier:  model.VerifierNoop,
		Publisher: model.PublisherIpfs,
		Docker: model.JobSpecDocker{
			Image:      "ubuntu",
			Entrypoint: entrypoint,
		},
	}
	j.Spec.Inputs = []model.StorageSpec{
		{
			StorageSource: model.StorageSourceURLDownload,
			URL:           fmt.Sprintf("%s/%s", svr.URL, testCase.file1),
			Path:          testCase.mount1,
		},
		{
			StorageSource: model.StorageSourceURLDownload,
			URL:           fmt.Sprintf("%s/%s", svr.URL, testCase.file2),
			Path:          testCase.mount2,
		},
	}
	j.Deal = model.Deal{Concurrency: 1}

	submittedJob, err := apiClient.Submit(ctx, j, nil)
	require.NoError(t, err)

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
	require.NoError(t, err)

	outputDir := t.TempDir()

	shards, err := resolver.GetShards(ctx, submittedJob.ID)
	require.NoError(t, err)
	require.True(t, len(shards) > 0, "No shards created during submit job.")

	shard := shards[0]
	require.NotEmpty(t, shard.PublishedResult.CID)

	node, err := stack.GetNode(ctx, shard.NodeID)
	require.NoError(t, err)

	outputPath := filepath.Join(outputDir, shard.PublishedResult.CID)
	err = node.IPFSClient.Get(ctx, shard.PublishedResult.CID, outputPath)
	require.NoError(t, err)
	require.FileExists(t, fmt.Sprintf("%s/stdout", outputPath))

	stdout, err := os.ReadFile(fmt.Sprintf("%s/stdout", outputPath))
	log.Debug().Str("stdout", string(stdout)).Msg("stdout")
	require.NoError(t, err)

	require.Equal(t, testCase.files[fmt.Sprintf("/%s", testCase.file1)]+
		testCase.files[fmt.Sprintf("/%s", testCase.file2)],
		string(stdout))
}

func getSimpleTestCase() URLBasedTestCase {
	file1 := "hello-cid-1.txt"
	file2 := "hello-cid-2.txt"
	return URLBasedTestCase{
		file1:  file1,
		file2:  file2,
		mount1: "/inputs-1",
		mount2: "/inputs-2",
		files: map[string]string{
			fmt.Sprintf("/%s", file1): "Before you marry a person, you should first make them use a computer with slow Internet to see who they really are.\n",
			fmt.Sprintf("/%s", file2): "I walk around like everything's fine, but deep down, inside my shoe, my sock is sliding off.\n",
		},
	}
}

func (s *MultipleCIDSuite) TestMultipleURLs() {
	testCase := getSimpleTestCase()
	handler := func(w http.ResponseWriter, r *http.Request) {
		content, ok := testCase.files[r.URL.Path]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("not found"))
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(content))
		}
	}
	runURLTest(s.T(), handler, testCase)
}

// both starts should be before both ends if we are downloading in parallel
func (s *MultipleCIDSuite) TestURLsInParallel() {
	mutex := sync.Mutex{}
	testCase := getSimpleTestCase()

	accessTimes := map[string]int64{}
	getAccessTime := func() int64 {
		return time.Now().UnixNano() / int64(time.Millisecond)
	}
	getAccessKey := func(filename, append string) string {
		return fmt.Sprintf("%s_%s", filename, append)
	}
	setAccessTime := func(key string) {
		mutex.Lock()
		defer mutex.Unlock()
		accessTimes[key] = getAccessTime()
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		setAccessTime(getAccessKey(r.URL.Path, "start"))
		time.Sleep(time.Second * 1)
		setAccessTime(getAccessKey(r.URL.Path, "end"))
		content, ok := testCase.files[r.URL.Path]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("not found"))
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(content))
		}

	}
	runURLTest(s.T(), handler, testCase)

	start1, ok := accessTimes["/"+getAccessKey(testCase.file1, "start")]
	require.True(s.T(), ok)
	start2, ok := accessTimes["/"+getAccessKey(testCase.file2, "start")]
	require.True(s.T(), ok)
	end1, ok := accessTimes["/"+getAccessKey(testCase.file1, "end")]
	require.True(s.T(), ok)
	end2, ok := accessTimes["/"+getAccessKey(testCase.file2, "end")]
	require.True(s.T(), ok)

	require.True(s.T(), start2 < end1, "start 2 should be before end 1")
	require.True(s.T(), start1 < end2, "start 1 should be before end 2")
}

func (s *MultipleCIDSuite) TestFlakyURLs() {
	mutex := sync.Mutex{}
	testCase := getSimpleTestCase()
	accessCounter := map[string]int{}
	increaseCounter := func(key string) int {
		mutex.Lock()
		defer mutex.Unlock()
		accessCount, ok := accessCounter[key]
		if !ok {
			accessCount = 0
		}
		accessCount++
		accessCounter[key] = accessCount
		return accessCount
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		accessCounts := increaseCounter(r.URL.Path)
		if accessCounts < 3 {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("not found"))
			return
		}
		content, ok := testCase.files[r.URL.Path]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("not found"))
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(content))
		}

	}
	runURLTest(s.T(), handler, testCase)
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
		false,
		computenode.ComputeNodeConfig{
			JobSelectionPolicy: computenode.JobSelectionPolicy{
				Locality: computenode.Anywhere,
			},
		},
	)

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
				fmt.Sprintf("cat %s && cat %s",
					path.Join(urlmount, urlfile),
					path.Join(ipfsmount, ipfsfile),
				),
			},
		},
	}
	j.Spec.Inputs = []model.StorageSpec{
		{
			StorageSource: model.StorageSourceURLDownload,
			URL:           fmt.Sprintf("%s/%s", svr.URL, urlfile),
			Path:          urlmount,
		},
		{
			StorageSource: model.StorageSourceIPFS,
			CID:           cid,
			Path:          path.Join(ipfsmount, ipfsfile),
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

	outputDir := s.T().TempDir()
	require.NotEmpty(s.T(), shard.PublishedResult.CID)

	outputPath := filepath.Join(outputDir, shard.PublishedResult.CID)
	err = node.IPFSClient.Get(ctx, shard.PublishedResult.CID, outputPath)
	require.NoError(s.T(), err)

	stdout, err := os.ReadFile(fmt.Sprintf("%s/stdout", outputPath))
	require.NoError(s.T(), err)

	require.Equal(s.T(), URLContent+IPFSContent, string(stdout))
}
