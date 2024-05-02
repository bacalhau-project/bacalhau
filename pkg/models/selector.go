package models

import (
	"errors"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"

	"github.com/bacalhau-project/bacalhau/pkg/lib/validate"
)

// LabelSelectorRequirement A selector that contains values, a key, and an operator that relates the key and values.
// These are based on labels library from kubernetes package. While we use labels.Requirement to represent the label selector requirements
// in the command line arguments as the library supports multiple parsing formats, and we also use it when matching selectors to labels
// as that's what the library expects, labels.Requirements are not serializable, so we need to convert them to LabelSelectorRequirements.
type LabelSelectorRequirement struct {
	// key is the label key that the selector applies to.
	Key string `json:"Key"`
	// operator represents a key's relationship to a set of values.
	// Valid operators are In, NotIn, Exists and DoesNotExist.
	Operator selection.Operator `json:"Operator"`
	// values is an array of string values. If the operator is In or NotIn,
	// the values array must be non-empty. If the operator is Exists or DoesNotExist,
	// the values array must be empty. This array is replaced during a strategic
	Values []string `json:"Values,omitempty"`
}

func (r *LabelSelectorRequirement) String() string {
	return fmt.Sprintf("%s %s %s", r.Key, r.Operator, strings.Join(r.Values, "|"))
}

func (r *LabelSelectorRequirement) Copy() *LabelSelectorRequirement {
	if r == nil {
		return nil
	}
	return &LabelSelectorRequirement{
		Key:      r.Key,
		Operator: r.Operator,
		Values:   r.Values,
	}
}

func (r *LabelSelectorRequirement) Validate() error {
	var mErr error
	if validate.IsBlank(r.Key) {
		mErr = errors.Join(mErr, errors.New("selector key cannot be blank"))
	}
	switch r.Operator {
	case selection.In, selection.NotIn:
		if validate.IsEmpty(r.Values) {
			mErr = errors.Join(mErr, errors.New("selector values cannot be empty for In or NotIn operators"))
		}
	case selection.Exists, selection.DoesNotExist:
		if !validate.IsEmpty(r.Values) {
			mErr = errors.Join(mErr, errors.New("selector values must be empty for Exists or DoesNotExist operators"))
		}
	default:
		if len(r.Values) != 1 {
			mErr = errors.Join(mErr, errors.New("selector values must have exactly one value for other operators"))
		}
	}
	return mErr
}

func ToLabelSelectorRequirements(requirements ...labels.Requirement) []*LabelSelectorRequirement {
	labelSelectorRequirements := make([]*LabelSelectorRequirement, 0, len(requirements))
	for _, requirement := range requirements {
		labelSelectorRequirements = append(labelSelectorRequirements, &LabelSelectorRequirement{
			Key:      requirement.Key(),
			Operator: requirement.Operator(),
			Values:   requirement.Values().List(),
		})
	}
	return labelSelectorRequirements
}

func FromLabelSelectorRequirements(requirements ...*LabelSelectorRequirement) ([]labels.Requirement, error) {
	var labelSelectorRequirements []labels.Requirement
	for _, requirement := range requirements {
		req, err := labels.NewRequirement(requirement.Key, requirement.Operator, requirement.Values)
		if err != nil {
			return nil, err
		}
		labelSelectorRequirements = append(labelSelectorRequirements, *req)
	}
	return labelSelectorRequirements, nil
}
