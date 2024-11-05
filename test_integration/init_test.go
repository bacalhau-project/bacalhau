package test_integration

import (
	"bacalhau/integration_tests/utils"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
)

var globalTestExecutionId string

func TestMain(m *testing.M) {
	globalTestExecutionId = strings.ToLower(strings.Split(uuid.New().String(), "-")[0])
	log.Println("====> Starting the whole test flow: ", globalTestExecutionId)

	err := utils.SetTestGlobalEnvVariables(map[string]string{})
	if err != nil {
		log.Println("Error Setting up Test Env Variables: ", err.Error())
		os.Exit(1)
	}

	err = utils.BuildBaseImages(globalTestExecutionId)
	if err != nil {
		log.Println("Error building base images: ", err.Error())
		os.Exit(1)
	}

	exitCode := m.Run()

	err = utils.DeleteDockerTestImagesAndPrune(globalTestExecutionId)
	if err != nil {
		log.Println("Error cleaning up base images: ", err.Error())
		os.Exit(1)
	}

	//Exit with the same code as the test run
	os.Exit(exitCode)
}
