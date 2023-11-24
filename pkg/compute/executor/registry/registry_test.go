//go:build unit || !integration

package registry_test

import (
	"path"
	"path/filepath"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/compute/executor/registry"
	"github.com/stretchr/testify/suite"
)

type RegistryTestSuite struct {
	suite.Suite
}

func TestRegistryTestSuite(t *testing.T) {
	suite.Run(t, new(RegistryTestSuite))
}

func lengthenPath(dir string) string {
	return path.Join("../../../../testdata/plugins/executors", dir)
}

func (s *RegistryTestSuite) TestConfigLoading() {
	type testcase struct {
		testname             string
		directory            string
		expectedErrorStrings []string
	}

	testcases := []testcase{
		{
			testname:             "Only a name",
			directory:            lengthenPath("empty"),
			expectedErrorStrings: []string{"path to executable is required"},
		},
		{
			testname:             "Duplicate name",
			directory:            lengthenPath("duplicate"),
			expectedErrorStrings: []string{"name 'test' already registered"},
		},
		{
			testname:             "Success",
			directory:            lengthenPath("success"),
			expectedErrorStrings: []string{},
		},
	}

	registry := registry.New()

	for _, tc := range testcases {
		s.Run(tc.testname, func() {
			pluginPath, _ := filepath.Abs(tc.directory)
			err := registry.Load(pluginPath)
			if len(tc.expectedErrorStrings) == 0 {
				s.Require().NoError(err)
				return
			}

			s.Require().Error(err)
			for _, e := range tc.expectedErrorStrings {
				s.Require().Contains(err.Error(), e)
			}
		})
	}

}
