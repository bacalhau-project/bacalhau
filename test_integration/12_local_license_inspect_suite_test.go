package test_integration

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

type LocalLicenseInspectSuite struct {
	BaseDockerComposeTestSuite
}

func NewLocalLicenseInspectSuite() *LocalLicenseInspectSuite {
	s := &LocalLicenseInspectSuite{}
	s.GlobalRunIdentifier = globalTestExecutionId
	s.SuiteRunIdentifier = strings.ToLower(strings.Split(uuid.New().String(), "-")[0])
	return s
}

func (s *LocalLicenseInspectSuite) SetupSuite() {
	rawDockerComposeFilePath := "./common_assets/docker_compose_files/orchestrator-node-with-custom-start-command.yml"
	s.Context, s.Cancel = context.WithCancel(context.Background())

	orchestratorConfigFile := s.commonAssets("nodes_configs/12_basic_orchestrator_config.yaml")
	orchestratorStartCommand := fmt.Sprintf("bacalhau serve --config=%s", orchestratorConfigFile)
	extraRenderingData := map[string]interface{}{
		"OrchestratorStartCommand": orchestratorStartCommand,
	}
	s.BaseDockerComposeTestSuite.SetupSuite(rawDockerComposeFilePath, extraRenderingData)
}

func (s *LocalLicenseInspectSuite) TearDownSuite() {
	s.T().Log("Tearing down [Test Suite] in LocalLicenseInspectSuite...")
	s.BaseDockerComposeTestSuite.TearDownSuite()
}

func (s *LocalLicenseInspectSuite) TestValidateLocalLicense() {
	licenseFile := s.commonAssets("licenses/test-license.json")

	licenseInspectionOutput, err := s.executeCommandInDefaultJumpbox(
		[]string{
			"bacalhau", "license", "inspect", licenseFile,
		},
	)
	s.Require().NoErrorf(err, "Error inspecting license: %q", err)

	expectedOutput := `Product      = Bacalhau
License ID   = e66d1f3a-a8d8-4d57-8f14-00722844afe2
Customer ID  = test-customer-id-123
Valid Until  = 2045-07-28
Version      = v1
Capabilities = max_nodes=1
Metadata     = {}`

	s.Require().Contains(licenseInspectionOutput, expectedOutput)
}

func (s *LocalLicenseInspectSuite) TestValidateLocalLicenseJSONOutput() {
	licenseFile := s.commonAssets("licenses/test-license.json")

	licenseInspectionOutput, err := s.executeCommandInDefaultJumpbox(
		[]string{
			"bacalhau",
			"license",
			"inspect",
			licenseFile,
			"--output=json",
		},
	)
	s.Require().NoErrorf(err, "Error inspecting license: %q", err)

	output, err := s.convertStringToDynamicJSON(licenseInspectionOutput)
	s.Require().NoError(err)

	productName, err := output.Query("$.product")
	s.Require().NoError(err)
	s.Require().Equal("Bacalhau", productName.String())

	licenseID, err := output.Query("$.license_id")
	s.Require().NoError(err)
	s.Require().Equal("e66d1f3a-a8d8-4d57-8f14-00722844afe2", licenseID.String())

	customerID, err := output.Query("$.customer_id")
	s.Require().NoError(err)
	s.Require().Equal("test-customer-id-123", customerID.String())

	licenseVersion, err := output.Query("$.license_version")
	s.Require().NoError(err)
	s.Require().Equal("v1", licenseVersion.String())

	capabilitiesMaxNodes, err := output.Query("$.capabilities.max_nodes")
	s.Require().NoError(err)
	s.Require().Equal("1", capabilitiesMaxNodes.String())
}

func (s *LocalLicenseInspectSuite) TestInValidateLocalLicense() {
	licenseFile := s.commonAssets("licenses/test-license-invalid.json")

	_, err := s.executeCommandInDefaultJumpbox(
		[]string{
			"bacalhau", "license", "inspect", licenseFile,
		},
	)

	s.Require().ErrorContains(err, "invalid license: failed to parse license: token signature is invalid")
}

func TestLocalLicenseInspectSuite(t *testing.T) {
	suite.Run(t, NewLocalLicenseInspectSuite())
}
