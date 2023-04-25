//go:build unit || !integration

package docker

import (
	"context"
	"fmt"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ImageResolverSuite struct {
	suite.Suite
}

func TestImageResolverSuite(t *testing.T) {
	suite.Run(t, new(ImageResolverSuite))
}

func errorResolver(c context.Context, i string, creds config.DockerCredentials) (string, error) {
	return "", fmt.Errorf("an error occurred")
}

func fullResolver() imageResolverFunc {
	client, _ := NewDockerClient()
	return client.ImageDigest
}

func valueResolver(val string) imageResolverFunc {
	return func(c context.Context, i string, creds config.DockerCredentials) (string, error) {
		return fmt.Sprintf("sha256:%s", val), nil
	}
}

func (s *ImageResolverSuite) TestResolverCases() {

	type testcase struct {
		name        string
		image       string
		initial_tag string
		error       bool
		expected    string
		resolver    imageResolverFunc
	}

	ctx := context.Background()

	testcases := []testcase{
		{
			name:        "error leaves image intact",
			image:       "ubuntu:latest",
			initial_tag: "latest",
			error:       true,
			expected:    "ubuntu:latest",
			resolver:    errorResolver,
		},
		{
			name:        "nothing",
			image:       "ubuntu",
			initial_tag: "",
			error:       false,
			expected:    "ubuntu@sha256:hash",
			resolver:    valueResolver("hash"),
		},
		{
			name:        "already digested",
			image:       "ubuntu@sha256:something",
			initial_tag: "sha256:something",
			error:       false,
			expected:    "ubuntu@sha256:something",
			resolver:    errorResolver,
		},
		{
			name:        "repo but no digest",
			image:       "ghcr.io/org/user/ubuntu:latest",
			initial_tag: "latest",
			error:       false,
			expected:    "ghcr.io/org/user/ubuntu@sha256:hash",
			resolver:    valueResolver("hash"),
		},
		{
			name:        "name no tag",
			image:       "ubuntu",
			initial_tag: "",
			error:       false,
			expected:    "ubuntu@sha256:hash",
			resolver:    valueResolver("hash"),
		},
		// {
		// 	name:        "remote resolver",
		// 	image:       "ubuntu:kinetic",
		// 	initial_tag: "kinetic",
		// 	error:       false,
		// 	expected:    "ubuntu@sha256:a9a425d086dbb34c1b5b99765596e2a3cc79b33826866c51cd4508d8eb327d2b",
		// 	resolver:    fullResolver(),
		// },
	}

	for _, tc := range testcases {
		s.Run(tc.name, func() {
			i := NewImageID(tc.image)
			require.Equal(s.T(), tc.initial_tag, i.tag.String())

			newImageID, err := ResolveImageID(ctx, i, tc.resolver)
			if tc.error {
				require.Error(s.T(), err)
			} else {
				require.NoError(s.T(), err)
			}

			require.Equal(s.T(), tc.expected, newImageID.String())
		})
	}

}
