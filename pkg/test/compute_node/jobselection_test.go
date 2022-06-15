package compute_node

import (
	"context"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/compute_node"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/stretchr/testify/assert"
)

// test that when we have RejectStatelessJobs turned on
// we don't accept a job with no volumes
// but when it's not turned on the job is actually selected
func TestJobSelectionNoVolumes(t *testing.T) {
	runTest := func(rejectSetting, expectedResult bool) {
		computeNode, _, cm := SetupTest(t, compute_node.JobSelectionPolicy{
			RejectStatelessJobs: rejectSetting,
		})
		defer cm.Cleanup()

		result, err := computeNode.SelectJob(context.Background(), "requester_id", GetJobSpec(""))
		assert.NoError(t, err)
		assert.Equal(t, result, expectedResult)
	}

	runTest(true, false)
	runTest(false, true)
}

func TestJobSelectionLocality(t *testing.T) {

	// get the CID so we can use it in the tests below but without it actually being
	// added to the server (so we can test locality anywhere)
	EXAMPLE_TEXT := "hello"
	cid, err := (func() (string, error) {
		_, ipfsStack, cm := SetupTest(t, compute_node.JobSelectionPolicy{})
		defer cm.Cleanup()
		return ipfsStack.AddTextToNodes(1, []byte(EXAMPLE_TEXT))
	}())
	assert.NoError(t, err)

	runTest := func(locality compute_node.JobSelectionDataLocality, shouldAddData, expectedResult bool) {

		computeNode, ipfsStack, cm := SetupTest(t, compute_node.JobSelectionPolicy{
			Locality: locality,
		})
		defer cm.Cleanup()

		if shouldAddData {
			_, err := ipfsStack.AddTextToNodes(1, []byte(EXAMPLE_TEXT))
			assert.NoError(t, err)
		}

		result, err := computeNode.SelectJob(context.Background(), "requester_id", GetJobSpec(cid))
		assert.NoError(t, err)
		assert.Equal(t, result, expectedResult)
	}

	// we are local - we do have the file - we should accept
	runTest(compute_node.Local, true, true)

	// we are local - we don't have the file - we should reject
	runTest(compute_node.Local, false, false)

	// we are anywhere - we do have the file - we should accept
	runTest(compute_node.Anywhere, true, true)

	// we are anywhere - we don't have the file - we should accept
	runTest(compute_node.Anywhere, false, true)
}
