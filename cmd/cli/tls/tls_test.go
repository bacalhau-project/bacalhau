package tls_test

import (
	"fmt"
	"testing"

	cmdtesting "github.com/bacalhau-project/bacalhau/cmd/testing"
	"github.com/stretchr/testify/suite"
)

func TestTLSSuite(t *testing.T) {
	suite.Run(t, new(TLSSuite))
}

type TLSSuite struct {
	cmdtesting.BaseTLSSuite
}

func (s *TLSSuite) TestInsecureOutput() {
	_, out, err := cmdtesting.ExecuteTestCobraCommand("version", "--tls",
		"--api-host", s.Host,
		"--api-port", fmt.Sprint(s.Port))
	if err != nil {
		fmt.Printf("OLGIBBONS ERROR: %#v", err)
	}
	fmt.Printf("OLGIBBONS out: %#v", out)
	//aliveInfo := &apimodels.IsAliveResponse{}
	//err = marshaller.JSONUnmarshalWithMax([]byte(out), &aliveInfo)
	//s.Require().NoError(err, "Could not unmarshall the output into json - %+v", err)
	//s.Require().True(aliveInfo.IsReady())
}
