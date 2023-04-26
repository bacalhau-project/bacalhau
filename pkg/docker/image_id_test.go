//go:build unit || !integration

package docker

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ImageIDSuite struct {
	suite.Suite
}

func TestImageIDSuite(t *testing.T) {
	suite.Run(t, new(ImageIDSuite))
}

func (s *ImageIDSuite) TestImageIDStringer() {
	type testCase struct {
		name     string
		imageID  string
		expected string
		error    bool
		digest   bool
	}

	testCases := []testCase{
		{name: "simple latest", imageID: "ubuntu:latest", expected: "ubuntu:latest", error: false, digest: false},
		{name: "simple specific", imageID: "ubuntu:kinetic", expected: "ubuntu:kinetic", error: false, digest: false},
		{name: "simple none", imageID: "ubuntu", expected: "ubuntu", error: false, digest: false},
		{
			name:     "simple digest",
			imageID:  "ubuntu@sha256:6f4ca5ddeb85491f815d6ec8179c72e88ba207fadfaedb130d5c839a6f9e83c7",
			expected: "ubuntu@sha256:6f4ca5ddeb85491f815d6ec8179c72e88ba207fadfaedb130d5c839a6f9e83c7",
			error:    false,
			digest:   true,
		},
		{name: "simple repo", imageID: "ghcr.io/ubuntu:kinetic", expected: "ghcr.io/ubuntu:kinetic", error: false, digest: false},
		{
			name:     "less simple repo",
			imageID:  "ghcr.io/organisation/user/ubuntu:kinetic",
			expected: "ghcr.io/organisation/user/ubuntu:kinetic",
			error:    false,
			digest:   false},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			id, err := NewImageID(tc.imageID)
			if tc.error {
				require.Error(s.T(), err)
			} else {
				require.NoError(s.T(), err)
			}
			require.Equal(s.T(), tc.digest, id.HasDigest())
			require.Equal(s.T(), tc.expected, id.String())
		})
	}

}
