//go:build integration || !unit

/* spell-checker: disable */

package s3_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	s3helper "github.com/bacalhau-project/bacalhau/pkg/s3"
	s3test "github.com/bacalhau-project/bacalhau/pkg/s3/test"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"

	"github.com/stretchr/testify/suite"
)

const root = "integration-tests-do-not-delete/"
const prefix1 = root + "set1/"
const prefix2 = root + "set2/"

type StorageTestSuite struct {
	*s3test.HelperSuite
}

func TestStorageTestSuite(t *testing.T) {
	helperSuite := s3test.NewTestHelper(t, s3test.HelperSuiteParams{})
	suite.Run(t, &StorageTestSuite{HelperSuite: helperSuite})
}

func (s *StorageTestSuite) TestHasStorageLocally() {
	ctx := context.Background()
	res, err := s.Storage.HasStorageLocally(ctx, models.InputSource{})
	s.Require().NoError(err)
	s.False(res)
}

func (s *StorageTestSuite) TestIsInstalled() {
	ctx := context.Background()
	res, err := s.Storage.IsInstalled(ctx)
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
		pattern         string
		expectedOutputs []expectedOutput
		checksum        string
		versionID       string
		shouldFail      bool
		partitionConfig s3helper.PartitionConfig
		jobCount        int
		partitionIndex  int
	}{
		{
			name: "single object",
			key:  prefix1 + "001.txt",
			expectedOutputs: []expectedOutput{
				{"001", "001.txt"},
			},
		},
		{
			name:            "single object - part 0",
			key:             prefix1 + "001.txt",
			expectedOutputs: []expectedOutput{},
			partitionConfig: s3helper.PartitionConfig{
				Type: s3helper.PartitionKeyTypeObject,
			},
			jobCount:       2,
			partitionIndex: 0,
		},
		{
			name: "single object - part 1",
			key:  prefix1 + "001.txt",
			expectedOutputs: []expectedOutput{
				{"001", "001.txt"},
			},
			partitionConfig: s3helper.PartitionConfig{
				Type: s3helper.PartitionKeyTypeObject,
			},
			jobCount:       2,
			partitionIndex: 1,
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
			name: "single directory with object partitioning - part 0",
			key:  prefix1,
			expectedOutputs: []expectedOutput{
				{"002", "002.txt"},
				{"101", "101.txt"},
			},
			partitionConfig: s3helper.PartitionConfig{
				Type: s3helper.PartitionKeyTypeObject,
			},
			jobCount:       2,
			partitionIndex: 0,
		},
		{
			name: "single directory with object partitioning - part 1",
			key:  prefix1,
			expectedOutputs: []expectedOutput{
				{"001", "001.txt"},
				{"102", "102.txt"},
			},
			partitionConfig: s3helper.PartitionConfig{
				Type: s3helper.PartitionKeyTypeObject,
			},
			jobCount:       2,
			partitionIndex: 1,
		},
		{
			name: "single directory with regex partitioning - part 0",
			key:  prefix1,
			expectedOutputs: []expectedOutput{
				{"101", "101.txt"},
				{"102", "102.txt"},
			},
			partitionConfig: s3helper.PartitionConfig{
				Type:    s3helper.PartitionKeyTypeRegex,
				Pattern: `\d0`,
			},
			jobCount:       2,
			partitionIndex: 0,
		},
		{
			name: "single directory with regex partitioning - part 1",
			key:  prefix1,
			expectedOutputs: []expectedOutput{
				{"001", "001.txt"},
				{"002", "002.txt"},
			},
			partitionConfig: s3helper.PartitionConfig{
				Type:    s3helper.PartitionKeyTypeRegex,
				Pattern: `\d0`,
			},
			jobCount:       2,
			partitionIndex: 1,
		},
		{
			name: "single directory with substring partitioning - part 0",
			key:  prefix1,
			expectedOutputs: []expectedOutput{
				{"001", "001.txt"},
				{"101", "101.txt"},
			},
			partitionConfig: s3helper.PartitionConfig{
				Type:       s3helper.PartitionKeyTypeSubstring,
				StartIndex: 2,
				EndIndex:   3,
			},
			jobCount:       2,
			partitionIndex: 0,
		},
		{
			name: "single directory with substring partitioning - part 1",
			key:  prefix1,
			expectedOutputs: []expectedOutput{
				{"002", "002.txt"},
				{"102", "102.txt"},
			},
			partitionConfig: s3helper.PartitionConfig{
				Type:       s3helper.PartitionKeyTypeSubstring,
				StartIndex: 2,
				EndIndex:   3,
			},
			jobCount:       2,
			partitionIndex: 1,
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
			name: "single directory trailing asterisk partitioning",
			key:  prefix1 + "*",
			expectedOutputs: []expectedOutput{
				{"002", "002.txt"},
				{"101", "101.txt"},
			},
			partitionConfig: s3helper.PartitionConfig{
				Type: s3helper.PartitionKeyTypeObject,
			},
			jobCount:       2,
			partitionIndex: 0,
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
			name: "nested directory with regex partitioning",
			key:  prefix2,
			expectedOutputs: []expectedOutput{
				{"301", "nested/301.txt"},
				{"302", "nested/302.txt"},
			},
			partitionConfig: s3helper.PartitionConfig{
				Type:    s3helper.PartitionKeyTypeRegex,
				Pattern: `nested/\d0`,
			},
			jobCount:       3,
			partitionIndex: 2,
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
			name: "file pattern - part 0",
			key:  prefix1 + "00*",
			expectedOutputs: []expectedOutput{
				{"001", "001.txt"},
			},
			partitionConfig: s3helper.PartitionConfig{
				Type:       s3helper.PartitionKeyTypeSubstring,
				StartIndex: 0,
				EndIndex:   1,
			},
			jobCount:       2,
			partitionIndex: 0,
		},
		{
			name: "file pattern - part 1",
			key:  prefix1 + "00*",
			expectedOutputs: []expectedOutput{
				{"002", "002.txt"},
			},
			partitionConfig: s3helper.PartitionConfig{
				Type:       s3helper.PartitionKeyTypeSubstring,
				StartIndex: 0,
				EndIndex:   1,
			},
			jobCount:       2,
			partitionIndex: 1,
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
			name: "directory pattern - part 0",
			key:  root + "set*",
			expectedOutputs: []expectedOutput{
				{"001", "set1/001.txt"},
				{"101", "set1/101.txt"},
				{"201", "set2/201.txt"},
				{"301", "set2/nested/301.txt"},
			},
			partitionConfig: s3helper.PartitionConfig{
				Type:    s3helper.PartitionKeyTypeRegex,
				Pattern: `\d\.txt`,
			},
			jobCount:       2,
			partitionIndex: 0,
		},
		{
			name: "directory pattern - part 1",
			key:  root + "set*",
			expectedOutputs: []expectedOutput{
				{"002", "set1/002.txt"},
				{"102", "set1/102.txt"},
				{"202", "set2/202.txt"},
				{"302", "set2/nested/302.txt"},
			},
			partitionConfig: s3helper.PartitionConfig{
				Type:    s3helper.PartitionKeyTypeRegex,
				Pattern: `\d\.txt`,
			},
			jobCount:       2,
			partitionIndex: 1,
		},
		{
			name:    "single directory filter",
			key:     prefix1 + "*",
			pattern: "[0-1]01.txt",
			expectedOutputs: []expectedOutput{
				{"001", "001.txt"},
				{"101", "101.txt"},
			},
		},
		{
			name:    "nested directory filter",
			key:     prefix2,
			pattern: "nested/.*",
			expectedOutputs: []expectedOutput{
				{"301", "nested/301.txt"},
				{"302", "nested/302.txt"},
			},
		},
		{
			name:            "filter filters all",
			key:             prefix1 + "*",
			pattern:         "nonexistent",
			expectedOutputs: []expectedOutput{},
		},
		{
			name:    "filter with no key",
			pattern: fmt.Sprintf("^%s.*", prefix1),
			expectedOutputs: []expectedOutput{
				{"001", prefix1 + "001.txt"},
				{"002", prefix1 + "002.txt"},
				{"101", prefix1 + "101.txt"},
				{"102", prefix1 + "102.txt"},
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

			// Set default job count and partition index if not specified
			if tc.jobCount == 0 {
				tc.jobCount = 1
			}

			storageSpec := models.InputSource{
				Source: &models.SpecConfig{
					Type: models.StorageSourceS3,
					Params: s3helper.SourceSpec{
						Bucket:         s.Bucket,
						Key:            tc.key,
						Filter:         tc.pattern,
						Region:         s.Region,
						ChecksumSHA256: tc.checksum,
						VersionID:      tc.versionID,
						Partition:      tc.partitionConfig,
					}.ToMap(),
				},
			}

			execution := mock.Execution()
			execution.Job.Count = tc.jobCount
			execution.PartitionIndex = tc.partitionIndex

			size, err := s.Storage.GetVolumeSize(ctx, execution, storageSpec)
			if tc.shouldFail {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			s.Equal(uint64(len(tc.expectedOutputs)*4), size) // each file is 4 bytes long

			volume, err := s.Storage.PrepareStorage(ctx, s.T().TempDir(), execution, storageSpec)
			s.Require().NoError(err)

			// check that the files are there
			_, err = os.Stat(volume.Source)
			s.Require().NoError(err)

			// check that the files have the expected content
			for _, expectedFile := range tc.expectedOutputs {
				s.equalContent(expectedFile.content, volume.Source, expectedFile.path)
			}

			// check that the files are not there anymore
			err = s.Storage.CleanupStorage(ctx, storageSpec, volume)
			s.Require().NoError(err)

			_, err = os.Stat(volume.Source)
			s.Require().ErrorAs(err, &os.ErrNotExist)
		})
	}
}

func (s *StorageTestSuite) TestNotFound() {
	ctx := context.Background()
	storageSpec := models.InputSource{
		Source: &models.SpecConfig{
			Type: models.StorageSourceS3,
			Params: s3helper.SourceSpec{
				Bucket: s.Bucket,
				Key:    prefix1 + "00",
				Region: s.Region,
			}.ToMap(),
		},
	}

	_, err := s.Storage.GetVolumeSize(ctx, mock.Execution(), storageSpec)
	s.Require().Error(err)

	_, err = s.Storage.PrepareStorage(ctx, s.T().TempDir(), mock.Execution(), storageSpec)
	s.Require().Error(err)
}

func (s *StorageTestSuite) equalContent(expected string, filePaths ...string) {
	bytes, err := os.ReadFile(filepath.Join(filePaths...))
	s.Require().NoError(err)
	s.Equal(expected+"\n", string(bytes))
}
