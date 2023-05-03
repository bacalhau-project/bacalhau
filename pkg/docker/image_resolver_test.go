//go:build unit || !integration

package docker

import (
	"context"
	"fmt"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/cache"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ImageResolverSuite struct {
	suite.Suite
}

func TestImageResolverSuite(t *testing.T) {
	MustHaveDocker(t)

	suite.Run(t, new(ImageResolverSuite))
}

func errorResolver(c context.Context, i string, forceRemote bool, creds config.DockerCredentials) (*ImageManifest, error) {
	return nil, fmt.Errorf("an error occurred")
}

func fullResolver() imageResolverFunc {
	client, _ := NewDockerClient()
	return client.ImageDistribution
}

func valueResolver(val string) imageResolverFunc {
	return func(c context.Context, i string, forceRemote bool, creds config.DockerCredentials) (*ImageManifest, error) {
		return &ImageManifest{digest: fmt.Sprintf("sha256:%s", val)}, nil
	}
}

func (s *ImageResolverSuite) TestResolverCases() {

	type testcase struct {
		name        string
		image       string
		initial_tag string
		error       bool
		digest      bool
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
			digest:      false,
			expected:    "ubuntu:latest",
			resolver:    errorResolver,
		},
		{
			name:        "nothing",
			image:       "ubuntu",
			initial_tag: "",
			error:       false,
			digest:      true,
			expected:    "ubuntu@sha256:hash",
			resolver:    valueResolver("hash"),
		},
		{
			name:        "already digested",
			image:       "ubuntu@sha256:a9a425d086dbb34c1b5b99765596e2a3cc79b33826866c51cd4508d8eb327d2b",
			initial_tag: "sha256:a9a425d086dbb34c1b5b99765596e2a3cc79b33826866c51cd4508d8eb327d2b",
			error:       false,
			digest:      true,
			expected:    "ubuntu@sha256:a9a425d086dbb34c1b5b99765596e2a3cc79b33826866c51cd4508d8eb327d2b",
			resolver:    errorResolver,
		},
		{
			name:        "repo but no digest",
			image:       "ghcr.io/org/user/ubuntu:latest",
			initial_tag: "latest",
			error:       false,
			digest:      true,
			expected:    "ghcr.io/org/user/ubuntu@sha256:hash",
			resolver:    valueResolver("hash"),
		},
		{
			name:        "name no tag",
			image:       "ubuntu",
			initial_tag: "",
			error:       false,
			digest:      true,
			expected:    "ubuntu@sha256:hash",
			resolver:    valueResolver("hash"),
		},
		// {
		// 	name:        "remote resolver",
		// 	image:       "ubuntu:kinetic",
		// 	initial_tag: "kinetic",
		// 	error:       false,
		// 	digest:      true,
		// 	expected:    "ubuntu@sha256:490b20d4c90f834abcf386620a8906d21821aaa4db91c4665016883f043a10e4",
		// 	resolver:    fullResolver(),
		// },
	}

	mockCache := cache.NewMockCache[string]()

	for _, tc := range testcases {
		s.Run(tc.name, func() {
			i, err := NewImageID(tc.image)
			require.NoError(s.T(), err)
			require.NotNil(s.T(), i)
			require.Equal(s.T(), tc.initial_tag, i.tag.String())

			resolved := NewImageResolver(i)
			err = resolved.Resolve(ctx, tc.resolver, mockCache)
			if tc.error {
				require.Error(s.T(), err)
			} else {
				require.NoError(s.T(), err)
			}

			if tc.digest {
				require.NoError(s.T(), err)
				require.Equal(s.T(), tc.expected, resolved.Digest())

				if !i.HasDigest() {
					// If the image didn't already have a digest, check what we
					// created ended up in a cache.
					cachedDigest, found := mockCache.Get(i.String())
					require.True(s.T(), found)
					require.Equal(s.T(), tc.expected, cachedDigest)
				}

			} else {
				require.Empty(s.T(), resolved.Digest())

				_, found := mockCache.Get(i.String())
				require.False(s.T(), found)
			}

			// Cleanup the cache for the next run
			mockCache.Delete(i.String())
		})
	}

	mockCache.Close()
}
