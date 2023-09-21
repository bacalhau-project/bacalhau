//go:build unit || !integration

package legacy_test

import (
	"testing"

	_ "github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/migration/legacy"
	"github.com/stretchr/testify/suite"
)

type LegacyFromSuite struct {
	suite.Suite
}

func TestLegacyFromSuite(t *testing.T) {
	suite.Run(t, new(LegacyFromSuite))
}

func (s *LegacyFromSuite) TestLegacyStorageSpec() {
	testcases := []struct {
		name        string
		arg         model.StorageSpec
		expected    *models.SpecConfig
		expectError bool
	}{
		{
			name: "ipfs_ok",
			arg: model.StorageSpec{
				StorageSource: model.StorageSourceIPFS,
				CID:           "abcdefghi",
			},
			expected: &models.SpecConfig{
				Type: models.StorageSourceIPFS,
				Params: map[string]interface{}{
					"CID": "abcdefghi",
				},
			},
			expectError: false,
		},
		{
			name:        "ipfs_err",
			arg:         model.StorageSpec{StorageSource: model.StorageSourceIPFS},
			expected:    nil,
			expectError: true,
		},
		{
			name: "s3_ok",
			arg: model.StorageSpec{
				StorageSource: model.StorageSourceS3,
				S3: &model.S3StorageSpec{
					Bucket: "bucket",
				},
			},
			expected: &models.SpecConfig{
				Type: models.StorageSourceS3,
				Params: map[string]interface{}{
					"Bucket":         "bucket",
					"ChecksumSHA256": "",
					"Endpoint":       "",
					"Key":            "",
					"Region":         "",
					"VersionID":      "",
				},
			},
			expectError: false,
		},
		{
			name: "s3_err",
			arg: model.StorageSpec{
				StorageSource: model.StorageSourceS3,
			},
			expected:    nil,
			expectError: true,
		},
		{
			name: "url_ok",
			arg: model.StorageSpec{
				StorageSource: model.StorageSourceURLDownload,
				URL:           "http://localhost",
			},
			expected: &models.SpecConfig{
				Type: models.StorageSourceURL,
				Params: map[string]interface{}{
					"URL": "http://localhost",
				},
			},
			expectError: false,
		},
		{
			name:        "url_err",
			arg:         model.StorageSpec{StorageSource: model.StorageSourceURLDownload},
			expected:    nil,
			expectError: true,
		},
		{
			name: "url_err_invalid",
			arg:  model.StorageSpec{StorageSource: model.StorageSourceURLDownload},
			expected: &models.SpecConfig{
				Type: models.StorageSourceURL,
				Params: map[string]interface{}{
					"CID": "http://localhost",
				},
			},
			expectError: true,
		},
		{
			name: "inline_ok",
			arg: model.StorageSpec{
				StorageSource: model.StorageSourceInline,
				URL:           "data://",
			},
			expected: &models.SpecConfig{
				Type: models.StorageSourceInline,
				Params: map[string]interface{}{
					"URL": "data://",
				},
			},
			expectError: false,
		},
		{
			name:        "inline_err",
			arg:         model.StorageSpec{StorageSource: model.StorageSourceURLDownload},
			expected:    nil,
			expectError: true,
		},
		{
			name: "local_ok",
			arg: model.StorageSpec{
				StorageSource: model.StorageSourceLocalDirectory,
				SourcePath:    "/tmp",
				ReadWrite:     false,
			},
			expected: &models.SpecConfig{
				Type: models.StorageSourceLocalDirectory,
				Params: map[string]interface{}{
					"SourcePath": "/tmp",
					"ReadWrite":  false,
				},
			},
			expectError: false,
		},
		{
			name: "local_err",
			arg: model.StorageSpec{
				StorageSource: model.StorageSourceLocalDirectory,
				Path:          "/tmp",
			},
			expected:    nil,
			expectError: true,
		},
		{
			name: "invalid",
			arg: model.StorageSpec{
				StorageSource: model.StorageSourceType(10000),
			},
			expected:    nil,
			expectError: true,
		},
	}

	for _, tc := range testcases {
		s.T().Run(tc.name, func(t *testing.T) {
			config, err := legacy.FromLegacyStorageSpec(tc.arg)
			if tc.expectError {
				s.Require().Error(err)
				return
			}

			s.Require().Equal(tc.expected, config)
		})
	}
}
