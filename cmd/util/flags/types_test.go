//go:build unit || !integration

package flags

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

func TestParsePublisherSpec(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    *models.SpecConfig
		wantErr bool
	}{
		{
			name:  "simple ipfs publisher",
			value: "ipfs",
			want: &models.SpecConfig{
				Type:   models.PublisherIPFS,
				Params: make(map[string]interface{}),
			},
			wantErr: false,
		},
		{
			name:  "local publisher",
			value: "local",
			want: &models.SpecConfig{
				Type:   models.PublisherLocal,
				Params: make(map[string]interface{}),
			},
			wantErr: false,
		},
		{
			name:  "s3 with bucket and key",
			value: "s3://mybucket/mykey",
			want: &models.SpecConfig{
				Type: models.PublisherS3,
				Params: map[string]interface{}{
					"Bucket":   "mybucket",
					"Key":      "mykey",
					"Region":   "",
					"Endpoint": "",
				},
			},
			wantErr: false,
		},
		{
			name:  "s3 with opt",
			value: "s3://mybucket/mykey,opt=region=us-west-2,opt=endpoint=https://s3.custom.com",
			want: &models.SpecConfig{
				Type: models.PublisherS3,
				Params: map[string]interface{}{
					"Bucket":   "mybucket",
					"Key":      "mykey",
					"Region":   "us-west-2",
					"Endpoint": "https://s3.custom.com",
				},
			},
			wantErr: false,
		},
		{
			name:  "s3 with option",
			value: "s3://mybucket/mykey,option=region=us-west-2,option=endpoint=https://s3.custom.com",
			want: &models.SpecConfig{
				Type: models.PublisherS3,
				Params: map[string]interface{}{
					"Bucket":   "mybucket",
					"Key":      "mykey",
					"Region":   "us-west-2",
					"Endpoint": "https://s3.custom.com",
				},
			},
			wantErr: false,
		},
		{
			name:    "invalid option format",
			value:   "s3://mybucket/mykey,invalidoption",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "unknown publisher type",
			value:   "unknown://mybucket/mykey",
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parsePublisherSpec(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("parsePublisherSpec() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseInputSource(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    *models.InputSource
		wantErr bool
	}{
		{
			name:  "simple ipfs source",
			value: "ipfs://Qm12345:/data",
			want: &models.InputSource{
				Source: &models.SpecConfig{
					Type: models.StorageSourceIPFS,
					Params: map[string]interface{}{
						"CID": "Qm12345",
					},
				},
				Target: "/data",
				Alias:  "",
			},
			wantErr: false,
		},
		{
			name:  "http source with destination",
			value: "http://example.com/data:/inputs/data",
			want: &models.InputSource{
				Source: &models.SpecConfig{
					Type: models.StorageSourceURL,
					Params: map[string]interface{}{
						"URL": "http://example.com/data",
					},
				},
				Target: "/inputs/data",
				Alias:  "",
			},
			wantErr: false,
		},
		{
			name:  "s3 source with options",
			value: "s3://mybucket/mykey,opt=region=us-west-2,opt=endpoint=https://s3.custom.com",
			want: &models.InputSource{
				Source: &models.SpecConfig{
					Type: models.StorageSourceS3,
					Params: map[string]interface{}{
						"Bucket":         "mybucket",
						"Key":            "mykey",
						"Region":         "us-west-2",
						"Endpoint":       "https://s3.custom.com",
						"ChecksumSHA256": "",
						"Filter":         "",
						"VersionID":      "",
					},
				},
				Target: "/inputs",
				Alias:  "",
			},
			wantErr: false,
		},
		{
			name:  "s3 source with mixed options",
			value: "s3://mybucket/mykey,option=region=us-west-2,opt=endpoint=https://s3.custom.com,option=checksum256=1234,opt=filter=abc*,opt=versionID=098",
			want: &models.InputSource{
				Source: &models.SpecConfig{
					Type: models.StorageSourceS3,
					Params: map[string]interface{}{
						"Bucket":         "mybucket",
						"Key":            "mykey",
						"Region":         "us-west-2",
						"Endpoint":       "https://s3.custom.com",
						"ChecksumSHA256": "1234",
						"Filter":         "abc*",
						"VersionID":      "098",
					},
				},
				Target: "/inputs",
				Alias:  "",
			},
			wantErr: false,
		},
		{
			name:  "file source with read-write option",
			value: "file:///path/to/file,opt=rw=true",
			want: &models.InputSource{
				Source: &models.SpecConfig{
					Type: models.StorageSourceLocalDirectory,
					Params: map[string]interface{}{
						"SourcePath": "/path/to/file",
						"ReadWrite":  true,
					},
				},
				Target: "/inputs",
				Alias:  "",
			},
			wantErr: false,
		},
		{
			name:    "invalid format",
			value:   "s3://mybucket/mykey,invalidoption",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "unknown storage schema",
			value:   "unknown://mybucket/mykey",
			want:    nil,
			wantErr: true,
		},
		{
			name:  "source without destination",
			value: "ipfs://Qm12345",
			want: &models.InputSource{
				Source: &models.SpecConfig{
					Type: models.StorageSourceIPFS,
					Params: map[string]interface{}{
						"CID": "Qm12345",
					},
				},
				Target: "/inputs",
				Alias:  "",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseInputSource(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseInputSource() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
