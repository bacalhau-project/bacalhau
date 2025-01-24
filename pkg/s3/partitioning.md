# S3 Object Partitioning

This documentation describes the partitioning system for S3 objects, which enables efficient distribution of object processing across multiple workers. The system supports multiple partitioning strategies to accommodate different use cases and data organization patterns.

## Overview

The partitioning system allows you to split a collection of S3 objects across multiple processors using deterministic hashing. This ensures:
- Even distribution of objects
- Deterministic assignment of objects to partitions
- Support for various partitioning strategies
- Graceful handling of edge cases

## Partition Types

### 1. None (`PartitionKeyTypeNone`)
- No partitioning is applied
- All objects are processed as a single group
- Useful when partitioning is not needed or when handling small datasets

### 2. Object (`PartitionKeyTypeObject`)
- Partitions based on the complete object key
- Uses deterministic hashing of the entire object path
- Best for random or unpredictable key patterns
- Ensures even distribution across partitions

### 3. Regex (`PartitionKeyTypeRegex`)
- Partitions using regex pattern matches from object keys
- Configuration requires:
    - `Pattern`: A valid regex pattern
- Behavior:
    - For patterns with capture groups:
        - Combines all non-empty capture groups with a special delimiter (ASCII Unit Separator)
        - Hashes the combined string to determine partition
    - For patterns without capture groups:
        - Uses the full matched string for hashing
        - e.g., pattern `00\d` will use "001" for partitioning if it matches "001.txt"
    - Falls back to partition 0 if no match is found in the key
- Useful for:
    - Complex naming patterns requiring multiple parts (using capture groups)
    - Simple pattern matching (without capture groups)
    - Extracting specific parts of object keys

### 4. Substring (`PartitionKeyTypeSubstring`)
- Partitions based on a specific portion of the object key
- Configuration requires:
    - `StartIndex`: Beginning of substring (inclusive)
    - `EndIndex`: End of substring (exclusive)
- Validates that:
    - StartIndex ≥ 0
    - EndIndex > StartIndex
- Falls back to partition 0 if the key is shorter than EndIndex
- Useful for keys with fixed-width segments or known positions

### 5. Date (`PartitionKeyTypeDate`)
- Partitions based on dates found in object keys
- Configuration requires:
    - `DateFormat`: Go time format string (e.g., "2006-01-02")
- Behavior:
    - Attempts to parse date from the beginning of the key
    - Uses parsed date formatted back to string for hashing
    - Falls back to partition 0 if date parsing fails
- Ideal for time-series data or date-based organization

## Configuration

### PartitionConfig Structure
```go
type PartitionConfig struct {
    Type        PartitionKeyType
    Pattern     string    // For regex partitioning
    StartIndex  int      // For substring partitioning
    EndIndex    int      // For substring partitioning
    DateFormat  string   // For date partitioning
}
```

### Validation Rules
- Type must be one of the supported PartitionKeyTypes
- Regex partitioning:
    - Pattern cannot be empty
    - Pattern must be a valid regex
- Substring partitioning:
    - StartIndex must be non-negative
    - EndIndex must be greater than StartIndex
- Date partitioning:
    - DateFormat cannot be empty
    - DateFormat must be a valid Go time format

## Implementation Details

### Key Processing
1. Directories are filtered out before partitioning
2. Object keys are sanitized by:
    - Trimming the common prefix
    - Removing leading slashes
3. Special handling for single partition case:
    - If total partitions = 1, returns all objects without processing

### deterministic Hashing
- Uses FNV-1a hash algorithm
- Ensures deterministic distribution
- Formula: `partition_index = hash(key) % total_partitions`

### Error Handling
- Validates configuration before processing
- Provides fallback behavior for edge cases
- Returns descriptive error messages for invalid configurations

## Usage Examples

### Regex Partitioning
#### Regex Partitioning with Capture Groups
```go
config := PartitionConfig{
    Type:    PartitionKeyTypeRegex,
    Pattern: `data/(\d{4})/(\d{2})/.*\.csv`,
}
```
This will partition based on year and month capture groups in paths like "data/2024/01/file.csv"

