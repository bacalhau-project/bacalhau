package model

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNetworkConfig_IsValid(t *testing.T) {
	tests := []struct {
		name    string
		domains []string
		wantErr bool
	}{
		{
			name:    "ip4-is-valid",
			domains: []string{"192.168.0.1"},
			wantErr: false,
		},
		{
			name:    "ip6-is-valid",
			domains: []string{"0000:0000:0000:0000:0000:0000:0000:0001"},
			wantErr: false,
		},
		{
			name:    "a-domain",
			domains: []string{"example.com"},
			wantErr: false,
		},
		{
			name:    "domain-with-dot-at-start-is-okay",
			domains: []string{".example.com"},
			wantErr: false,
		},
		{
			name:    "not-a-domain",
			domains: []string{"at@.walker"},
			wantErr: true,
		},
		{
			name:    "don't-support-cidr",
			domains: []string{"192.168.0.1/32"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := NetworkConfig{
				Type:    NetworkHTTP,
				Domains: tt.domains,
			}
			if err := n.IsValid(); tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
