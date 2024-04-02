//go:build unit || !integration

package cmdtesting

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestTLSSuite(t *testing.T) {
	suite.Run(t, new(TLSSuite))
}

type TLSSuite struct {
	BaseTLSSuite
}

func (s *TLSSuite) TestTLSflagWithSelfSignedCertificate() {
	_, _, err := s.ExecuteTestCobraCommand("list", "--tls")
	s.Require().Error(err)
}

func (s *TLSSuite) TestTLSWithInsecureFlag() {
	_, _, err := s.ExecuteTestCobraCommand("list", "--tls", "--insecure")

	s.Require().NoError(err)
}

func (s *TLSSuite) TestTLSWithCACert() {
	_, _, err := s.ExecuteTestCobraCommand("list", "--tls", "--cacert", s.TempCACertFilePath)
	s.Require().NoError(err, "failed to execute Cobra Command")
}
