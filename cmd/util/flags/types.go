package flags

import (
	"fmt"
	"strings"

	"github.com/spf13/pflag"
	"golang.org/x/exp/slices"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	storage_ipfs "github.com/bacalhau-project/bacalhau/pkg/storage/ipfs"

	"github.com/bacalhau-project/bacalhau/cmd/util/output"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	storage_url "github.com/bacalhau-project/bacalhau/pkg/storage/url/urldownload"
)

// A Parser is a function that can convert a string into a native object.
type Parser[T any] func(string) (T, error)

// A KeyValueParser is like a Parser except that it returns two values
// representing a key and a value.
type KeyValueParser[K comparable, V any] func(string) (K, V, error)

// A Stringer is a function that can convert a native object into a string.
type Stringer[T any] func(*T) string

// A KeyValueStringer is like a Stringer except that it converts native objects
// representing a key and a value into a string.
type KeyValueStringer[K comparable, V any] func(*K, *V) string

// A ValueFlag is a pflag.Value that knows how to take a command line value
// represented as a string and set it as a native object into a struct.
type ValueFlag[T any] struct {
	// A pointer to a variable that will be set by this flag.
	// This will be a pointer to some struct value we want to set.
	value *T

	// A Parser to turn the command line string into a native value.
	parser Parser[T]

	// A Stringer to turn the default value for the flag back into a native
	// string, to be printed as help.
	stringer Stringer[T]

	// How the value should be described in the help string. (e.g. string, int)
	typeStr string
}

// Set implements pflag.Value
func (s *ValueFlag[T]) Set(input string) error {
	value, err := s.parser(input)
	*s.value = value
	return err
}

// String implements pflag.Value
func (s *ValueFlag[T]) String() string {
	return s.stringer(s.value)
}

// Type implements pflag.Value
func (s *ValueFlag[T]) Type() string {
	return s.typeStr
}

var _ pflag.Value = (*ValueFlag[int])(nil)

// An ArrayValueFlag is like a ValueFlag except it will add the command line
// value into a slice of values, and hence can be used for flags that are meant
// to appear multiple times.
type ArrayValueFlag[T any] struct {
	// A pointer to a variable that will be set by this flag.
	// This will be a pointer to some struct value we want to set.
	value *[]T

	// A Parser to turn the command line string into a native value.
	parser Parser[T]

	// A Stringer to turn the default value for the flag back into a native
	// string, to be printed as help.
	stringer Stringer[T]

	// How the value should be described in the help string. (e.g. string, int)
	typeStr string
}

// Set implements pflag.Value
func (s *ArrayValueFlag[T]) Set(input string) error {
	value, err := s.parser(input)
	*s.value = append(*s.value, value)
	return err
}

// String implements pflag.Value
func (s *ArrayValueFlag[T]) String() string {
	strs := make([]string, 0, len(*s.value))
	for _, spec := range *s.value {
		strs = append(strs, s.stringer(&spec))
	}
	return strings.Join(strs, ", ")
}

// Type implements pflag.Value
func (s *ArrayValueFlag[T]) Type() string {
	return s.typeStr
}

// Converts a value flag into a flag that can accept multiple of the same value.
func ArrayValueFlagFrom[T any](singleFlag func(*T) *ValueFlag[T]) func(*[]T) *ArrayValueFlag[T] {
	flag := singleFlag(nil)
	return func(value *[]T) *ArrayValueFlag[T] {
		return &ArrayValueFlag[T]{
			value:    value,
			parser:   flag.parser,
			stringer: flag.stringer,
			typeStr:  flag.typeStr,
		}
	}
}

var _ pflag.Value = (*ArrayValueFlag[int])(nil)

// A MapValueFlag is like a ValueFlag except it will add the command line
// value into a map of values, and hence can be used for flags that are meant
// to appear multiple times and represent a key-value structure.
type MapValueFlag[K comparable, V any] struct {
	// A pointer to a variable that will be set by this flag.
	// This will be a pointer to some struct value we want to set.
	value *map[K]V

	// A Parser to turn the command line string into a native value.
	parser KeyValueParser[K, V]

	// A Stringer to turn the default value for the flag back into a native
	// string, to be printed as help.
	stringer KeyValueStringer[K, V]

	// How the value should be described in the help string. (e.g. string, int)
	typeStr string
}

// Set implements pflag.Value
func (s *MapValueFlag[K, V]) Set(input string) error {
	key, value, err := s.parser(input)
	(*s.value)[key] = value
	return err
}

// String implements pflag.Value
func (s *MapValueFlag[K, V]) String() string {
	strs := make([]string, len(*s.value))
	for key, value := range *s.value {
		strs = append(strs, s.stringer(&key, &value))
	}
	return strings.Join(strs, ", ")
}

// Type implements pflag.Value
func (s *MapValueFlag[K, V]) Type() string {
	return s.typeStr
}

var _ pflag.Value = (*MapValueFlag[int, int])(nil)

