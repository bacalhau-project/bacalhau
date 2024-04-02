//go:build unit || !integration

package cmdtesting

import (
	"path/filepath"
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
	cacertFilepath, err := filepath.Abs("../../testdata/certs/dev-ca.crt")
	s.Require().NoError(err)
	_, _, err = s.ExecuteTestCobraCommand("list", "--tls", "--cacert", cacertFilepath)
	s.Require().NoError(err, "failed to execute Cobra Command")
}
