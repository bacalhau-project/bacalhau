package types

type ProbeExec struct {
	Command []string          `json:"command,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

type ProbeHttpScheme string

const (
	Http  ProbeHttpScheme = "http"
	Https                 = "https"
)

type ProbeHttp struct {
	Path        string            `json:"path,omitempty"`
	Port        int               `json:"port,omitempty"`
	Host        string            `json:"host,omitempty"`
	Scheme      ProbeHttpScheme   `json:"scheme,omitempty"`
	HTTPHeaders map[string]string `json:"httpHeaders,omitempty"`
}

// configuration for k8s style "probes"
// these can be used to reach out to external scripts
// and/or servers - for example to ask
// "should we run this job"
type Probe struct {
	Exec     *ProbeExec `json:"exec,omitempty"`
	HTTPPost *ProbeHttp `json:"httpPost,omitempty"`
}
