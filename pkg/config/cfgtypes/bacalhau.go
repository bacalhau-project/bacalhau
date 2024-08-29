package cfgtypes

import (
	"reflect"
	"strings"
)

// NB: Developers, after making changes to this struct or any of its children, run go generate.

//go:generate go run gen/generate.go ./
type Bacalhau struct {
	API                 API                    `yaml:"API,omitempty"`
	NameProvider        string                 `yaml:"NameProvider,omitempty"`
	DataDir             string                 `yaml:"DataDir,omitempty"`
	StrictVersionMatch  bool                   `yaml:"StrictVersionMatch,omitempty"`
	Orchestrator        Orchestrator           `yaml:"Orchestrator,omitempty"`
	Compute             Compute                `yaml:"Compute,omitempty"`
	WebUI               WebUI                  `yaml:"WebUI,omitempty"`
	InputSources        InputSourcesConfig     `yaml:"InputSources,omitempty"`
	Publishers          PublishersConfig       `yaml:"Publishers,omitempty"`
	Engines             EngineConfig           `yaml:"Engines,omitempty"`
	ResultDownloaders   ResultDownloaders      `yaml:"ResultDownloaders,omitempty"`
	JobDefaults         JobDefaults            `yaml:"JobDefaults,omitempty"`
	JobAdmissionControl JobAdmissionControl    `yaml:"JobAdmissionControl,omitempty"`
	Logging             Logging                `yaml:"Logging,omitempty"`
	UpdateConfig        UpdateConfig           `yaml:"UpdateConfig,omitempty"`
	FeatureFlags        FeatureFlags           `yaml:"FeatureFlags,omitempty"`
	DefaultPublisher    DefaultPublisherConfig `yaml:"DefaultPublisher,omitempty"`
}

func AllKeys() map[string]reflect.Type {
	config := Bacalhau{}
	paths := make(map[string]reflect.Type)
	buildPathMap(reflect.ValueOf(config), "", paths)
	return paths
}

func buildPathMap(v reflect.Value, prefix string, paths map[string]reflect.Type) {
	switch v.Kind() {
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			field := v.Type().Field(i)
			tag := field.Tag.Get("yaml")
			if tag == "" {
				tag = field.Name
			} else {
				tag = strings.Split(tag, ",")[0]
			}
			fieldPath := prefix + strings.ToLower(tag)
			buildPathMap(v.Field(i), fieldPath+".", paths)
		}
	case reflect.Map, reflect.Slice, reflect.Array, reflect.Ptr:
		paths[prefix[:len(prefix)-1]] = v.Type()
	default:
		paths[prefix[:len(prefix)-1]] = v.Type()
	}
}

type API struct {
	Host string     `yaml:"Host,omitempty"`
	Port int        `yaml:"Port,omitempty"`
	TLS  TLS        `yaml:"TLS,omitempty"`
	Auth AuthConfig `yaml:"Auth,omitempty"`
}

type TLS struct {
	CertFile string `yaml:"CertFile,omitempty"`
	KeyFile  string `yaml:"KeyFile,omitempty"`
	CAFile   string `yaml:"CAFile,omitempty"`

	// client only
	UseTLS   bool `yaml:"UseTLS,omitempty"`
	Insecure bool `yaml:"Insecure"`

	// orchestrator only fields
	SelfSigned        bool   `yaml:"SelfSigned,omitempty"`
	AutoCert          string `yaml:"AutoCert,omitempty"`
	AutoCertCachePath string `yaml:"AutoCertCachePath,omitempty"`
}

type WebUI struct {
	Enabled bool   `yaml:"Enabled,omitempty"`
	Listen  string `yaml:"Listen,omitempty"`
}

type Logging struct {
	Level                string   `yaml:"Level,omitempty"`
	Mode                 string   `yaml:"Mode,omitempty"`
	LogDebugInfoInterval Duration `yaml:"LogDebugInfoInterval,omitempty"`
}

type FeatureFlags struct {
	ExecTranslation bool `yaml:"ExecTranslation,omitempty"`
}

type UpdateConfig struct {
	Interval Duration `yaml:"Interval,omitempty"`
}

type JobAdmissionControl struct {
	RejectStatelessJobs bool   `yaml:"RejectStatelessJobs,omitempty"`
	AcceptNetworkedJobs bool   `yaml:"AcceptNetworkedJobs,omitempty"`
	ProbeHTTP           string `yaml:"ProbeHTTP,omitempty"`
	ProbeExec           string `yaml:"ProbeExec,omitempty"`
}

type TaskTimeoutConfig struct {
	TotalTimeout     Duration `yaml:"TotalTimeout,omitempty"`
	ExecutionTimeout Duration `yaml:"ExecutionTimeout,omitempty"`
}
