//go:build unit || !integration

package envvar

import (
	"reflect"
	"sort"
	"testing"
)

func TestToSlice(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want []string
	}{
		{
			name: "empty map",
			env:  map[string]string{},
			want: nil,
		},
		{
			name: "single key-value pair",
			env:  map[string]string{"KEY": "VALUE"},
			want: []string{"KEY=VALUE"},
		},
		{
			name: "multiple key-value pairs",
			env: map[string]string{
				"KEY1": "VALUE1",
				"KEY2": "VALUE2",
			},
			want: []string{"KEY1=VALUE1", "KEY2=VALUE2"},
		},
		{
			name: "values with special characters",
			env: map[string]string{
				"PATH":   "/usr/local/bin:/usr/bin",
				"SPACES": "value with spaces",
			},
			want: []string{
				"PATH=/usr/local/bin:/usr/bin",
				"SPACES=value with spaces",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToSlice(tt.env)
			// Sort both slices since map iteration order is random
			if len(got) > 0 && len(tt.want) > 0 {
				sortStrings(got)
				sortStrings(tt.want)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ToSlice() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFromSlice(t *testing.T) {
	tests := []struct {
		name string
		env  []string
		want map[string]string
	}{
		{
			name: "empty slice",
			env:  []string{},
			want: map[string]string{},
		},
		{
			name: "single key-value pair",
			env:  []string{"KEY=VALUE"},
			want: map[string]string{"KEY": "VALUE"},
		},
		{
			name: "multiple key-value pairs",
			env:  []string{"KEY1=VALUE1", "KEY2=VALUE2"},
			want: map[string]string{
				"KEY1": "VALUE1",
				"KEY2": "VALUE2",
			},
		},
		{
			name: "invalid format",
			env:  []string{"INVALID", "KEY=VALUE"},
			want: map[string]string{"KEY": "VALUE"},
		},
		{
			name: "empty value",
			env:  []string{"KEY="},
			want: map[string]string{"KEY": ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FromSlice(tt.env); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FromSlice() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSanitize(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{
			name:  "no changes needed",
			value: "normal_value",
			want:  "normal_value",
		},
		{
			name:  "spaces to underscore",
			value: "value with spaces",
			want:  "value_with_spaces",
		},
		{
			name:  "equals to underscore",
			value: "key=value",
			want:  "key_value",
		},
		{
			name:  "multiple spaces and equals",
			value: "key = value = something",
			want:  "key___value___something",
		},
		{
			name:  "empty string",
			value: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Sanitize(tt.value); got != tt.want {
				t.Errorf("Sanitize() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMerge(t *testing.T) {
	tests := []struct {
		name     string
		base     map[string]string
		priority map[string]string
		want     map[string]string
	}{
		{
			name:     "empty maps",
			base:     map[string]string{},
			priority: map[string]string{},
			want:     map[string]string{},
		},
		{
			name: "priority overrides base",
			base: map[string]string{
				"KEY1": "BASE1",
				"KEY2": "BASE2",
			},
			priority: map[string]string{
				"KEY1": "PRIORITY1",
			},
			want: map[string]string{
				"KEY1": "PRIORITY1",
				"KEY2": "BASE2",
			},
		},
		{
			name: "non-overlapping keys",
			base: map[string]string{
				"BASE_KEY": "BASE_VALUE",
			},
			priority: map[string]string{
				"PRIORITY_KEY": "PRIORITY_VALUE",
			},
			want: map[string]string{
				"BASE_KEY":     "BASE_VALUE",
				"PRIORITY_KEY": "PRIORITY_VALUE",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Merge(tt.base, tt.priority); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Merge() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMergeSlices(t *testing.T) {
	tests := []struct {
		name     string
		base     []string
		priority []string
		want     []string
	}{
		{
			name:     "empty slices",
			base:     []string{},
			priority: []string{},
			want:     nil,
		},
		{
			name:     "priority overrides base",
			base:     []string{"KEY1=BASE1", "KEY2=BASE2"},
			priority: []string{"KEY1=PRIORITY1"},
			want:     []string{"KEY1=PRIORITY1", "KEY2=BASE2"},
		},
		{
			name:     "non-overlapping keys",
			base:     []string{"BASE_KEY=BASE_VALUE"},
			priority: []string{"PRIORITY_KEY=PRIORITY_VALUE"},
			want:     []string{"BASE_KEY=BASE_VALUE", "PRIORITY_KEY=PRIORITY_VALUE"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MergeSlices(tt.base, tt.priority)
			// Sort slices since map iteration order is random
			if len(got) > 0 && len(tt.want) > 0 {
				sortStrings(got)
				sortStrings(tt.want)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MergeSlices() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper function to sort string slices for comparison
func sortStrings(s []string) {
	sort.Strings(s)
}
