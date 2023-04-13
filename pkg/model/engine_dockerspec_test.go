package model

import (
	"reflect"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/executor/docker/spec"
)

func TestAsDockerSpec(t *testing.T) {
	tests := []struct {
		name    string
		engine  EngineSpec
		want    *spec.JobSpecDocker
		wantErr bool
	}{
		{
			name: "valid docker spec",
			engine: EngineSpec{
				Type: EngineDocker,
				Params: map[string]interface{}{
					"Image":                "example/image",
					"Entrypoint":           []string{"entry1", "entry2"},
					"EnvironmentVariables": []string{"VAR1=value1", "VAR2=value2"},
					"WorkingDirectory":     "/app",
				},
			},
			want: &spec.JobSpecDocker{
				Image:                "example/image",
				Entrypoint:           []string{"entry1", "entry2"},
				EnvironmentVariables: []string{"VAR1=value1", "VAR2=value2"},
				WorkingDirectory:     "/app",
			},
			wantErr: false,
		},
		{
			name: "invalid engine type",
			engine: EngineSpec{
				Type:   EngineWasm,
				Params: map[string]interface{}{},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "uninitialized params",
			engine: EngineSpec{
				Type: EngineDocker,
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.engine.AsDockerSpec()
			if (err != nil) != tt.wantErr {
				t.Errorf("AsDockerSpec() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AsDockerSpec() = %v, want %v", got, tt.want)
			}
		})
	}
}
