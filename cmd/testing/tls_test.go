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
	// NB(forrest): we expect an error here because we are using self signed certs
	// and not providing the client command with the cacert making the connection 'insecure'
	// in this command we don't provide the --insecure flag or the certificate file (as done in below commands)
	// meaning we expect an error.
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
