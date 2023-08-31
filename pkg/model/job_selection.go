package model

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// Job selection policy configuration
//
//go:generate stringer -type=JobSelectionDataLocality -linecomment
type JobSelectionDataLocality int64

func (i *JobSelectionDataLocality) UnmarshalText(text []byte) error {
	out, err := ParseJobSelectionDataLocality(string(text))
	if err != nil {
		return err
	}
	*i = out
	return nil
}

func (i JobSelectionDataLocality) MarshalYAML() (interface{}, error) {
	return i.String(), nil
}

func (i *JobSelectionDataLocality) UnmarshalYAML(value *yaml.Node) error {
	out, err := ParseJobSelectionDataLocality(value.Value)
	if err != nil {
		return err
	}
	*i = out
	return nil
}

const (
	Local    JobSelectionDataLocality = 0 // local
	Anywhere JobSelectionDataLocality = 1 // anywhere
)

func ParseJobSelectionDataLocality(s string) (ret JobSelectionDataLocality, err error) {
	for typ := Local; typ <= Anywhere; typ++ {
		if equal(typ.String(), s) {
			return typ, nil
		}
	}

	return Local, fmt.Errorf("%T: unknown type '%s'", Local, s)
}

// describe the rules for how a compute node selects an incoming job
type JobSelectionPolicy struct {
	// this describes if we should run a job based on
	// where the data is located - i.e. if the data is "local"
	// or if the data is "anywhere"
	Locality JobSelectionDataLocality `json:"locality" yaml:"Locality"`
	// should we reject jobs that don't specify any data
	// the default is "accept"
	RejectStatelessJobs bool `json:"reject_stateless_jobs" yaml:"RejectStatelessJobs"`
	// should we accept jobs that specify networking
	// the default is "reject"
	AcceptNetworkedJobs bool `json:"accept_networked_jobs" yaml:"AcceptNetworkedJobs"`
	// external hooks that decide if we should take on the job or not
	// if either of these are given they will override the data locality settings
	ProbeHTTP string `json:"probe_http,omitempty" yaml:"ProbeHTTP"`
	ProbeExec string `json:"probe_exec,omitempty" yaml:"ProbeExec"`
}

// generate a default empty job selection policy
func NewDefaultJobSelectionPolicy() JobSelectionPolicy {
	return JobSelectionPolicy{
		Locality: Anywhere,
	}
}