func SeparatorParser(sep string) KeyValueParser[string, string] {
	return func(input string) (string, string, error) {
		slices := strings.Split(input, sep)
		if len(slices) != 2 {
			return "", "", fmt.Errorf("%q should contain exactly one %s", input, sep)
		}
		return slices[0], slices[1], nil
	}
}

func parseIPFSStorageSpec(input string) (*models.InputSource, error) {
	cid, path, err := SeparatorParser(":")(input)
	if err != nil {
		return nil, err
	}
	ipfsSpec, err := storage_ipfs.NewSpecConfig(cid)
	if err != nil {
		return nil, err
	}
	return &models.InputSource{
		Source: ipfsSpec,
		Alias:  fmt.Sprintf("ipfs://%s", cid),
		Target: path,
	}, nil
}

func NewIPFSStorageSpecArrayFlag(value *[]*models.InputSource) *ArrayValueFlag[*models.InputSource] {
	return &ArrayValueFlag[*models.InputSource]{
		value:    value,
		parser:   parseIPFSStorageSpec,
		stringer: func(s **models.InputSource) string { return (*s).Source.Type },
		typeStr:  "cid:path",
	}
}

func parseURLStorageSpec(inputURL string) (*models.InputSource, error) {
	u, err := storage_url.IsURLSupported(inputURL)
	if err != nil {
		return nil, err
	}
	urlSpec, err := storage_url.NewSpecConfig(u.String())
	if err != nil {
		return nil, err
	}
	return &models.InputSource{
		Source: urlSpec,
		Alias:  u.String(),
		Target: "/inputs",
	}, nil
}

func NewURLStorageSpecArrayFlag(value *[]*models.InputSource) *ArrayValueFlag[*models.InputSource] {
	return &ArrayValueFlag[*models.InputSource]{
		value:    value,
		parser:   parseURLStorageSpec,
		stringer: func(s **models.InputSource) string { return (*s).Source.Type },
		typeStr:  "url",
	}
}

type TargetingMode bool

const (
	TargetAny TargetingMode = false
	TargetAll TargetingMode = true
)

func (t TargetingMode) String() string {
	if bool(t) {
		return "all"
	} else {
		return "any"
	}
}

func ParseTargetingMode(s string) (TargetingMode, error) {
	switch s {
	case "any":
		return TargetAny, nil
	case "all":
		return TargetAll, nil
	default:
		return TargetAny, fmt.Errorf(`expecting "any" or "all", not %q`, s)
	}
}

func TargetingFlag(value *TargetingMode) *ValueFlag[TargetingMode] {
	return &ValueFlag[TargetingMode]{
		value:    value,
		parser:   ParseTargetingMode,
		stringer: func(tm *TargetingMode) string { return tm.String() },
		typeStr:  "all|any",
	}
}

func DataLocalityFlag(value *models.JobSelectionDataLocality) *ValueFlag[models.JobSelectionDataLocality] {
	return &ValueFlag[models.JobSelectionDataLocality]{
		value:    value,
		parser:   models.ParseJobSelectionDataLocality,
		stringer: func(l *models.JobSelectionDataLocality) string { return l.String() },
		typeStr:  "local|anywhere",
	}
}

func LoggingFlag(value *logger.LogMode) *ValueFlag[logger.LogMode] {
	return &ValueFlag[logger.LogMode]{
		value:    value,
		parser:   logger.ParseLogMode,
		stringer: func(p *logger.LogMode) string { return string(*p) },
		typeStr:  "logging-mode",
	}
}

func OutputFormatFlag(value *output.OutputFormat) *ValueFlag[output.OutputFormat] {
	return &ValueFlag[output.OutputFormat]{
		value: value,
		parser: func(s string) (output.OutputFormat, error) {
			o := output.OutputFormat(s)
			if !slices.Contains(output.AllFormats, o) {
				return "", fmt.Errorf("should be one of %q", output.AllFormats)
			}
			return o, nil
		},
		stringer: func(o *output.OutputFormat) string { return string(*o) },
		typeStr:  "format",
	}
}

func ResultPathFlag(value *[]*models.ResultPath) *ArrayValueFlag[*models.ResultPath] {
	return &ArrayValueFlag[*models.ResultPath]{
		value:  value,
		parser: parseResultPath,
		stringer: func(r **models.ResultPath) string {
			return fmt.Sprintf("%+v", r)
		},
		typeStr: "ResultPath",
	}
}

func parseResultPath(value string) (*models.ResultPath, error) {
	tokens := strings.Split(value, ":")
	if len(tokens) != 2 || tokens[0] == "" || tokens[1] == "" {
		return nil, fmt.Errorf("invalid output volume: %s", value)
	}
	return &models.ResultPath{
		Name: tokens[0],
		Path: tokens[1],
	}, nil
}

func NetworkFlag(value *models.Network) *ValueFlag[models.Network] {
	return &ValueFlag[models.Network]{
		value:    value,
		parser:   models.ParseNetwork,
		stringer: func(n *models.Network) string { return n.String() },
		typeStr:  "network-type",
	}
}
