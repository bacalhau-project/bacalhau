package s3

import (
	"fmt"
	"hash/fnv"
	"regexp"
	"strings"
	"time"
)

const (
	// fallbackPartitionIndex We’ll assign any “unmatched” object to this partition index
	fallbackPartitionIndex = 0

	// captureGroupDelimiter is ASCII Unit Separator (0x1F).
	// Used to join regex capture groups when computing partition hash.
	// This character is not allowed in S3 object keys (which only permit ASCII 32-126, plus 128-255),
	// making it a safe delimiter that won't conflict with key contents.
	captureGroupDelimiter = "\x1F"
)

// PartitionKeyType represents the type of partitioning to apply
type PartitionKeyType string

const (
	PartitionKeyTypeNone      PartitionKeyType = "none"
	PartitionKeyTypeObject    PartitionKeyType = "object"
	PartitionKeyTypeRegex     PartitionKeyType = "regex"
	PartitionKeyTypeSubstring PartitionKeyType = "substring"
	PartitionKeyTypeDate      PartitionKeyType = "date"
)

// PartitionConfig defines how to generate partition keys from object paths
type PartitionConfig struct {
	Type PartitionKeyType

	// For regex partitioning
	Pattern string

	// For substring partitioning
	StartIndex int
	EndIndex   int

	// For date partitioning
	DateFormat string
}

func (c *PartitionConfig) Validate() error {
	// First validate the partition type itself
	switch c.Type {
	case PartitionKeyTypeNone, PartitionKeyTypeObject, PartitionKeyTypeRegex,
		PartitionKeyTypeSubstring, PartitionKeyTypeDate:
		// Valid types
	default:
		if c.Type != "" {
			return NewS3InputSourceError(BadRequestErrorCode, fmt.Sprintf("unsupported partition key type %s", c.Type))
		}
	}

	// Then validate type-specific configurations
	switch c.Type {
	case PartitionKeyTypeRegex:
		if c.Pattern == "" {
			return NewS3InputSourceError(BadRequestErrorCode, "regex pattern cannot be empty")
		}
		if _, err := regexp.Compile(c.Pattern); err != nil {
			return NewS3InputSourceError(BadRequestErrorCode, fmt.Sprintf("invalid regex pattern: %s", err.Error()))
		}

	case PartitionKeyTypeSubstring:
		if c.StartIndex < 0 {
			return NewS3InputSourceError(BadRequestErrorCode, "start index cannot be negative")
		}
		if c.EndIndex <= c.StartIndex {
			return NewS3InputSourceError(BadRequestErrorCode, "end index must be greater than start index")
		}

	case PartitionKeyTypeDate:
		if c.DateFormat == "" {
			return NewS3InputSourceError(BadRequestErrorCode, "date format cannot be empty")
		}

		if err := validateDateFormat(c.DateFormat); err != nil {
			return err
		}
	}
	return nil
}

// validateDateFormat validates the date format string
func validateDateFormat(layout string) error {
	// Reference time: Jan 2, 2006 at 15:04:05 UTC
	ref := time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC)

	// Format the reference time using the layout
	s := ref.Format(layout)

	// Parse the resulting string back into a time
	_, err := time.Parse(layout, s)
	if err != nil {
		// If parsing fails, the layout is invalid
		return NewS3InputSourceError(BadRequestErrorCode, fmt.Sprintf("invalid date format: %s", layout))
	}

	return nil
}

// PartitionObjects applies the configured partitioning strategy to a slice of objects
func PartitionObjects(
	objects []ObjectSummary,
	totalPartitions int,
	partitionIndex int,
	source SourceSpec,
) ([]ObjectSummary, error) {
	if err := source.Partition.Validate(); err != nil {
		return nil, err
	}
	if totalPartitions <= 0 {
		return nil, NewS3InputSourceError(BadRequestErrorCode, "job partitions/count must be greater than 0")
	}
	if partitionIndex < 0 || partitionIndex >= totalPartitions {
		return nil, NewS3InputSourceError(
			BadRequestErrorCode, fmt.Sprintf("partition index must be between 0 and %d", totalPartitions-1))
	}

	// filter out directories
	objects = filterDirectories(objects)

	// If there is only 1 partition, just return the entire set
	if totalPartitions == 1 {
		return objects, nil
	}

	// Handle both empty and "none" types the same way
	if source.Partition.Type == "" || source.Partition.Type == PartitionKeyTypeNone {
		// Return all objects unmodified for both empty and "none" types
		return objects, nil
	}

	// Sanitize the prefix for pattern matching
	prefix := strings.TrimSpace(source.Key)
	prefix = strings.TrimSuffix(prefix, "*")

	switch source.Partition.Type {
	case PartitionKeyTypeObject:
		return partitionByObject(objects, totalPartitions, partitionIndex)
	case PartitionKeyTypeRegex:
		return partitionByRegex(objects, totalPartitions, partitionIndex, prefix, source.Partition.Pattern)
	case PartitionKeyTypeSubstring:
		return partitionBySubstring(
			objects, totalPartitions, partitionIndex, prefix, source.Partition.StartIndex, source.Partition.EndIndex)
	case PartitionKeyTypeDate:
		return partitionByDate(
			objects, totalPartitions, partitionIndex, prefix, source.Partition.DateFormat)
	default:
		return nil, NewS3InputSourceError(BadRequestErrorCode, fmt.Sprintf("unsupported partition key type: %s", source.Partition.Type))
	}
}

