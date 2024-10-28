package test_integration

import (
	"bacalhau/integration_tests/utils"
	"github.com/google/uuid"
	"log"
	"os"
	"strings"
	"testing"
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

	//ctx := context.Background()
	//err = utils.CompileBacalhau(ctx, "../main.go")
	//if err != nil {
	//	log.Println("Error compiling the bacalhau binary: ", err.Error())
	//	os.Exit(1)
	//}
	//
	//// TODO: Maybe we do not need to created images, but just inject
	//// TODO: them with artifacts before container starts the starts (certs and binary and configs)
	//err = utils.BuildBaseImages(globalTestExecutionId)
	//if err != nil {
	//	log.Println("Error building base images: ", err.Error())
	//	os.Exit(1)
	//}

	exitCode := m.Run()

	err = utils.DeleteDockerTestImagesAndPrune(globalTestExecutionId)
	if err != nil {
		log.Println("Error cleaning up base images: ", err.Error())
		os.Exit(1)
	}

	// TODO: Better cleaning
	//os.Remove("./common_assets/bacalhau_bin")
	//Exit with the same code as the test run
	os.Exit(exitCode)
}
