package system

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type SystemContextSuite struct {
	suite.Suite
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestSystemContextSuite(t *testing.T) {
	suite.Run(t, new(SystemContextSuite))
}

// Before all suite
func (suite *SystemContextSuite) SetupAllSuite() {

}

// Before each test
func (suite *SystemContextSuite) SetupTest() {
	require.NoError(suite.T(), InitConfigForTesting())
}

func (suite *SystemContextSuite) TearDownTest() {
}

func (suite *SystemContextSuite) TearDownAllSuite() {

}

func TestOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := make(chan struct{}, 1)
	seenHandler := false
	OnCancel(ctx, func() {
		seenHandler = true
		ch <- struct{}{}
	})

	cancel()
	<-ch
	require.True(t, seenHandler, "OnCancel() callback not called")
}
