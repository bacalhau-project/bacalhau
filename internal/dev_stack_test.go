package internal

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/internal/ipfs"
	"github.com/stretchr/testify/assert"
)

func runTest(t *testing.T, stack *DevStack) error {

	// create test data
	// ipfs add file on 2 nodes
	// submit job on 1 node
	// wait for job to be done
	// download results and check sanity

	testDir, err := ioutil.TempDir("", "bacalhau-test")
	if err != nil {
		return err
	}

	testFilePath := fmt.Sprintf("%s/test.txt", testDir)

	dataBytes := []byte(`apple
orange
pineapple
pear
peach
cherry
kiwi
strawberry
lemon
raspberry
`)

	err = os.WriteFile(testFilePath, dataBytes, 0644)
	if err != nil {
		return err
	}

	// ipfs add the file to 2 nodes
	// this tests self selection
	for i, node := range stack.Nodes {
		if i >= 2 {
			continue
		}

		addResults, err := ipfs.IpfsCommand(node.IpfsRepo, []string{
			"add", "-Q", testFilePath,
		})

		if err != nil {
			return err
		}

		fmt.Printf("ipfs add results: %s\n", addResults)
	}

	return nil
}

func TestDevStack(t *testing.T) {

	ctx := context.Background()
	ctxWithCancel, cancelFunction := context.WithCancel(ctx)
	defer cancelFunction()

	os.Setenv("DEBUG", "true")

	stack, err := NewDevStack(ctxWithCancel, 3)
	assert.NoError(t, err)
	if err != nil {
		log.Fatalf("Unable to create devstack: %s", err)
	}

	// we need a better method for this - i.e. waiting for all the ipfs nodes to be ready
	time.Sleep(time.Second * 10)

	fmt.Printf("Running tests...\n")

	err = runTest(t, stack)

	if err != nil {
		fmt.Printf("Test error: %s...\n", err)
	}

	assert.NoError(t, err)

}
