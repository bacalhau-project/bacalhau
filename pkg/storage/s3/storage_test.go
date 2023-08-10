//go:build integration || !unit

package s3

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	s3helper "github.com/bacalhau-project/bacalhau/pkg/s3"

	"github.com/stretchr/testify/suite"
)

const bucket = "bacalhau-test-datasets"
const root = "integration-tests-do-not-delete/"
const prefix1 = root + "set1/"
const prefix2 = root + "set2/"
const region = "eu-west-1"

type StorageTestSuite struct {
	suite.Suite
	storage *StorageProvider
}

func (s *StorageTestSuite) SetupSuite() {
	cfg, err := s3helper.DefaultAWSConfig()
	s.Require().NoError(err)
	if !s3helper.HasValidCredentials(cfg) {
		s.T().Skip("No valid AWS credentials found")
	}

	clientProvider := s3helper.NewClientProvider(s3helper.ClientProviderParams{
		AWSConfig: cfg,
	})
	s.storage = NewStorage(StorageProviderParams{
		LocalDir:       s.T().TempDir(),
		ClientProvider: clientProvider,
	})
}

func TestStorageTestSuite(t *testing.T) {
	suite.Run(t, new(StorageTestSuite))
}

func (s *StorageTestSuite) TestHasStorageLocally() {
	ctx := context.Background()
	res, err := s.storage.HasStorageLocally(ctx, models.Artifact{})
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
		checksum        string
		versionID       string
		shouldFail      bool
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
			key:  root + "set*",
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
		{
			name:     "correct checksum",
			key:      prefix1 + "001.txt",
			checksum: "aLgNZhlhklWk0ATVRHbeUfkVes0KnZfNKUoKOGLK090=",
			expectedOutputs: []expectedOutput{
				{"001", "001.txt"},
			},
		},
		{
			name:       "bad checksum",
			key:        prefix1 + "001.txt",
			checksum:   "aLgNZhlhklWk0ATVRHbeUfkVes0KnZfNKUoKOGLK999=",
			shouldFail: true,
		},
		{
			name:       "no checksum",
			key:        prefix1 + "002.txt",
			checksum:   "aLgNZhlhklWk0ATVRHbeUfkVes0KnZfNKUoKOGLK999=",
			shouldFail: true,
		},
		{
			name: "versioned object - current version",
			key:  root + "version_file.txt",
			expectedOutputs: []expectedOutput{
				{"002", "version_file.txt"},
			},
		},
		{
			name:      "versioned object - current version explicit",
			key:       root + "version_file.txt",
			versionID: "Xwdg4C5YWv1_Hf5kVUIZbE1grU9XkuFA",
			expectedOutputs: []expectedOutput{
				{"002", "version_file.txt"},
			},
		},
		{
			name:      "versioned object - older version explicit",
			key:       root + "version_file.txt",
			versionID: "6QFI1rFeNw.GXFc09yPy2G..wMKaLz9C",
			expectedOutputs: []expectedOutput{
				{"001", "version_file.txt"},
			},
		},
		{
			name:       "versioned object - wrong version",
			key:        root + "version_file.txt",
			versionID:  "lxVWhWi1Z94vwDBOKYp.E9UlvTELWUEO",
			shouldFail: true,
		},
		{
			name:      "versioned object and checksum",
			key:       root + "version_file.txt",
			versionID: "6QFI1rFeNw.GXFc09yPy2G..wMKaLz9C",
			checksum:  "aLgNZhlhklWk0ATVRHbeUfkVes0KnZfNKUoKOGLK090=",
			expectedOutputs: []expectedOutput{
				{"001", "version_file.txt"},
			},
		},
	} {
		s.Run(tc.name, func() {
			ctx := context.Background()
			storageSpec := models.Artifact{
				StorageSource: models.StorageSourceS3,
				S3: &models.S3StorageSpec{
					Bucket:         bucket,
					Key:            tc.key,
					Region:         region,
					ChecksumSHA256: tc.checksum,
					VersionID:      tc.versionID,
				},
			}
			size, err := s.storage.GetVolumeSize(ctx, storageSpec)
			if tc.shouldFail {
				s.Error(err)
				return
			}
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
	storageSpec := models.Artifact{
		StorageSource: models.StorageSourceS3,
		S3: &models.S3StorageSpec{
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
