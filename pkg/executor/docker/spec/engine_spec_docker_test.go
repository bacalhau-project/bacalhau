package spec

import (
	"reflect"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/model"
)

func TestAsJobSpecDocker(t *testing.T) {
	tests := []struct {
		name    string
		engine  model.EngineSpec
		want    *model.JobSpecDocker
		wantErr bool
	}{
		{
			name: "Valid EngineSpec",
			engine: model.EngineSpec{
				Type: model.DockerEngineType,
				Spec: map[string]interface{}{
					model.DockerEngineImageKey:      "test-image",
					model.DockerEngineEntrypointKey: []interface{}{"entry1", "entry2"},
					model.DockerEngineEnvVarKey:     []interface{}{"ENV1=value1", "ENV2=value2"},
					model.DockerEngineWorkDirKey:    "/app",
				},
			},
			want: &model.JobSpecDocker{
				Image:                "test-image",
				Entrypoint:           []string{"entry1", "entry2"},
				EnvironmentVariables: []string{"ENV1=value1", "ENV2=value2"},
				WorkingDirectory:     "/app",
			},
			wantErr: false,
		},
		{
			name: "Invalid EngineSpec Type",
			engine: model.EngineSpec{
				Type: 999,
				Spec: map[string]interface{}{},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Uninitialized EngineSpec",
			engine: model.EngineSpec{
				Type: model.DockerEngineType,
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := model.AsJobSpecDocker(tt.engine)
			if (err != nil) != tt.wantErr {
				t.Errorf("AsJobSpecDocker() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AsJobSpecDocker() = %v, want %v", got, tt.want)
			}
		})
	}
}
