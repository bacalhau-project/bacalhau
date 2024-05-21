package flags

import (
	"encoding/csv"
	"fmt"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/pflag"
	"golang.org/x/exp/slices"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/semantic"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/models"

	"github.com/bacalhau-project/bacalhau/cmd/util/output"
	legacy_job "github.com/bacalhau-project/bacalhau/pkg/legacyjob"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	publisher_ipfs "github.com/bacalhau-project/bacalhau/pkg/publisher/ipfs"
	publisher_local "github.com/bacalhau-project/bacalhau/pkg/publisher/local"
	publisher_s3 "github.com/bacalhau-project/bacalhau/pkg/s3"
	storage_ipfs "github.com/bacalhau-project/bacalhau/pkg/storage/ipfs"
	storage_local "github.com/bacalhau-project/bacalhau/pkg/storage/local_directory"
	storage_s3 "github.com/bacalhau-project/bacalhau/pkg/storage/s3"
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

func SeparatorParser(sep string) KeyValueParser[string, string] {
	return func(input string) (string, string, error) {
		slices := strings.Split(input, sep)
		if len(slices) != 2 {
			return "", "", fmt.Errorf("%q should contain exactly one %s", input, sep)
		}
		return slices[0], slices[1], nil
	}
}

func parseIPFSStorageSpec(input string) (model.StorageSpec, error) {
	cid, path, err := SeparatorParser(":")(input)
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
	u, err := storage_url.IsURLSupported(inputURL)
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

func strStringer(str *string) string {
	if str == nil {
		return ""
	}
	return *str
}

func strParser(str string) (string, error) {
	return str, nil
}

func EngineFlag(value *string) *ValueFlag[string] {
	return &ValueFlag[string]{
		value:    value,
		parser:   strParser,
		stringer: strStringer,
		typeStr:  "engine",
	}
}

func PublisherFlag(value *string) *ValueFlag[string] {
	return &ValueFlag[string]{
		value:    value,
		parser:   strParser,
		stringer: strStringer,
		typeStr:  "publisher",
	}
}

func StorageSourceFlag(value *string) *ValueFlag[string] {
	return &ValueFlag[string]{
		value:    value,
		parser:   strParser,
		stringer: strStringer,
		typeStr:  "storage-source",
	}
}

var (
	EnginesFlag        = ArrayValueFlagFrom(EngineFlag)
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

func DataLocalityFlag(value *semantic.JobSelectionDataLocality) *ValueFlag[semantic.JobSelectionDataLocality] {
	return &ValueFlag[semantic.JobSelectionDataLocality]{
		value:    value,
		parser:   semantic.ParseJobSelectionDataLocality,
		stringer: func(l *semantic.JobSelectionDataLocality) string { return l.String() },
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

func StorageTypeFlag(value *types.StorageType) *ValueFlag[types.StorageType] {
	return &ValueFlag[types.StorageType]{
		value:    value,
		parser:   types.ParseStorageType,
		stringer: func(p *types.StorageType) string { return p.String() },
		typeStr:  "storage-type",
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
	if !legacy_job.IsSafeAnnotation(s) {
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

func JobSelectionCLIFlags(policy *node.JobSelectionPolicy) *pflag.FlagSet {
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
	flags.Var(StorageSourcesFlag(&config.Storages), "disable-storage", "A storage type to disable.")

	return flags
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

func NetworkV2Flag(value *models.Network) *ValueFlag[models.Network] {
	return &ValueFlag[models.Network]{
		value:    value,
		parser:   models.ParseNetwork,
		stringer: func(n *models.Network) string { return n.String() },
		typeStr:  "network-type",
	}
}

func PublisherSpecFlag(value **models.SpecConfig) *ValueFlag[*models.SpecConfig] {
	return &ValueFlag[*models.SpecConfig]{
		value:  value,
		parser: parsePublisherSpec,
		stringer: func(i **models.SpecConfig) string {
			return (*i).Type
		},
		typeStr: "SpecConfig",
	}
}

func parsePublisherSpec(value string) (*models.SpecConfig, error) {
	csvReader := csv.NewReader(strings.NewReader(value))
	fields, err := csvReader.Read()
	if err != nil {
		return nil, err
	}

	var destinationURI string
	options := make(map[string]string)

	for i, field := range fields {
		key, val, ok := strings.Cut(field, "=")

		if !ok {
			// parsing simple format of just publisher type
			if i == 0 {
				destinationURI = field
				continue
			} else {
				return nil, fmt.Errorf("invalid publisher option: %s. Must be a key=value pair", field)
			}
		}

		key = strings.ToLower(key)
		switch key {
		case "opt", "option":
			k, v, _ := strings.Cut(val, "=")
			if k != "" {
				options[k] = v
			}
		default:
			return nil, fmt.Errorf("invalid publisher option: %s", field)
		}
	}
	v, err := publisherStringToSpecConfig(destinationURI, options)
	return v, err
}

func publisherStringToSpecConfig(destinationURI string, options map[string]string) (*models.SpecConfig, error) {
	destinationURI = strings.Trim(destinationURI, " '\"")
	parsedURI, err := url.Parse(destinationURI)
	if err != nil {
		return nil, err
	}

	// handle scenarios where the destinationURI is just the scheme/publisher type, e.g. ipfs
	if parsedURI.Scheme == "" {
		parsedURI.Scheme = parsedURI.Path
	}

	var res *models.SpecConfig
	switch parsedURI.Scheme {
	case "ipfs":
		res = publisher_ipfs.NewSpecConfig()
	case "s3":
		var bucket, key string
		var opts []publisher_s3.PublisherOption
		if _, ok := options["bucket"]; !ok {
			bucket = parsedURI.Host
		}
		if _, ok := options["key"]; !ok {
			key = strings.TrimLeft(parsedURI.Path, "/")
		}
		region, ok := options["region"]
		if ok {
			opts = append(opts, publisher_s3.WithPublisherRegion(region))
		}
		endpoint, ok := options["endpoint"]
		if ok {
			opts = append(opts, publisher_s3.WithPublisherEndpoint(endpoint))
		}
		res, err = publisher_s3.NewPublisherSpec(bucket, key, opts...)
		if err != nil {
			return nil, err
		}
	case "local":
		res = publisher_local.NewSpecConfig()
	default:
		return nil, fmt.Errorf("unknown publisher type: %s", parsedURI.Scheme)
	}

	return res, nil
}

func InputSourceFlag(value *[]*models.InputSource) *ArrayValueFlag[*models.InputSource] {
	return &ArrayValueFlag[*models.InputSource]{
		value:    value,
		parser:   parseInputSource,
		stringer: func(i **models.InputSource) string { return fmt.Sprintf("%+v", i) },
		typeStr:  "InputSource",
	}
}

func parseInputSource(value string) (*models.InputSource, error) {
	csvReader := csv.NewReader(strings.NewReader(value))
	fields, err := csvReader.Read()
	if err != nil {
		return nil, err
	}

	var sourceURI string
	destination := "/inputs" // default destination
	options := make(map[string]string)

	for i, field := range fields {
		key, val, ok := strings.Cut(field, "=")

		if !ok {
			// parsing simple format of source:destination
			if i == 0 {
				parsedURI, err := url.Parse(field)
				if err != nil {
					return nil, err
				}
				// find the last colon, excluding the schema part
				schema := parsedURI.Scheme
				trimmedURI := strings.TrimPrefix(field, schema+"://")
				index := strings.LastIndex(trimmedURI, ":")
				if index == -1 {
					sourceURI = field
				} else {
					sourceURI = schema + "://" + trimmedURI[:index]
					destination = trimmedURI[index+1:]
				}
				continue
			} else {
				return nil, fmt.Errorf("invalid storage option: %s. Must be a key=value pair", field)
			}
		}

		key = strings.ToLower(key)
		switch key {
		case "source", "src":
			sourceURI = val
		case "target", "dst", "destination":
			destination = val
		case "opt", "option":
			k, v, _ := strings.Cut(val, "=")
			if k != "" {
				options[k] = v
			}
		default:
			return nil, fmt.Errorf("unpexted key %s in field %s", key, field)
		}
	}
	// TODO there is little to no documentation on how to use alias
	// https://docs.bacalhau.org/setting-up/jobs/input-source#inputsource-parameters
	// at first glance it appears its only used in edge cases for wasm. unclear how to provide it over CLI.
	alias := ""
	storageSpec, err := storageStringToSpecConfig(sourceURI, destination, alias, options)
	if err != nil {
		return nil, err
	}
	return storageSpec, nil
}

//nolint:gocyclo
func storageStringToSpecConfig(sourceURI, destinationPath, alias string, options map[string]string) (*models.InputSource, error) {
	sourceURI = strings.Trim(sourceURI, " '\"")
	destinationPath = strings.Trim(destinationPath, " '\"")
	parsedURI, err := url.Parse(sourceURI)
	if err != nil {
		return nil, err
	}

	var sc *models.SpecConfig
	switch parsedURI.Scheme {
	case "ipfs":
		sc, err = storage_ipfs.NewSpecConfig(parsedURI.Host)
		if err != nil {
			return nil, err
		}
	case "http", "https":
		sc, err = storage_url.NewSpecConfig(sourceURI)
		if err != nil {
			return nil, err
		}
	case "s3":
		s3spec := storage_s3.SourceSpec{}
		s3spec.Bucket = parsedURI.Host
		s3spec.Key = strings.TrimLeft(parsedURI.Path, "/")
		for key, value := range options {
			switch key {
			case "endpoint":
				s3spec.Endpoint = value
			case "region":
				s3spec.Region = value
			case "versionID", "version-id", "version_id":
				s3spec.VersionID = value
			case "checksum-256", "checksum256", "checksum_256":
				s3spec.ChecksumSHA256 = value
			case "filter":
				s3spec.Filter = value
			default:
				return nil, fmt.Errorf("unknown option %q for storage %s", key, parsedURI.Scheme)
			}
			if err := s3spec.Validate(); err != nil {
				return nil, err
			}
			sc = &models.SpecConfig{
				Type:   models.StorageSourceS3,
				Params: s3spec.ToMap(),
			}
		}
	case "file":
		source := filepath.Join(parsedURI.Host, parsedURI.Path)
		var rw bool
		for key, value := range options {
			switch key {
			case "rw", "read-write", "read_write", "readwrite":
				readwrite, parseErr := strconv.ParseBool(value)
				if parseErr != nil {
					return nil, fmt.Errorf("failed to parse read-write option: %s", parseErr)
				}
				rw = readwrite
			default:
				return nil, fmt.Errorf("unknown option %s", key)
			}
		}
		sc, err = storage_local.NewSpecConfig(source, rw)
		if err != nil {
			return nil, err
		}
	case "git", "gitlfs":
		return nil, fmt.Errorf("unsupported type: %s", parsedURI.Scheme)
	default:
		return nil, fmt.Errorf("unknown storage schema: %s", parsedURI.Scheme)
	}

	return &models.InputSource{
		Source: sc,
		Alias:  alias,
		Target: destinationPath,
	}, nil
}
