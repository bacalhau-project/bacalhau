package v1beta1

import (
	"encoding/json"
	"fmt"

	"github.com/filecoin-project/bacalhau/pkg/model/v1alpha1"
)

//go:generate stringer -type=APIVersion
type APIVersion int

const (
	apiVersionUnknown APIVersion = iota // must be first
	V1alpha1
	V1beta1
	apiVersionDone // must be last
)

func APIVersionLatest() APIVersion {
	return apiVersionDone - 1
}

func ParseAPIVersion(str string) (APIVersion, error) {
	for typ := apiVersionUnknown + 1; typ < apiVersionDone; typ++ {
		if equal(typ.String(), str) {
			return typ, nil
		}
	}

	return apiVersionUnknown, fmt.Errorf(
		"unknown apiversion '%s'", str)
}

func APIVersionParseJob(versionString string, data string) (Job, error) {
	version, err := ParseAPIVersion(versionString)
	if err != nil {
		return Job{}, err
	}
	if version == V1alpha1 {
		var oldJob v1alpha1.Job
		err := json.Unmarshal([]byte(data), &oldJob)
		if err != nil {
			return Job{}, fmt.Errorf("error parsing V1alpha1 Job JSON: %s", data)
		}
		return ConvertV1alpha1Job(oldJob), nil
	} else if version == V1beta1 {
		var job Job
		err := json.Unmarshal([]byte(data), &job)
		if err != nil {
			return Job{}, fmt.Errorf("error parsing V1beta1 Job JSON: %s", data)
		}
		return job, nil
	}
	return Job{}, fmt.Errorf("unknown api version '%s'", version)
}

func APIVersionParseJobEvent(versionString string, data string) (JobEvent, error) {
	version, err := ParseAPIVersion(versionString)
	if err != nil {
		return JobEvent{}, err
	}
	if version == V1alpha1 {
		var oldEvent v1alpha1.JobEvent
		err := json.Unmarshal([]byte(data), &oldEvent)
		if err != nil {
			return JobEvent{}, fmt.Errorf("error parsing V1alpha1 JobEvent JSON: %s %s", err.Error(), data)
		}
		return ConvertV1alpha1JobEvent(oldEvent), nil
	} else if version == V1beta1 {
		var ev JobEvent
		err := json.Unmarshal([]byte(data), &ev)
		if err != nil {
			return JobEvent{}, fmt.Errorf("error parsing V1beta1 JobEvent JSON: %s %s", err.Error(), data)
		}
		return ev, nil
	}
	return JobEvent{}, fmt.Errorf("unknown api version '%s'", version)
}

func APIVersionParseJobLocalEvent(versionString string, data string) (JobLocalEvent, error) {
	version, err := ParseAPIVersion(versionString)
	if err != nil {
		return JobLocalEvent{}, err
	}
	if version == V1alpha1 {
		var oldEvent v1alpha1.JobLocalEvent
		err := json.Unmarshal([]byte(data), &oldEvent)
		if err != nil {
			return JobLocalEvent{}, fmt.Errorf("error parsing V1alpha1 JobLocalEvent JSON: %s %s", err.Error(), data)
		}
		return ConvertV1alpha1JobLocalEvent(oldEvent), nil
	} else if version == V1beta1 {
		var ev JobLocalEvent
		err := json.Unmarshal([]byte(data), &ev)
		if err != nil {
			return JobLocalEvent{}, fmt.Errorf("error parsing V1beta1 JobLocalEvent JSON: %s %s", err.Error(), data)
		}
		return ev, nil
	}
	return JobLocalEvent{}, fmt.Errorf("unknown api version '%s'", version)
}

func APIVersionParseJobState(versionString string, data string) (JobState, error) {
	if data == "" {
		return JobState{}, nil
	}
	version, err := ParseAPIVersion(versionString)
	if err != nil {
		return JobState{}, err
	}
	if version == V1alpha1 {
		var oldEvent v1alpha1.JobState
		err := json.Unmarshal([]byte(data), &oldEvent)
		if err != nil {
			return JobState{}, fmt.Errorf("error parsing V1alpha1 JobState JSON: %s", data)
		}
		return ConvertV1alpha1JobState(oldEvent), nil
	} else if version == V1beta1 {
		var ev JobState
		err := json.Unmarshal([]byte(data), &ev)
		if err != nil {
			return JobState{}, fmt.Errorf("error parsing V1beta1 Job JSON: %s", data)
		}
		return ev, nil
	}
	return JobState{}, fmt.Errorf("unknown api version '%s'", version)
}
