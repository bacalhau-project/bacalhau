package utils

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	dc "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/stretchr/testify/suite"
)

func buildDockerImage(dockerfilePath, imageName, imageTag string) error {
	cli, err := dc.NewClientWithOpts(dc.FromEnv)
	if err != nil {
		return fmt.Errorf("failed to build docker image %q: failed to create Docker client: %v", imageName, err)
	}

	// Create a tar archive of the build context
	tar, err := archive.TarWithOptions(".", &archive.TarOptions{})
	if err != nil {
		return fmt.Errorf("failed to build docker image %q: failed to create tar archive: %v", imageName, err)
	}
	defer tar.Close()

	// Set build options
	opts := types.ImageBuildOptions{
		Dockerfile: dockerfilePath,
		Tags:       []string{fmt.Sprintf("%s:%s", imageName, imageTag)},
		Remove:     true,
	}

	// Build the image
	resp, err := cli.ImageBuild(context.Background(), tar, opts)
	if err != nil {
		return fmt.Errorf("failed to build docker image %q: failed to build image: %v", imageName, err)
	}
	defer resp.Body.Close()

	// Print build output
	_, err = io.Copy(os.Stdout, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to build docker image %q: failed to read build output: %v", imageName, err)
	}

	// Verify that the image was really built
	_, err = cli.ImageInspect(context.Background(), fmt.Sprintf("%s:%s", imageName, imageTag))
	if err != nil {
		return fmt.Errorf("unable to find image  %s:%s. Error: %v", imageName, imageTag, err)
	}

	fmt.Printf("image %s:%s built successfully\n", imageName, imageTag)
	return nil
}

func DeleteDockerTestImagesAndPrune(suffix string) error {
	// Create a new Docker client
	ctx := context.Background()
	cli, err := dc.NewClientWithOpts(dc.FromEnv)
	if err != nil {
		return fmt.Errorf("deleting docker images: failed to create Docker client: %v", err)
	}
	defer cli.Close()

	// List all imagesList
	imagesList, err := cli.ImageList(ctx, image.ListOptions{All: true})
	if err != nil {
		return fmt.Errorf("deleting docker images: failed to list imagesList: %v", err)
	}

	// Iterate through the imagesList and delete those that match the suffix
	for _, currentImage := range imagesList {
		for _, tag := range currentImage.RepoTags {
			if strings.HasSuffix(tag, suffix) {
				fmt.Printf("Deleting currentImage: %s with ID: %s\n", tag, currentImage.ID)
				_, err := cli.ImageRemove(context.Background(), currentImage.ID, image.RemoveOptions{Force: true, PruneChildren: true})
				if err != nil {
					fmt.Printf("deleting docker images: Failed to delete currentImage %s: %v\n", tag, err)
				}
				// Break after deleting to avoid trying to delete the same currentImage multiple times
				break
			}
		}
	}

	pruneReport, err := cli.ImagesPrune(ctx, filters.NewArgs())
	if err != nil {
		return fmt.Errorf("deleting docker images: failed to prune images: %v", err)
	}
	fmt.Printf(
		"Pruned %d images, reclaimed space: %d bytes\n",
		len(pruneReport.ImagesDeleted),
		pruneReport.SpaceReclaimed,
	)
	return nil
}

