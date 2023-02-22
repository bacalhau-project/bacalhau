package bacalhau

import (
	"fmt"
	"strings"

	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/storage/url/urldownload"
	"github.com/spf13/pflag"
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
		spec := spec
		strs = append(strs, s.stringer(&spec))
	}
	return strings.Join(strs, ", ")
}

// Type implements pflag.Value
func (s *ArrayValueFlag[T]) Type() string {
	return s.typeStr
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
		key, value := key, value
		strs = append(strs, s.stringer(&key, &value))
	}
	return strings.Join(strs, ", ")
}

// Type implements pflag.Value
func (s *MapValueFlag[K, V]) Type() string {
	return s.typeStr
}

var _ pflag.Value = (*MapValueFlag[int, int])(nil)

func separatorParser(sep string) KeyValueParser[string, string] {
	return func(input string) (string, string, error) {
		slices := strings.Split(input, sep)
		if len(slices) != 2 {
			return "", "", fmt.Errorf("%q should contain exactly one %s", input, sep)
		}
		return slices[0], slices[1], nil
	}
}

func parseIPFSStorageSpec(input string) (model.StorageSpec, error) {
	cid, path, err := separatorParser(":")(input)
	return model.StorageSpec{
		StorageSource: model.StorageSourceIPFS,
		CID:           cid,
		Path:          path,
	}, err
}

func storageSpecToIPFSMount(input *model.StorageSpec) string {
	return fmt.Sprintf("%s:%s", input.CID, input.Path)
}

func NewIPFSStorageSpecArrayFlag(value *[]model.StorageSpec) *ArrayValueFlag[model.StorageSpec] {
	return &ArrayValueFlag[model.StorageSpec]{
		value:    value,
		parser:   parseIPFSStorageSpec,
		stringer: storageSpecToIPFSMount,
		typeStr:  "cid:path",
	}
}

func parseURLStorageSpec(inputURL string) (model.StorageSpec, error) {
	u, err := urldownload.IsURLSupported(inputURL)
	if err != nil {
		return model.StorageSpec{}, err
	}
	return model.StorageSpec{
		StorageSource: model.StorageSourceURLDownload,
		URL:           u.String(),
		Path:          "/inputs",
	}, nil
}

func NewURLStorageSpecArrayFlag(value *[]model.StorageSpec) *ArrayValueFlag[model.StorageSpec] {
	return &ArrayValueFlag[model.StorageSpec]{
		value:    value,
		parser:   parseURLStorageSpec,
		stringer: func(s *model.StorageSpec) string { return s.URL },
		typeStr:  "url",
	}
}

func VerifierFlag(value *model.Verifier) *ValueFlag[model.Verifier] {
	return &ValueFlag[model.Verifier]{
		value:    value,
		parser:   model.ParseVerifier,
		stringer: func(v *model.Verifier) string { return v.String() },
		typeStr:  "verifier",
	}
}

func PublisherFlag(value *model.Publisher) *ValueFlag[model.Publisher] {
	return &ValueFlag[model.Publisher]{
		value:    value,
		parser:   model.ParsePublisher,
		stringer: func(p *model.Publisher) string { return p.String() },
		typeStr:  "publisher",
	}
}

func NetworkFlag(value *model.Network) *ValueFlag[model.Network] {
	return &ValueFlag[model.Network]{
		value:    value,
		parser:   model.ParseNetwork,
		stringer: func(n *model.Network) string { return n.String() },
		typeStr:  "network-type",
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

func EnvVarMapFlag(value *map[string]string) *MapValueFlag[string, string] {
	return &MapValueFlag[string, string]{
		value:    value,
		parser:   separatorParser("="),
		stringer: func(k *string, v *string) string { return fmt.Sprintf("%s=%s", *k, *v) },
		typeStr:  "key=value",
	}
}

func parseTag(s string) (string, error) {
	var err error
	if !job.IsSafeAnnotation(s) {
		err = fmt.Errorf("%q is not a valid tag", s)
	}
	return s, err
}

func IncludedTagFlag(value *[]model.IncludedTag) *ArrayValueFlag[model.IncludedTag] {
	return &ArrayValueFlag[model.IncludedTag]{
		value: value,
		parser: func(s string) (model.IncludedTag, error) {
			s, err := parseTag(s)
			return model.IncludedTag(s), err
		},
		stringer: func(t *model.IncludedTag) string { return string(*t) },
		typeStr:  "tag",
	}
}

func ExcludedTagFlag(value *[]model.ExcludedTag) *ArrayValueFlag[model.ExcludedTag] {
	return &ArrayValueFlag[model.ExcludedTag]{
		value: value,
		parser: func(s string) (model.ExcludedTag, error) {
			s, err := parseTag(s)
			return model.ExcludedTag(s), err
		},
		stringer: func(t *model.ExcludedTag) string { return string(*t) },
		typeStr:  "tag",
	}
}
