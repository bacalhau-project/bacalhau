//go:build integration || !unit

package s3

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/stretchr/testify/suite"
)

const bucket = "bacalhau-test-datasets"
const prefix1 = "integration-tests-do-not-delete/set1/"
const prefix2 = "integration-tests-do-not-delete/set2/"
const region = "eu-west-1"

type StorageTestSuite struct {
	suite.Suite
	storage *StorageProvider
}

func (s *StorageTestSuite) SetupSuite() {
	os.Setenv("AWS_PROFILE", "osoul")
	cfg, err := DefaultAWSConfig()
	s.Require().NoError(err)
	if !HasValidCredentials(cfg) {
		s.T().Skip("No valid AWS credentials found")
	}

	s.storage = NewStorage(StorageProviderParams{
		LocalDir:  s.T().TempDir(),
		AWSConfig: cfg,
	})
}

func TestStorageTestSuite(t *testing.T) {
	suite.Run(t, new(StorageTestSuite))
}

func (s *StorageTestSuite) TestHasStorageLocally() {
	ctx := context.Background()
	res, err := s.storage.HasStorageLocally(ctx, model.StorageSpec{})
	s.Require().NoError(err)
	s.False(res)
}

func (s *StorageTestSuite) TestIsInstalled() {
	ctx := context.Background()
	res, err := s.storage.IsInstalled(ctx)
	s.Require().NoError(err)
	s.True(res)
}

func (s *StorageTestSuite) TestStorage() {
	type expectedOutput struct {
		content string
		path    string
	}

	for _, tc := range []struct {
		name            string
		key             string
		expectedOutputs []expectedOutput
	}{
		{
			name: "single object",
			key:  prefix1 + "001.txt",
			expectedOutputs: []expectedOutput{
				{"001", "001.txt"},
			},
		},
		{
			name: "single directory",
			key:  prefix1,
			expectedOutputs: []expectedOutput{
				{"001", "001.txt"},
				{"002", "002.txt"},
				{"101", "101.txt"},
				{"102", "102.txt"},
			},
		},
		{
			name: "single directory trailing asterisk",
			key:  prefix1 + "*",
			expectedOutputs: []expectedOutput{
				{"001", "001.txt"},
				{"002", "002.txt"},
				{"101", "101.txt"},
				{"102", "102.txt"},
			},
		},
		{
			name: "nested directory",
			key:  prefix2,
			expectedOutputs: []expectedOutput{
				{"201", "201.txt"},
				{"202", "202.txt"},
				{"301", "nested/301.txt"},
				{"302", "nested/302.txt"},
			},
		},
		{
			name: "file pattern",
			key:  prefix1 + "00*",
			expectedOutputs: []expectedOutput{
				{"001", "001.txt"},
				{"002", "002.txt"},
			},
		},
		{
			name: "directory pattern",
			key:  "integration-tests-do-not-delete/set*",
			expectedOutputs: []expectedOutput{
				{"001", "set1/001.txt"},
				{"002", "set1/002.txt"},
				{"101", "set1/101.txt"},
				{"102", "set1/102.txt"},
				{"201", "set2/201.txt"},
				{"202", "set2/202.txt"},
				{"301", "set2/nested/301.txt"},
				{"302", "set2/nested/302.txt"},
			},
		},
	} {
		s.Run(tc.name, func() {
			ctx := context.Background()
			storageSpec := model.StorageSpec{
				StorageSource: model.StorageSourceS3,
				S3: &model.S3StorageSpec{
					Bucket: bucket,
					Key:    tc.key,
					Region: region,
				},
			}
			size, err := s.storage.GetVolumeSize(ctx, storageSpec)
			s.Require().NoError(err)
			s.Equal(uint64(len(tc.expectedOutputs)*4), size) // each file is 4 bytes long

			volume, err := s.storage.PrepareStorage(ctx, storageSpec)
			s.Require().NoError(err)

			// check that the files are there
			_, err = os.Stat(volume.Source)
			s.Require().NoError(err)

			// check that the files have the expected content
			for _, expectedFile := range tc.expectedOutputs {
				s.equalContent(expectedFile.content, volume.Source, expectedFile.path)
			}

			// check that the files are not there anymore
			err = s.storage.CleanupStorage(ctx, storageSpec, volume)
			s.Require().NoError(err)

			_, err = os.Stat(volume.Source)
			s.Require().ErrorAs(err, &os.ErrNotExist)
		})
	}
}

func (s *StorageTestSuite) TestNotFound() {
	ctx := context.Background()
	storageSpec := model.StorageSpec{
		StorageSource: model.StorageSourceS3,
		S3: &model.S3StorageSpec{
			Bucket: bucket,
			Key:    prefix1 + "00",
			Region: region,
		},
	}

	_, err := s.storage.GetVolumeSize(ctx, storageSpec)
	s.Require().Error(err)

	_, err = s.storage.PrepareStorage(ctx, storageSpec)
	s.Require().Error(err)
}

func (s *StorageTestSuite) equalContent(expected string, filePaths ...string) {
	bytes, err := os.ReadFile(filepath.Join(filePaths...))
	s.Require().NoError(err)
	s.Equal(expected+"\n", string(bytes))
}
