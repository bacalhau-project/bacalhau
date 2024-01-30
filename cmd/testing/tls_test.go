package cmdtesting

import (
	"fmt"
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
	_, out, err := s.ExecuteTestCobraCommand("list", "--tls",
		"--api-host", s.Host,
		"--api-port", fmt.Sprint(s.Port))

	s.Require().NoError(err, "failed to execute Cobra Command")
	//TO DO: Change ExecuteTestCobraCommand to pass the certificate verification error
	s.Require().Contains(out, "tls: failed to verify certificate")
}

func (s *TLSSuite) TestTLSWithInsecureFlag() {
	_, out, err := s.ExecuteTestCobraCommand("list", "--tls", "--insecure",
		"--api-host", s.Host,
		"--api-port", fmt.Sprint(s.Port))

	s.Require().NoError(err, "failed to execute Cobra Command")
	s.Require().Contains(out, "CREATED", "ID", "JOB", "STATE", "PUBLISHED")
}

func (s *TLSSuite) TestTLSWithCACert() {
	cacertFilepath := "../../testdata/certs/dev-ca.crt"
	_, out, err := s.ExecuteTestCobraCommand("list", "--tls", "--cacert", cacertFilepath,
		"--api-host", s.Host,
		"--api-port", fmt.Sprint(s.Port))
	s.Require().NoError(err, "failed to execute Cobra Command")
	s.Require().Contains(out, "CREATED", "ID", "JOB", "STATE", "PUBLISHED")
}
