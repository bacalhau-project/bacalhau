package bacalhau

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/job"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/storage/url/urldownload"
	"github.com/spf13/pflag"
	"golang.org/x/exp/slices"
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

func EngineFlag(value *model.Engine) *ValueFlag[model.Engine] {
	return &ValueFlag[model.Engine]{
		value:    value,
		parser:   model.ParseEngine,
		stringer: func(e *model.Engine) string { return e.String() },
		typeStr:  "engine",
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

func StorageSourceFlag(value *model.StorageSourceType) *ValueFlag[model.StorageSourceType] {
	return &ValueFlag[model.StorageSourceType]{
		value:    value,
		parser:   model.ParseStorageSourceType,
		stringer: func(s *model.StorageSourceType) string { return s.String() },
		typeStr:  "storage-source",
	}
}

var (
	EnginesFlag        = ArrayValueFlagFrom(EngineFlag)
	VerifiersFlag      = ArrayValueFlagFrom(VerifierFlag)
	PublishersFlag     = ArrayValueFlagFrom(PublisherFlag)
	StorageSourcesFlag = ArrayValueFlagFrom(StorageSourceFlag)
)

func NetworkFlag(value *model.Network) *ValueFlag[model.Network] {
	return &ValueFlag[model.Network]{
		value:    value,
		parser:   model.ParseNetwork,
		stringer: func(n *model.Network) string { return n.String() },
		typeStr:  "network-type",
	}
}

func TargetingFlag(value *model.TargetingMode) *ValueFlag[model.TargetingMode] {
	return &ValueFlag[model.TargetingMode]{
		value:    value,
		parser:   model.ParseTargetingMode,
		stringer: func(tm *model.TargetingMode) string { return tm.String() },
		typeStr:  "all|any",
	}
}

func DataLocalityFlag(value *model.JobSelectionDataLocality) *ValueFlag[model.JobSelectionDataLocality] {
	return &ValueFlag[model.JobSelectionDataLocality]{
		value:    value,
		parser:   model.ParseJobSelectionDataLocality,
		stringer: func(l *model.JobSelectionDataLocality) string { return l.String() },
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

func EnvVarMapFlag(value *map[string]string) *MapValueFlag[string, string] {
	return &MapValueFlag[string, string]{
		value:    value,
		parser:   separatorParser("="),
		stringer: func(k *string, v *string) string { return fmt.Sprintf("%s=%s", *k, *v) },
		typeStr:  "key=value",
	}
}

func URLFlag(value **url.URL, schemes ...string) *ValueFlag[*url.URL] {
	return &ValueFlag[*url.URL]{
		value: value,
		parser: func(s string) (u *url.URL, err error) {
			u, err = url.Parse(s)
			if u != nil && !slices.Contains(schemes, u.Scheme) {
				err = fmt.Errorf("URL scheme must be one of: %v", schemes)
			}
			return
		},
		stringer: func(u **url.URL) string {
			if u == nil || (*u) == nil {
				return ""
			} else {
				return (*u).String()
			}
		},
		typeStr: "url",
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

func JobSelectionCLIFlags(policy *model.JobSelectionPolicy) *pflag.FlagSet {
	flags := pflag.NewFlagSet("Job Selection Policy", pflag.ContinueOnError)

	flags.Var(
		DataLocalityFlag(&policy.Locality), "job-selection-data-locality",
		`Only accept jobs that reference data we have locally ("local") or anywhere ("anywhere").`,
	)
	flags.BoolVar(
		&policy.RejectStatelessJobs, "job-selection-reject-stateless", policy.RejectStatelessJobs,
		`Reject jobs that don't specify any data.`,
	)
	flags.BoolVar(
		&policy.AcceptNetworkedJobs, "job-selection-accept-networked", policy.AcceptNetworkedJobs,
		`Accept jobs that require network access.`,
	)
	flags.StringVar(
		&policy.ProbeHTTP, "job-selection-probe-http", policy.ProbeHTTP,
		`Use the result of a HTTP POST to decide if we should take on the job.`,
	)
	flags.StringVar(
		&policy.ProbeExec, "job-selection-probe-exec", policy.ProbeExec,
		`Use the result of a exec an external program to decide if we should take on the job.`,
	)

	return flags
}

func DisabledFeatureCLIFlags(config *node.FeatureConfig) *pflag.FlagSet {
	flags := pflag.NewFlagSet("Disabled Features", pflag.ContinueOnError)

	flags.Var(EnginesFlag(&config.Engines), "disable-engine", "An engine type to disable.")
	flags.Var(PublishersFlag(&config.Publishers), "disable-publisher", "A publisher type to disable.")
	flags.Var(VerifiersFlag(&config.Verifiers), "disable-verifier", "A verifier to disable.")
	flags.Var(StorageSourcesFlag(&config.Storages), "disable-storage", "A storage type to disable.")

	return flags
}

func DealCLIFlags(deal *model.Deal) *pflag.FlagSet {
	flags := pflag.NewFlagSet("Deal", pflag.ContinueOnError)

	flags.Var(TargetingFlag(&deal.TargetingMode), "target",
		`Whether to target the minimum number of matching nodes ("any") (default) or all matching nodes ("all")`)
	flags.IntVarP(&deal.Concurrency, "concurrency", "c", deal.Concurrency,
		`How many nodes should run the job when using --target=any`)
	flags.IntVar(&deal.Confidence, "confidence", deal.Confidence,
		`The minimum number of nodes that must agree on a verification result`)
	flags.IntVar(&deal.MinBids, "min-bids", deal.MinBids,
		`Minimum number of bids that must be received before concurrency-many bids will be accepted (at random)`)

	return flags
}
