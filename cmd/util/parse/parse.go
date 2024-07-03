package parse

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/labels"

	"github.com/bacalhau-project/bacalhau/cmd/util/flags"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

func NodeSelector(nodeSelector string) ([]*models.LabelSelectorRequirement, error) {
	selector := strings.TrimSpace(nodeSelector)
	if len(selector) == 0 {
		return nil, nil
	}
	requirements, err := labels.ParseToRequirements(selector)
	if err != nil {
		return nil, fmt.Errorf("failed to parse node selector: %w", err)
	}
	tmp := models.ToLabelSelectorRequirements(requirements...)
	out := make([]*models.LabelSelectorRequirement, 0, len(tmp))
	for _, r := range tmp {
		out = append(out, r.Copy())
	}
	return out, nil
}

func StringSliceToMap(slice []string) (map[string]string, error) {
	result := make(map[string]string)
	for _, item := range slice {
		key, value, err := flags.SeparatorParser("=")(item)
		if err != nil {
			return nil, fmt.Errorf("expected 'key=value', received invalid format for key-value pair: %s", item)
		}
		result[key] = value
	}
	return result, nil
}