func BuildBaseImages(testIdentifier string) error {
	err := buildDockerImage(
		"common_assets/dockerfiles/Dockerfile-JumpboxNode",
		"bacalhau-test-jumpbox-"+testIdentifier,
		testIdentifier,
	)
	if err != nil {
		return fmt.Errorf("error creating the bacalhau-test-jumpbox image: %v", err)
	}

	err = buildDockerImage(
		"common_assets/dockerfiles/Dockerfile-ComputeNode",
		"bacalhau-test-compute-"+testIdentifier,
		testIdentifier,
	)
	if err != nil {
		return fmt.Errorf("error creating the bacalhau-test-compute image: %v", err)
	}

	err = buildDockerImage(
		"common_assets/dockerfiles/Dockerfile-DockerImageRegistryNode",
		"bacalhau-test-registry-"+testIdentifier,
		testIdentifier,
	)
	if err != nil {
		return fmt.Errorf("error creating the bacalhau-test-registry image: %v", err)
	}

	err = buildDockerImage(
		"common_assets/dockerfiles/Dockerfile-OrchestratorNode",
		"bacalhau-test-orchestrator-"+testIdentifier,
		testIdentifier,
	)
	if err != nil {
		return fmt.Errorf("error creating the bacalhau-test-orchestrator image: %v", err)
	}

	err = buildDockerImage(
		"common_assets/dockerfiles/Dockerfile-TraefikNode",
		"bacalhau-test-traefik-"+testIdentifier,
		testIdentifier,
	)
	if err != nil {
		return fmt.Errorf("error creating the bacalhau-test-traefik image: %v", err)
	}

	err = buildDockerImage(
		"common_assets/dockerfiles/Dockerfile-UbuntuOrchestratorNode",
		"bacalhau-test-ubuntu-orchestrator-"+testIdentifier,
		testIdentifier,
	)
	if err != nil {
		return fmt.Errorf("error creating the bacalhau-test-ubuntu-orchestrator image: %v", err)
	}

	return nil
}

func SetTestGlobalEnvVariables(additionalSetupEnvVars map[string]string) error {
	defaultEnvVars := map[string]string{
		"TESTCONTAINERS_RYUK_DISABLED":                  "false",
		"TESTCONTAINERS_RYUK_CONTAINER_PRIVILEGED":      "true",
		"TESTCONTAINERS_RYUK_CONTAINER_STARTUP_TIMEOUT": "7m",
		"TESTCONTAINERS_RYUK_CONNECTION_TIMEOUT":        "10m",
		"TESTCONTAINERS_RYUK_RECONNECTION_TIMEOUT":      "7m",
		"TESTCONTAINERS_RYUK_VERBOSE":                   "true",
		"DOCKER_API_VERSION":                            "1.45",
	}

	// Merge additional env vars with default ones
	if additionalSetupEnvVars != nil {
		for key, value := range additionalSetupEnvVars {
			defaultEnvVars[key] = value
		}
	}

	for key, value := range defaultEnvVars {
		err := os.Setenv(key, value)
		if err != nil {
			return err
		}
	}

	return nil
}

func ExtractJobIDFromOutput(jobRunOutput string, s *suite.Suite) string {
	s.Require().Regexpf(
		`Job successfully submitted\. Job ID: j-[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`,
		jobRunOutput,
		"Job output did not signal a successful job submission: %q",
		jobRunOutput,
	)

	re := regexp.MustCompile(`j-[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)
	jobID := re.FindString(jobRunOutput)

	s.Require().NotEmpty(jobID, "Job ID cannot be empty", jobID)
	s.Require().Regexp(`^j-[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`,
		jobID,
		"Extracted Job ID should match the expected format. Job ID found: "+jobID,
	)

	return jobID
}

func ExtractJobIDFromShortOutput(input string) (string, error) {
	pattern := `j-[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`

	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", err
	}

	match := re.FindString(input)
	if match == "" {
		return "", nil
	}

	return strings.TrimSpace(match), nil
}

func ExtractJobStateType(jobDescriptionJsonOutput string) (string, error) {
	// Define a struct that matches the structure of the JSON
	var data struct {
		Job struct {
			State struct {
				StateType string `json:"StateType"`
			} `json:"State"`
		} `json:"Job"`
	}

	// Cleanup the Json output. Unfortunate that the CLI prints extra
	// characters at the beginning and at the end
	startIndex := strings.Index(jobDescriptionJsonOutput, "{")
	endIndex := strings.LastIndex(jobDescriptionJsonOutput, "}")
	if startIndex == -1 || endIndex == -1 || startIndex > endIndex {
		return "", errors.New("invalid JSON structure for Job Description")
	}

	cleanedJsonJobDescription := jobDescriptionJsonOutput[startIndex : endIndex+1]

	// Unmarshal the JSON string into our struct
	err := json.Unmarshal([]byte(cleanedJsonJobDescription), &data)
	if err != nil {
		return "", err
	}

	return data.Job.State.StateType, nil
}
