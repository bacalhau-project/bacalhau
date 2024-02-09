package util_test

import (
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/util"
)

func TestGenerateCertificates(t *testing.T) {
	instance := util.NewCertificateParams()
	util.GenerateCertificates(instance)
}