// partitionByObject partitions objects by hashing their full key
func partitionByObject(objects []ObjectSummary, totalPartitions int, partitionIndex int) ([]ObjectSummary, error) {
	var result []ObjectSummary
	for _, obj := range objects {
		if getPartitionIndex(*obj.Key, totalPartitions) == partitionIndex {
			result = append(result, obj)
		}
	}
	return result, nil
}

// partitionByRegex partitions objects by regex matching. If the pattern contains
// capture groups, partitioning is based on the concatenated capture groups.
// Otherwise, partitioning is based on the full match. Objects that don't match
// the pattern are assigned to partition 0.
func partitionByRegex(objects []ObjectSummary, totalPartitions int, partitionIndex int, prefix, pattern string) (
	[]ObjectSummary, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, NewS3InputSourceError(BadRequestErrorCode, fmt.Sprintf("invalid regex pattern: %s", err.Error()))
	}

	var result []ObjectSummary
	for _, obj := range objects {
		key := sanitizeKeyForPatternMatching(*obj.Key, prefix)
		matches := re.FindStringSubmatch(key)

		// Collect non-empty capture groups
		var matchGroups []string
		for i := 1; i < len(matches); i++ {
			if matches[i] != "" {
				matchGroups = append(matchGroups, matches[i])
			}
		}

		var pIndex int
		if len(matches) == 0 || matches[0] == "" {
			// No match at all or empty match - use fallback
			pIndex = fallbackPartitionIndex
		} else if len(matchGroups) == 0 {
			// No capture groups - use full match
			pIndex = getPartitionIndex(matches[0], totalPartitions)
		} else {
			// Use capture groups
			combinedKey := strings.Join(matchGroups, captureGroupDelimiter)
			pIndex = getPartitionIndex(combinedKey, totalPartitions)
		}

		if pIndex == partitionIndex {
			result = append(result, obj)
		}
	}
	return result, nil
}

// partitionBySubstring partitions objects by taking a substring of their key
func partitionBySubstring(
	objects []ObjectSummary, totalPartitions int, partitionIndex int, prefix string, startIndex int, endIndex int) (
	[]ObjectSummary, error) {
	var result []ObjectSummary
	for _, obj := range objects {
		key := sanitizeKeyForPatternMatching(*obj.Key, prefix)

		// Convert to rune slice for proper Unicode handling
		runes := []rune(key)

		var pIndex int
		if len(runes) < endIndex {
			pIndex = fallbackPartitionIndex
		} else {
			substr := string(runes[startIndex:endIndex])
			pIndex = getPartitionIndex(substr, totalPartitions)
		}

		if pIndex == partitionIndex {
			result = append(result, obj)
		}
	}
	return result, nil
}

// partitionByDate partitions objects by parsing dates from their keys
func partitionByDate(
	objects []ObjectSummary, totalPartitions, partitionIndex int, prefix, dateFormat string) (
	[]ObjectSummary, error) {
	var result []ObjectSummary
	for _, obj := range objects {
		key := sanitizeKeyForPatternMatching(*obj.Key, prefix)

		// We'll parse the first len(dateFormat) characters as the date
		dateStr := key[:min(len(key), len(dateFormat))]
		t, err := time.Parse(dateFormat, dateStr)

		var pIndex int
		if err != nil {
			// If it fails to parse, fallback to partition 0
			pIndex = fallbackPartitionIndex
		} else {
			dateKey := t.Format(dateFormat)
			pIndex = getPartitionIndex(dateKey, totalPartitions)
		}

		if pIndex == partitionIndex {
			result = append(result, obj)
		}
	}
	return result, nil
}

// sanitizeKeyForPatternMatching returns the relative path after the prefix
func sanitizeKeyForPatternMatching(objectKey string, prefix string) string {
	key := strings.TrimPrefix(objectKey, prefix)
	return strings.TrimPrefix(key, "/")
}

// getPartitionIndex returns the partition index for a given key using consistent hashing
func getPartitionIndex(key string, totalPartitions int) int {
	if totalPartitions <= 0 {
		return 0
	}
	hash := hashString(key)
	return int(hash % uint32(totalPartitions)) //nolint:gosec
}

// hashString returns a uint32 hash of the input string using FNV-1a
func hashString(s string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(s))
	return h.Sum32()
}

// filterDirectories filters out directories from the list of objects
func filterDirectories(objects []ObjectSummary) []ObjectSummary {
	var result []ObjectSummary
	for _, obj := range objects {
		if !obj.IsDir {
			result = append(result, obj)
		}
	}
	return result
}
