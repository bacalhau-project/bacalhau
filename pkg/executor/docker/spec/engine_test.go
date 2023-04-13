package spec

import (
	"reflect"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/model"
)

func TestAsDockerSpec(t *testing.T) {
	tests := []struct {
		name    string
		engine  model.EngineSpec
		want    *JobSpecDocker
		wantErr bool
	}{
		{
			name: "valid docker spec",
			engine: model.EngineSpec{
				Type: model.EngineDocker,
				Params: map[string]interface{}{
					DockerEngineImageKey:      "example/image",
					DockerEngineEntrypointKey: []string{"entry1", "entry2"},
					DockerEngineWorkDirKey:    "/app",
				},
			},
			want: &JobSpecDocker{
				Image:            "example/image",
				Entrypoint:       []string{"entry1", "entry2"},
				WorkingDirectory: "/app",
			},
			wantErr: false,
		},
		{
			name: "invalid engine type",
			engine: model.EngineSpec{
				Type:   model.EngineWasm,
				Params: map[string]interface{}{},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "uninitialized params",
			engine: model.EngineSpec{
				Type: model.EngineDocker,
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := AsDockerSpec(tt.engine)
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
