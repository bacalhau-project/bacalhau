package s3

/* spell-checker: disable */

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

type PartitionTestSuite struct {
	suite.Suite
}

func TestPartitionSuite(t *testing.T) {
	suite.Run(t, new(PartitionTestSuite))
}

func (s *PartitionTestSuite) TestConfigValidation() {
	tests := []struct {
		name        string
		config      PartitionConfig
		expectedErr string
	}{
		// Object partitioning
		{
			name: "valid object config - minimal",
			config: PartitionConfig{
				Type: PartitionKeyTypeObject,
			},
		},
		{
			name: "valid object config with unused fields",
			config: PartitionConfig{
				Type:       PartitionKeyTypeObject,
				Pattern:    "unused",
				StartIndex: 1,
				EndIndex:   2,
			},
		},

		// Regex partitioning
		{
			name: "valid regex - simple pattern",
			config: PartitionConfig{
				Type:    PartitionKeyTypeRegex,
				Pattern: `\d+`,
			},
		},
		{
			name: "valid regex - complex pattern",
			config: PartitionConfig{
				Type:    PartitionKeyTypeRegex,
				Pattern: `^(?:user|group)-(\d+)-\w+$`,
			},
		},
		{
			name: "empty regex pattern",
			config: PartitionConfig{
				Type:    PartitionKeyTypeRegex,
				Pattern: "",
			},
			expectedErr: "regex pattern cannot be empty",
		},
		{
			name: "invalid regex pattern - syntax error",
			config: PartitionConfig{
				Type:    PartitionKeyTypeRegex,
				Pattern: "[unclosed",
			},
			expectedErr: "invalid regex pattern",
		},

		// Substring partitioning
		{
			name: "valid substring - zero start",
			config: PartitionConfig{
				Type:       PartitionKeyTypeSubstring,
				StartIndex: 0,
				EndIndex:   5,
			},
		},
		{
			name: "valid substring - non-zero start",
			config: PartitionConfig{
				Type:       PartitionKeyTypeSubstring,
				StartIndex: 3,
				EndIndex:   10,
			},
		},
		{
			name: "negative start index",
			config: PartitionConfig{
				Type:       PartitionKeyTypeSubstring,
				StartIndex: -1,
				EndIndex:   5,
			},
			expectedErr: "start index cannot be negative",
		},
		{
			name: "end index equals start index",
			config: PartitionConfig{
				Type:       PartitionKeyTypeSubstring,
				StartIndex: 5,
				EndIndex:   5,
			},
			expectedErr: "end index must be greater than start index",
		},
		{
			name: "end index less than start index",
			config: PartitionConfig{
				Type:       PartitionKeyTypeSubstring,
				StartIndex: 10,
				EndIndex:   5,
			},
			expectedErr: "end index must be greater than start index",
		},

		// Date partitioning
		{
			name: "valid date ",
			config: PartitionConfig{
				Type:       PartitionKeyTypeDate,
				DateFormat: "2006-01-02",
			},
		},

		{
			name: "valid date - with timezone",
			config: PartitionConfig{
				Type:       PartitionKeyTypeDate,
				DateFormat: "2006-01-02T15:04:05Z07:00",
			},
		},
		{
			name: "empty date format",
			config: PartitionConfig{
				Type:       PartitionKeyTypeDate,
				DateFormat: "",
			},
			expectedErr: "date format cannot be empty",
		},

		{
			name: "empty partition type",
			config: PartitionConfig{
				Type: "",
			},
		},
		// Invalid types
		{
			name: "invalid partition type",
			config: PartitionConfig{
				Type: "invalid",
			},
			expectedErr: "unsupported partition key type",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			err := tt.config.Validate()
			if tt.expectedErr != "" {
				s.Require().Error(err)
				s.Contains(err.Error(), tt.expectedErr)
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

func (s *PartitionTestSuite) TestPartitionObjects_InvalidInputs() {
	objects := []ObjectSummary{createObjectSummary("test.txt", false)}
	source := SourceSpec{
		Key: "prefix/",
		Partition: PartitionConfig{
			Type: PartitionKeyTypeObject,
		},
	}

	tests := []struct {
		name            string
		objects         []ObjectSummary
		totalPartitions int
		partitionIndex  int
		expectedErr     string
	}{
		// Invalid partition counts
		{
			name:            "negative total partitions",
			objects:         objects,
			totalPartitions: -1,
			partitionIndex:  0,
			expectedErr:     "job partitions/count must be greater than 0",
		},
		{
			name:            "zero total partitions",
			objects:         objects,
			totalPartitions: 0,
			partitionIndex:  0,
			expectedErr:     "job partitions/count must be greater than 0",
		},

		// Invalid partition indices
		{
			name:            "negative partition index",
			objects:         objects,
			totalPartitions: 2,
			partitionIndex:  -1,
			expectedErr:     "partition index must be between 0 and",
		},
		{
			name:            "partition index equals total partitions",
			objects:         objects,
			totalPartitions: 2,
			partitionIndex:  2,
			expectedErr:     "partition index must be between 0 and",
		},
		{
			name:            "partition index exceeds total partitions",
			objects:         objects,
			totalPartitions: 2,
			partitionIndex:  3,
			expectedErr:     "partition index must be between 0 and",
		},

		// Object list variations
		{
			name:            "nil object list",
			objects:         nil,
			totalPartitions: 2,
			partitionIndex:  0,
			expectedErr:     "", // should handle nil list gracefully
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			_, err := PartitionObjects(tt.objects, tt.totalPartitions, tt.partitionIndex, source)
			if tt.expectedErr != "" {
				s.Require().Error(err)
				s.Contains(err.Error(), tt.expectedErr)
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

func (s *PartitionTestSuite) TestPartitionByObject() {
	tests := []struct {
		name            string
		paths           []string
		prefix          string
		totalPartitions int
		expected        [][]string
	}{
		{
			name: "basic file distribution",
			paths: []string{
				"file1.txt",
				"file2.txt",
				"dir/", // directory
				"file3.txt",
				"file4.txt",
			},
			prefix:          "",
			totalPartitions: 2,
			expected: [][]string{
				{"file1.txt", "file3.txt"}, // partition 0
				{"file2.txt", "file4.txt"}, // partition 1
			},
		},
		{
			name: "nested paths",
			paths: []string{
				"dir1/file1.txt",
				"dir1/dir2/file2.txt",
				"dir1/dir2/dir3/", // directory
				"dir1/dir2/dir3/file3.txt",
				"other/file4.txt",
			},
			prefix:          "",
			totalPartitions: 2,
			expected: [][]string{
				{"dir1/dir2/file2.txt", "dir1/dir2/dir3/file3.txt", "other/file4.txt"},
				{"dir1/file1.txt"},
			},
		},
		{
			name: "with prefix trimming",
			paths: []string{
				"prefix/subdir/file1.txt",
				"prefix/subdir/file2.txt",
				"prefix/other/file3.txt",
				"prefix/dir/", // directory
			},
			prefix:          "prefix/",
			totalPartitions: 2,
			expected: [][]string{
				{"prefix/subdir/file2.txt", "prefix/other/file3.txt"},
				{"prefix/subdir/file1.txt"},
			},
		},
		{
			name: "mixed depth paths",
			paths: []string{
				"short.txt",
				"dir/medium.txt",
				"dir/subdir/long.txt",
				"very/long/path/file.txt",
				"dir/", // directory
			},
			prefix:          "",
			totalPartitions: 2,
			expected: [][]string{
				{"dir/medium.txt"},
				{"short.txt", "dir/subdir/long.txt", "very/long/path/file.txt"},
			},
		},
		{
			name: "single partition",
			paths: []string{
				"file1.txt",
				"file2.txt",
				"dir/", // directory
			},
			prefix:          "",
			totalPartitions: 1,
			expected: [][]string{
				{"file1.txt", "file2.txt"}, // all files in single partition
			},
		},
		{
			name:            "empty list",
			paths:           []string{},
			prefix:          "",
			totalPartitions: 2,
			expected: [][]string{
				{}, // partition 0
				{}, // partition 1
			},
		},
		{
			name: "only directories",
			paths: []string{
				"dir1/",
				"dir2/",
				"dir1/dir3/",
			},
			prefix:          "",
			totalPartitions: 2,
			expected: [][]string{
				{}, // partition 0
				{}, // partition 1
			},
		},
		{
			name: "with special characters",
			paths: []string{
				"prefix/!@#.txt",
				"prefix/spaces in name.txt",
				"prefix/tabs\there.txt",
				"prefix/special/", // directory
			},
			prefix:          "prefix/",
			totalPartitions: 2,
			expected: [][]string{
				{"prefix/!@#.txt"},
				{"prefix/spaces in name.txt", "prefix/tabs\there.txt"},
			},
		},
		{
			name: "unicode filenames",
			paths: []string{
				"prefix/文件1.txt",
				"prefix/文件2.txt",
				"prefix/文件dir/", // directory
			},
			prefix:          "prefix/",
			totalPartitions: 2,
			expected: [][]string{
				{"prefix/文件2.txt"},
				{"prefix/文件1.txt"},
			},
		},
		{
			name: "many partitions",
			paths: []string{
				"file1.txt",
				"file2.txt",
				"file3.txt",
			},
			prefix:          "",
			totalPartitions: 5,
			expected: [][]string{
				{},            // partition 0
				{"file1.txt"}, // partition 1
				{},            // partition 2
				{"file2.txt"}, // partition 3
				{"file3.txt"}, // partition 4
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			objects := createObjectsFromStrings(tt.paths)
			spec := SourceSpec{
				Key: tt.prefix,
				Partition: PartitionConfig{
					Type: PartitionKeyTypeObject,
				},
			}
			s.verifyPartitioning(spec, objects, tt.totalPartitions, tt.expected)
		})
	}
}

func (s *PartitionTestSuite) TestPartitionByRegex() {
	tests := []struct {
		name            string
		paths           []string
		prefix          string
		pattern         string
		totalPartitions int
		expected        [][]string
		expectError     bool
		errorContains   string
	}{
		{
			name: "basic pattern matching with capture groups",
			paths: []string{
				"prefix/user123/file1.txt",
				"prefix/user123/file2.txt",
				"prefix/user123/another.txt",
				"prefix/user456/file3.txt",
				"prefix/user789/file4.txt",
				"prefix/invalid/file5.txt", // no match
				"prefix/userdir/",          // directory
			},
			prefix:          "prefix/",
			pattern:         `user(\d+)`,
			totalPartitions: 4,
			expected: [][]string{
				{"prefix/user456/file3.txt", "prefix/invalid/file5.txt"}, // partition 0 + fallback
				{"prefix/user789/file4.txt"},
				{},
				{"prefix/user123/file1.txt", "prefix/user123/file2.txt", "prefix/user123/another.txt"},
			},
		},
		{
			name: "all non-matching paths",
			paths: []string{
				"prefix/abc.txt",
				"prefix/def.txt",
				"prefix/ghi.txt",
				"prefix/dir/", // directory
			},
			prefix:          "prefix/",
			pattern:         `(\d{4}-\d{2})`, // looking for dates
			totalPartitions: 2,
			expected: [][]string{
				{"prefix/abc.txt", "prefix/def.txt", "prefix/ghi.txt"}, // all in fallback partition
				{}, // empty partition
			},
		},
		{
			name: "unicode in pattern",
			paths: []string{
				"prefix/用户123-数据.txt",
				"prefix/用户123-目录.txt",
				"prefix/用户456-数据.txt",
				"prefix/无效.txt",
				"prefix/目录/", // directory
			},
			prefix:          "prefix/",
			pattern:         `用户(\d+)`,
			totalPartitions: 2,
			expected: [][]string{
				{"prefix/用户456-数据.txt", "prefix/无效.txt"},
				{"prefix/用户123-数据.txt", "prefix/用户123-目录.txt"},
			},
		},
		{
			name: "nested paths",
			paths: []string{
				"prefix/a/user123/file.txt",
				"prefix/a/user123/another.txt",
				"prefix/b/user456/file.txt",
				"prefix/c/invalid/file.txt",
				"prefix/d/user789/", // directory
			},
			prefix:          "prefix/",
			pattern:         `user(\d+)`,
			totalPartitions: 2,
			expected: [][]string{
				{"prefix/b/user456/file.txt", "prefix/c/invalid/file.txt"},    // partition 0
				{"prefix/a/user123/file.txt", "prefix/a/user123/another.txt"}, // partition 1
			},
		},
		{
			name: "multiple capture groups",
			paths: []string{
				"prefix/user123-group456.txt",
				"prefix/user789-group012.txt",
				"prefix/user123-group999.txt",
				"prefix/user123-group999.log",
				"prefix/invalid.txt",
			},
			prefix:          "prefix/",
			pattern:         `user(\d+)-group(\d+)`,
			totalPartitions: 3,
			expected: [][]string{
				{"prefix/invalid.txt"},
				{"prefix/user123-group999.txt", "prefix/user123-group999.log"},
				{"prefix/user123-group456.txt", "prefix/user789-group012.txt"},
			},
		},
		{
			name: "pattern without capture groups",
			paths: []string{
				"prefix/001.txt",
				"prefix/002.txt",
				"prefix/003.txt",
				"prefix/abc.txt", // no match
			},
			prefix:          "prefix/",
			pattern:         `00\d`,
			totalPartitions: 3,
			expected: [][]string{
				{"prefix/abc.txt", "prefix/001.txt", "prefix/002.txt"}, // fallback for no match
				{"prefix/003.txt"},
				{},
			},
		},
		{
			name: "mix of full match and no match",
			paths: []string{
				"prefix/log-2024.txt",
				"prefix/log-2025.txt",
				"prefix/other.txt",
			},
			prefix:          "prefix/",
			pattern:         `log-20\d{2}`,
			totalPartitions: 2,
			expected: [][]string{
				{"prefix/other.txt", "prefix/log-2024.txt"}, // fallback + hash of "log-2024"
				{"prefix/log-2025.txt"},                     // hash of "log-2025"
			},
		},
		{
			name: "simple numeric pattern",
			paths: []string{
				"prefix/1.txt",
				"prefix/2.txt",
				"prefix/3.txt",
				"prefix/a.txt",
			},
			prefix:          "prefix/",
			pattern:         `\d`,
			totalPartitions: 2,
			expected: [][]string{
				{"prefix/a.txt", "prefix/1.txt", "prefix/3.txt"}, // fallback + hash distribution
				{"prefix/2.txt"}, // hash distribution
			},
		},
		{
			name: "empty capture group",
			paths: []string{
				"prefix/data1.txt",
				"prefix/data2.txt",
			},
			prefix:          "prefix/",
			pattern:         "()", // empty capture group
			totalPartitions: 2,
			expected: [][]string{
				{"prefix/data1.txt", "prefix/data2.txt"},
				{},
			},
		},
		{
			name:            "empty input",
			paths:           []string{},
			prefix:          "prefix/",
			pattern:         `\d+`,
			totalPartitions: 2,
			expected: [][]string{
				{},
				{},
			},
		},
		{
			name: "all directories",
			paths: []string{
				"prefix/user123/",
				"prefix/user456/",
			},
			prefix:          "prefix/",
			pattern:         `user\d+`,
			totalPartitions: 2,
			expected: [][]string{
				{},
				{},
			},
		},
		{
			name: "invalid regex",
			paths: []string{
				"test.txt",
			},
			prefix:          "",
			pattern:         "[", // invalid regex pattern
			totalPartitions: 2,
			expectError:     true,
			errorContains:   "invalid regex pattern",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			objects := createObjectsFromStrings(tt.paths)
			spec := SourceSpec{
				Key: tt.prefix,
				Partition: PartitionConfig{
					Type:    PartitionKeyTypeRegex,
					Pattern: tt.pattern,
				},
			}
			if tt.expectError {
				_, err := PartitionObjects(objects, tt.totalPartitions, 0, spec)
				s.Require().Error(err)
				s.Contains(err.Error(), tt.errorContains)
				return
			}
			s.verifyPartitioning(spec, objects, tt.totalPartitions, tt.expected)
		})
	}
}

func (s *PartitionTestSuite) TestPartitionBySubstring() {
	tests := []struct {
		name            string
		paths           []string
		prefix          string
		startIndex      int
		endIndex        int
		totalPartitions int
		expected        [][]string
	}{
		{
			name: "basic substring extraction",
			paths: []string{
				"prefix/abc123.txt",
				"prefix/def456.txt",
				"prefix/def999.txt",
				"prefix/def999-more-ignored-chars.txt",
				"prefix/ghi789.txt",
				"prefix/dir/", // directory
			},
			prefix:          "prefix/",
			startIndex:      0,
			endIndex:        3,
			totalPartitions: 4,
			expected: [][]string{
				{"prefix/def456.txt", "prefix/def999.txt", "prefix/def999-more-ignored-chars.txt"},
				{"prefix/ghi789.txt"}, // no match
				{},                    // no match
				{"prefix/abc123.txt"},
			},
		},
		{
			name: "substring with short key fallback",
			paths: []string{
				"prefix/short.txt",
				"prefix/medium-name.txt",
				"prefix/very-long-name.txt",
				"prefix/dir/", // directory
			},
			prefix:          "prefix/",
			startIndex:      0,
			endIndex:        10, // longer than some keys
			totalPartitions: 3,
			expected: [][]string{
				{"prefix/short.txt", "prefix/medium-name.txt"}, // fallback partition
				{"prefix/very-long-name.txt"},
				{}, // no match
			},
		},
		{
			name: "mid-string extraction",
			paths: []string{
				"prefix/user-123-abc.txt",
				"prefix/user-456-def.txt",
				"prefix/user-456-another.txt",
				"prefix/user-789-ghi.txt",
				"prefix/dir/", // directory
			},
			prefix:          "prefix/user-",
			startIndex:      0,
			endIndex:        3,
			totalPartitions: 2,
			expected: [][]string{
				{"prefix/user-456-def.txt", "prefix/user-456-another.txt"},
				{"prefix/user-123-abc.txt", "prefix/user-789-ghi.txt"},
			},
		},
		{
			name: "unicode substring",
			paths: []string{
				"prefix/用户123.txt",
				"prefix/户用456.txt",
				"prefix/用户789.txt",
				"prefix/dir/", // directory
			},
			prefix:          "prefix/",
			startIndex:      0,
			endIndex:        1,
			totalPartitions: 3,
			expected: [][]string{
				{"prefix/户用456.txt"},
				{}, // no match
				{"prefix/用户123.txt", "prefix/用户789.txt"},
			},
		},
		{
			name: "with special characters",
			paths: []string{
				"prefix/abc!@#.txt",
				"prefix/def$%^.txt",
				"prefix/xyz$%^.txt",
				"prefix/ghi&*(.txt",
				"prefix/dir/", // directory
			},
			prefix:          "prefix/",
			startIndex:      3,
			endIndex:        6,
			totalPartitions: 2,
			expected: [][]string{
				{"prefix/def$%^.txt", "prefix/xyz$%^.txt"},
				{"prefix/abc!@#.txt", "prefix/ghi&*(.txt"},
			},
		},
		{
			name: "nested paths",
			paths: []string{
				"prefix/a/deep/file1.txt",
				"prefix/b/deep/file2.txt",
				"prefix/b/another/file2.txt",
				"prefix/c/deep/file3.txt",
				"prefix/d/dir/", // directory
			},
			prefix:          "prefix/",
			startIndex:      0,
			endIndex:        1, // first character after prefix
			totalPartitions: 2,
			expected: [][]string{
				{"prefix/a/deep/file1.txt", "prefix/c/deep/file3.txt"},
				{"prefix/b/deep/file2.txt", "prefix/b/another/file2.txt"},
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			objects := createObjectsFromStrings(tt.paths)
			spec := SourceSpec{
				Key: tt.prefix,
				Partition: PartitionConfig{
					Type:       PartitionKeyTypeSubstring,
					StartIndex: tt.startIndex,
					EndIndex:   tt.endIndex,
				},
			}
			s.verifyPartitioning(spec, objects, tt.totalPartitions, tt.expected)
		})
	}
}

func (s *PartitionTestSuite) TestPartitionByDate() {
	tests := []struct {
		name            string
		paths           []string
		prefix          string
		dateFormat      string
		totalPartitions int
		expected        [][]string
	}{
		{
			name: "daily grouping",
			paths: []string{
				"prefix/2023-01-01-data.txt",
				"prefix/2023-01-02-data.txt",
				"prefix/2023-01-02-another.txt",
				"prefix/2023-01-02-more.txt",
				"prefix/2023-01-03-data.txt",
				"prefix/invalid-date.txt",
				"prefix/dates/", // directory
			},
			prefix:          "prefix/",
			dateFormat:      "2006-01-02",
			totalPartitions: 2,
			expected: [][]string{
				{"prefix/2023-01-01-data.txt", "prefix/2023-01-03-data.txt", "prefix/invalid-date.txt"},
				{"prefix/2023-01-02-data.txt", "prefix/2023-01-02-another.txt", "prefix/2023-01-02-more.txt"},
			},
		},
		{
			name: "monthly grouping",
			paths: []string{
				"prefix/2023-01/data1.txt",
				"prefix/2023-02/data2.txt",
				"prefix/2023-02/another.txt",
				"prefix/2023-02/more.txt",
				"prefix/2023-03/data3.txt",
				"prefix/invalid/data.txt",
				"prefix/months/", // directory
			},
			prefix:          "prefix/",
			dateFormat:      "2006-01",
			totalPartitions: 2,
			expected: [][]string{
				{"prefix/2023-01/data1.txt", "prefix/2023-03/data3.txt", "prefix/invalid/data.txt"},
				{"prefix/2023-02/data2.txt", "prefix/2023-02/another.txt", "prefix/2023-02/more.txt"},
			},
		},
		{
			name: "yearly grouping",
			paths: []string{
				"prefix/2021/data.txt",
				"prefix/2022/data.txt",
				"prefix/2022/another.txt",
				"prefix/2022/more.txt",
				"prefix/2023/data.txt",
				"prefix/invalid/data.txt",
				"prefix/years/", // directory
			},
			prefix:          "prefix/",
			dateFormat:      "2006",
			totalPartitions: 2,
			expected: [][]string{
				{"prefix/2021/data.txt", "prefix/2023/data.txt", "prefix/invalid/data.txt"},
				{"prefix/2022/data.txt", "prefix/2022/another.txt", "prefix/2022/more.txt"},
			},
		},
		{
			name: "with timezone",
			paths: []string{
				"prefix/2023-01-01T10:00:00Z.txt",
				"prefix/2023-01-01T15:30:00-07:00.txt",
				"prefix/2023-01-02T01:00:00+09:00.txt",
				"prefix/2023-01-02T02:00:00+09:00.txt",
				"prefix/2023-01-02T05:00:00+09:00.txt",
				"prefix/invalid.txt",
				"prefix/tz/", // directory
			},
			prefix:          "prefix/",
			dateFormat:      "2006-01-02T1",
			totalPartitions: 2,
			expected: [][]string{
				{"prefix/2023-01-02T01:00:00+09:00.txt", "prefix/2023-01-02T02:00:00+09:00.txt", "prefix/2023-01-02T05:00:00+09:00.txt", "prefix/invalid.txt"},
				{"prefix/2023-01-01T15:30:00-07:00.txt", "prefix/2023-01-01T10:00:00Z.txt"},
			},
		},
		{
			name: "mixed date formats",
			paths: []string{
				"prefix/20230101.txt",
				"prefix/2023-01-02.txt",
				"prefix/20230103.txt",
				"prefix/invalid.txt",
				"prefix/mixed/", // directory
			},
			prefix:          "prefix/",
			dateFormat:      "200601",
			totalPartitions: 3,
			expected: [][]string{
				{"prefix/2023-01-02.txt", "prefix/invalid.txt"},
				{"prefix/20230103.txt", "prefix/20230101.txt"},
				{}, // no match
			},
		},
		{
			name: "all invalid dates",
			paths: []string{
				"prefix/notadate1.txt",
				"prefix/notadate2.txt",
				"prefix/notadate3.txt",
				"prefix/dir/", // directory
			},
			prefix:          "prefix/",
			dateFormat:      "2006-01-02",
			totalPartitions: 2,
			expected: [][]string{
				{"prefix/notadate1.txt", "prefix/notadate2.txt", "prefix/notadate3.txt"}, // all in fallback partition
				{},
			},
		},
		{
			name: "nested date paths",
			paths: []string{
				"prefix/region/2023-01-01/data.txt",
				"prefix/region/2023-01-02/data.txt",
				"prefix/region/invalid/data.txt",
				"prefix/backup/", // directory
			},
			prefix:          "prefix/region/",
			dateFormat:      "2006-01-02",
			totalPartitions: 3,
			expected: [][]string{
				{"prefix/region/invalid/data.txt"},
				{"prefix/region/2023-01-02/data.txt"},
				{"prefix/region/2023-01-01/data.txt"},
			},
		},
		{
			name: "date ranges across partition boundaries",
			paths: []string{
				"prefix/2022-12-31-data.txt",
				"prefix/2023-01-01-data.txt",
				"prefix/2023-01-02-data.txt",
				"prefix/dir/", // directory
			},
			prefix:          "prefix/",
			dateFormat:      "2006-01-02",
			totalPartitions: 2,
			expected: [][]string{
				{"prefix/2023-01-01-data.txt", "prefix/2022-12-31-data.txt"},
				{"prefix/2023-01-02-data.txt"},
			},
		},
		{
			name: "incomplete date formats",
			paths: []string{
				"prefix/2023-01.txt",    // missing day
				"prefix/2023.txt",       // missing month and day
				"prefix/2023-01-01.txt", // complete date
				"prefix/dir/",           // directory
			},
			prefix:          "prefix/",
			dateFormat:      "2006-01-02",
			totalPartitions: 3,
			expected: [][]string{
				{"prefix/2023-01.txt", "prefix/2023.txt"}, // fallback partition
				{}, // no match
				{"prefix/2023-01-01.txt"},
			},
		},
		{
			name: "different date separators",
			paths: []string{
				"prefix/2023.01.01.txt",
				"prefix/2023.01.02.txt",
				"prefix/2023.01.02.log",
				"prefix/2023.01.03.txt",
				"prefix/dir/", // directory
			},
			prefix:          "prefix/",
			dateFormat:      "2006.01.02",
			totalPartitions: 2,
			expected: [][]string{
				{"prefix/2023.01.01.txt", "prefix/2023.01.03.txt"},
				{"prefix/2023.01.02.txt", "prefix/2023.01.02.log"},
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			objects := createObjectsFromStrings(tt.paths)
			spec := SourceSpec{
				Key: tt.prefix,
				Partition: PartitionConfig{
					Type:       PartitionKeyTypeDate,
					DateFormat: tt.dateFormat,
				},
			}
			s.verifyPartitioning(spec, objects, tt.totalPartitions, tt.expected)
		})
	}
}

func (s *PartitionTestSuite) TestEdgeCases() {
	tests := []struct {
		name            string
		paths           []string
		spec            SourceSpec
		totalPartitions int
		expected        [][]string
		expectError     bool
		errorContains   string
	}{
		{
			name:  "empty object list",
			paths: []string{},
			spec: SourceSpec{
				Key: "prefix/",
				Partition: PartitionConfig{
					Type: PartitionKeyTypeObject,
				},
			},
			totalPartitions: 2,
			expected: [][]string{
				{}, // partition 0
				{}, // partition 1
			},
		},
		{
			name: "all directories",
			paths: []string{
				"prefix/dir1/",
				"prefix/dir2/",
				"prefix/dir3/",
			},
			spec: SourceSpec{
				Key: "prefix/",
				Partition: PartitionConfig{
					Type: PartitionKeyTypeObject,
				},
			},
			totalPartitions: 2,
			expected: [][]string{
				{}, // partition 0
				{}, // partition 1
			},
		},
		{
			name: "paths with special characters",
			paths: []string{
				"prefix/!@#$%^&*.txt",
				"prefix/spaces in name.txt",
				"prefix/tab\tin name.txt",
				"prefix/newline\nin name.txt",
				"prefix/escaped\\slash.txt",
			},
			spec: SourceSpec{
				Key: "prefix/",
				Partition: PartitionConfig{
					Type: PartitionKeyTypeObject,
				},
			},
			totalPartitions: 2,
			expected: [][]string{
				{"prefix/escaped\\slash.txt", "prefix/newline\nin name.txt", "prefix/tab\tin name.txt"},
				{"prefix/spaces in name.txt", "prefix/!@#$%^&*.txt"},
			},
		},
		{
			name: "very long keys",
			paths: []string{
				"prefix/" + strings.Repeat("a", 1000) + ".txt",
				"prefix/" + strings.Repeat("b", 2000) + ".txt",
				"prefix/" + strings.Repeat("c", 3000) + ".txt",
			},
			spec: SourceSpec{
				Key: "prefix/",
				Partition: PartitionConfig{
					Type: PartitionKeyTypeObject,
				},
			},
			totalPartitions: 3,
			expected: [][]string{
				{"prefix/" + strings.Repeat("c", 3000) + ".txt"},
				{"prefix/" + strings.Repeat("b", 2000) + ".txt"},
				{"prefix/" + strings.Repeat("a", 1000) + ".txt"},
			},
		},
		{
			name: "mixed unicode scripts",
			paths: []string{
				"prefix/文件1.txt",   // Chinese
				"prefix/файл2.txt", // Russian
				"prefix/파일3.txt",   // Korean
				"prefix/ファイル4.txt", // Japanese
				"prefix/ملف5.txt",  // Arabic
			},
			spec: SourceSpec{
				Key: "prefix/",
				Partition: PartitionConfig{
					Type: PartitionKeyTypeObject,
				},
			},
			totalPartitions: 3,
			expected: [][]string{
				{"prefix/файл2.txt"},
				{"prefix/ファイル4.txt"},
				{"prefix/文件1.txt", "prefix/파일3.txt", "prefix/ملف5.txt"},
			},
		},
		{
			name: "paths with empty segments",
			paths: []string{
				"prefix//file1.txt",   // double slash
				"prefix/./file2.txt",  // current dir
				"prefix/../file3.txt", // parent dir
				"prefix/.file4.txt",   // hidden file
				"prefix/ file5.txt",   // leading space
				"prefix/file6.txt ",   // trailing space
			},
			spec: SourceSpec{
				Key: "prefix/",
				Partition: PartitionConfig{
					Type: PartitionKeyTypeObject,
				},
			},
			totalPartitions: 2,
			expected: [][]string{
				{"prefix//file1.txt", "prefix/../file3.txt", "prefix/file6.txt ", "prefix/.file4.txt"},
				{"prefix/./file2.txt", "prefix/ file5.txt"},
			},
		},
		{
			name: "mixed path separators",
			paths: []string{
				"prefix\\file1.txt",
				"prefix/file2.txt",
				"prefix\\sub\\file3.txt",
				"prefix/sub/file4.txt",
			},
			spec: SourceSpec{
				Key: "prefix/",
				Partition: PartitionConfig{
					Type: PartitionKeyTypeObject,
				},
			},
			totalPartitions: 2,
			expected: [][]string{
				{"prefix\\file1.txt", "prefix/file2.txt", "prefix\\sub\\file3.txt"},
				{"prefix/sub/file4.txt"},
			},
		},
		{
			name: "substring with zero-width characters",
			paths: []string{
				"prefix/\u200Bfile1.txt", // zero-width space
				"prefix/\uFEFFfile2.txt", // byte order mark
				"prefix/\u200Efile3.txt", // zero-width non-joiner
			},
			spec: SourceSpec{
				Key: "prefix/",
				Partition: PartitionConfig{
					Type:       PartitionKeyTypeSubstring,
					StartIndex: 0,
					EndIndex:   4,
				},
			},
			totalPartitions: 2,
			expected: [][]string{
				{"prefix/\u200Efile3.txt"},
				{"prefix/\u200Bfile1.txt", "prefix/\uFEFFfile2.txt"},
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			objects := createObjectsFromStrings(tt.paths)
			if tt.expectError {
				_, err := PartitionObjects(objects, tt.totalPartitions, 0, tt.spec)
				s.Require().Error(err)
				if tt.errorContains != "" {
					s.Contains(err.Error(), tt.errorContains)
				}
				return
			}
			s.verifyPartitioning(tt.spec, objects, tt.totalPartitions, tt.expected)
		})
	}
}

func createObjectSummary(key string, isDir bool) ObjectSummary {
	return ObjectSummary{
		Key:   &key,
		IsDir: isDir,
	}
}

// createObjectsFromStrings creates ObjectSummary slices from strings, treating paths with / suffix as directories
func createObjectsFromStrings(paths []string) []ObjectSummary {
	objects := make([]ObjectSummary, len(paths))
	for i, path := range paths {
		isDir := strings.HasSuffix(path, "/")
		objects[i] = createObjectSummary(path, isDir)
	}
	return objects
}

// verifyPartitionContents checks if actual partitions match expected content
func (s *PartitionTestSuite) verifyPartitionContents(actualPartitions [][]ObjectSummary, expectedPartitions [][]string) {
	s.Require().Equal(len(expectedPartitions), len(actualPartitions), "partition count mismatch")

	for i := range expectedPartitions {
		actualPaths := make([]string, len(actualPartitions[i]))
		for j, obj := range actualPartitions[i] {
			actualPaths[j] = *obj.Key
		}

		s.ElementsMatch(expectedPartitions[i], actualPaths,
			"partition %d content mismatch. Expected %s but found %s", i, expectedPartitions[i], actualPaths)
	}
}

// Helper methods for verification
func (s *PartitionTestSuite) verifyNoDirectories(partitions [][]ObjectSummary) {
	for _, partition := range partitions {
		for _, obj := range partition {
			s.False(obj.IsDir, "Directory found in partition: %s", *obj.Key)
		}
	}
}

func (s *PartitionTestSuite) verifyComplete(objects []ObjectSummary, partitions [][]ObjectSummary) {
	expectedCount := 0
	for _, obj := range objects {
		if !obj.IsDir {
			expectedCount++
		}
	}

	actualCount := 0
	for _, partition := range partitions {
		actualCount += len(partition)
	}

	s.Equal(expectedCount, actualCount, "Not all objects were distributed to partitions")
}

func (s *PartitionTestSuite) verifyConsistency(spec SourceSpec, objects []ObjectSummary, totalPartitions int) {
	initialPartitions := make([][]ObjectSummary, totalPartitions)
	for i := 0; i < totalPartitions; i++ {
		partition, err := PartitionObjects(objects, totalPartitions, i, spec)
		s.Require().NoError(err)
		initialPartitions[i] = partition
	}

	// Run multiple times to verify consistent distribution
	for run := 0; run < 3; run++ {
		for i := 0; i < totalPartitions; i++ {
			partition, err := PartitionObjects(objects, totalPartitions, i, spec)
			s.Require().NoError(err)
			s.Equal(len(initialPartitions[i]), len(partition), "Partition size changed on subsequent run")
			for j := range initialPartitions[i] {
				s.Equal(*initialPartitions[i][j].Key, *partition[j].Key, "Object distribution changed on subsequent run")
			}
		}
	}
}

func (s *PartitionTestSuite) verifyPartitioning(spec SourceSpec, objects []ObjectSummary, totalPartitions int, expected [][]string) {
	s.Require().NotNil(expected, "expected partition contents must not be nil")
	s.Require().Equal(totalPartitions, len(expected), "expected partition count must match totalPartitions")

	partitions := make([][]ObjectSummary, totalPartitions)
	for i := 0; i < totalPartitions; i++ {
		partition, err := PartitionObjects(objects, totalPartitions, i, spec)
		s.Require().NoError(err)
		partitions[i] = partition
	}

	s.verifyNoDirectories(partitions)
	s.verifyComplete(objects, partitions)
	s.verifyConsistency(spec, objects, totalPartitions)
	if expected != nil {
		s.verifyPartitionContents(partitions, expected)
	}
}