#### Regex Partitioning without Capture Groups
```go
config := PartitionConfig{
    Type:    PartitionKeyTypeRegex,
    Pattern: `00\d\.txt`,
}
```
This will partition based on the full matched string (e.g., "001.txt", "002.txt")`


### Date Partitioning
```go
config := PartitionConfig{
    Type:       PartitionKeyTypeDate,
    DateFormat: "2006-01-02",
}
```
This will partition objects with keys starting with dates like "2024-01-15-data.csv"

### Substring Partitioning
```go
config := PartitionConfig{
    Type:      PartitionKeyTypeSubstring,
    StartIndex: 5,
    EndIndex:   13,
}
```
This will partition based on characters 5-12 of the object key

## Prefix Trimming Logic

Before applying any partitioning strategy, the system processes object keys by removing the common prefix. This is handled by the `sanitizeKeyForPatternMatching` function using the following steps:

1. First, the source prefix is sanitized:
    - Whitespace is trimmed
    - Trailing wildcards (*) are removed

2. Then, for each object key:
    - The sanitized prefix is removed from the beginning of the key
    - Any leading forward slash (/) is removed

### Prefix Trimming Examples

Given source prefix: `"data/users/*"`
```
Original Key                    | Trimmed Key (used for partitioning)
-------------------------------|--------------------------------
data/users/2024/file.csv       | 2024/file.csv
data/users/archived/2023.csv   | archived/2023.csv
data/users//extra/file.csv     | extra/file.csv
other/data/users/file.csv      | other/data/users/file.csv  (no match)
```

Given source prefix: `"logs/"`
```
Original Key                    | Trimmed Key (used for partitioning)
-------------------------------|--------------------------------
logs/2024-01-15/server.log     | 2024-01-15/server.log
logs//app/debug.log            | app/debug.log
logs/error.log                 | error.log
archive/logs/old.log           | archive/logs/old.log  (no match)
```

Given source prefix: `""` (empty)
```
Original Key                    | Trimmed Key (used for partitioning)
-------------------------------|--------------------------------
data.csv                       | data.csv  (unchanged)
folder/file.txt                | folder/file.txt  (unchanged)
/root/file.bin                 | root/file.bin
```

### Impact on Partitioning Strategies

The prefix trimming affects how each partition type processes keys:

1. **Regex Partitioning**
    - Pattern matches against the trimmed key
    - Example with prefix `"data/"` and pattern `(\d{4})/(\d{2})`:
      ```
      Original: data/2024/01/file.csv
      Trimmed:  2024/01/file.csv
      Matches:  ["2024", "01"]
      ```

2. **Substring Partitioning**
    - Start and end indices apply to the trimmed key
    - Example with prefix `"logs/"` and indices (0,10):
      ```
      Original: logs/2024-01-15-server.log
      Trimmed:  2024-01-15-server.log
      Substring: "2024-01-15"
      ```

3. **Date Partitioning**
    - Date parsing starts at the beginning of the trimmed key
    - Example with prefix `"archive/"`:
      ```
      Original: archive/2024-01-15_backup.tar
      Trimmed:  2024-01-15_backup.tar
      Date Portion: "2024-01-15"
      ```

### Best Practices for Prefix Usage

1. Always consider the full path when designing prefixes:
    - Include parent directories if they're part of the pattern
    - Account for possible variations in path depth

2. Be cautious with trailing slashes:
    - They affect how the trimming behaves
    - Consider standardizing their usage in your application

3. When using wildcards:
    - They're automatically trimmed from the prefix
    - Use them to match variable portions of paths

## Best Practices

1. Choose the appropriate partition type based on your data organization:
    - Use Object for random or unpredictable keys
    - Use Regex for complex patterns requiring multiple parts
    - Use Substring for fixed-width segments
    - Use Date for time-series data

2. Consider fallback behavior:
    - All strategies fall back to partition 0 for unmatched cases
    - Design key patterns to minimize fallback scenarios

3. Performance considerations:
    - Regex partitioning has higher computational overhead
    - Substring and Date partitioning are more efficient
    - Object partitioning provides the most even distribution

4. Testing recommendations:
    - Validate partition distribution with sample data
    - Test edge cases and fallback scenarios
    - Verify date formats with different timezone scenarios
