//go:build unit || !integration

package models

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			if err := n.Validate(); tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDomainSet(t *testing.T) {
	tests := []struct {
		input, output []string
	}{
		{
			[]string{"foo.com", "bar.com"},
			[]string{"foo.com", "bar.com"},
		},
		{
			[]string{"y.foo.com", ".foo.com", "x.foo.com"},
			[]string{".foo.com"},
		},
		{
			[]string{"y.foo.com", "foo.com", "x.foo.com"},
			[]string{"y.foo.com", "foo.com", "x.foo.com"},
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%v->%v", test.input, test.output), func(t *testing.T) {
			set := NetworkConfig{Domains: test.input}
			require.ElementsMatch(t, test.output, set.DomainSet())
		})
	}
}

func TestDomainMatching(t *testing.T) {
	tests := []struct {
		require     func(require.TestingT, interface{}, interface{}, ...interface{})
		left, right string
	}{
		{require.Equal, "foo.com", "foo.com"},
		{require.Equal, ".foo.com", "foo.com"},
		{require.Equal, "foo.com", ".foo.com"},
		{require.Equal, " .foo.com", ".foo.com"},
		{require.Equal, "x.foo.com", ".foo.com"},
		{require.Equal, "y.x.foo.com", ".foo.com"},
		{require.NotEqual, "x.foo.com", "foo.com"},
		{require.NotEqual, "foo.com", "x.foo.com"},
		{require.NotEqual, "bar.com", "foo.com"},
		{require.NotEqual, ".bar.com", "foo.com"},
		{require.NotEqual, ".bar.com", ".foo.com"},
		{require.NotEqual, "bar.com", ".foo.com"},
		{require.Less, "zzz.com", "foo.com"},
		{require.Greater, "aaa.com", "foo.com"},
		{require.Equal, "FOO.com", "foo.COM"},
		{require.Less, "bFoo.com", "aFoo.com"},
		{require.Greater, "aFoo.com", "bFoo.com"},
		{require.Less, "x-foo.com", ".foo.com"},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%s<=>%s", test.left, test.right), func(t *testing.T) {
			test.require(t, 0, matchDomain(test.left, test.right))
		})
	}
}

func TestNetworkConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  NetworkConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid none network",
			config: NetworkConfig{
				Type: NetworkNone,
			},
			wantErr: false,
		},
		{
			name: "valid host network",
			config: NetworkConfig{
				Type: NetworkHost,
			},
			wantErr: false,
		},
		{
			name: "valid bridge network",
			config: NetworkConfig{
				Type: NetworkBridge,
			},
			wantErr: false,
		},
		{
			name: "valid http network with domains",
			config: NetworkConfig{
				Type:    NetworkHTTP,
				Domains: []string{"example.com", "test.com"},
			},
			wantErr: false,
		},
		{
			name: "invalid network type",
			config: NetworkConfig{
				Type: Network(999),
			},
			wantErr: true,
			errMsg:  "invalid networking type",
		},
		{
			name: "domains with non-http network",
			config: NetworkConfig{
				Type:    NetworkBridge,
				Domains: []string{"example.com"},
			},
			wantErr: true,
			errMsg:  "domains can only be set for HTTP network mode",
		},
		{
			name: "valid IP as domain",
			config: NetworkConfig{
				Type:    NetworkHTTP,
				Domains: []string{"192.168.1.1"},
			},
			wantErr: false,
		},
		{
			name: "port mappings with none network",
			config: NetworkConfig{
				Type: NetworkNone,
				Ports: []*PortMapping{
					{
						Name:   "http",
						Static: 8080,
					},
				},
			},
			wantErr: true,
			errMsg:  "port mappings can only be set for Host or Bridge network modes",
		},
		{
			name: "port mappings with http network",
			config: NetworkConfig{
				Type: NetworkHTTP,
				Ports: []*PortMapping{
					{
						Name:   "http",
						Static: 8080,
					},
				},
			},
			wantErr: true,
			errMsg:  "port mappings can only be set for Host or Bridge network modes",
		},
		{
			name: "duplicate static ports when specified",
			config: NetworkConfig{
				Type: NetworkBridge,
				Ports: []*PortMapping{
					{
						Name:   "http1",
						Static: 8080,
						Target: 80,
					},
					{
						Name:   "http2",
						Static: 8080,
						Target: 81,
					},
				},
			},
			wantErr: true,
			errMsg:  "duplicate port mapping static port",
		},
		{
			name: "no duplicate check for unspecified static ports",
			config: NetworkConfig{
				Type: NetworkBridge,
				Ports: []*PortMapping{
					{
						Name:   "http1",
						Target: 80,
					},
					{
						Name:   "http2",
						Target: 81,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "duplicate target ports in bridge mode",
			config: NetworkConfig{
				Type: NetworkBridge,
				Ports: []*PortMapping{
					{
						Name:   "http1",
						Static: 8080,
						Target: 80,
					},
					{
						Name:   "http2",
						Static: 8081,
						Target: 80,
					},
				},
			},
			wantErr: true,
			errMsg:  "duplicate port mapping target port",
		},
		{
			name: "no duplicate check for target ports in host mode",
			config: NetworkConfig{
				Type: NetworkHost,
				Ports: []*PortMapping{
					{
						Name:   "http1",
						Static: 8080,
					},
					{
						Name:   "http2",
						Static: 8081,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "duplicate port names",
			config: NetworkConfig{
				Type: NetworkBridge,
				Ports: []*PortMapping{
					{
						Name:   "http",
						Static: 8080,
						Target: 80,
					},
					{
						Name:   "http",
						Static: 8081,
						Target: 81,
					},
				},
			},
			wantErr: true,
			errMsg:  "duplicate port mapping name",
		},
		{
			name: "auto-allocated ports in bridge mode",
			config: NetworkConfig{
				Type: NetworkBridge,
				Ports: []*PortMapping{
					{
						Name:   "http",
						Target: 80,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid host network binding",
			config: NetworkConfig{
				Type: NetworkBridge,
				Ports: []*PortMapping{
					{
						Name:        "http",
						Static:      8080,
						Target:      80,
						HostNetwork: "127.0.0.1",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing port name",
			config: NetworkConfig{
				Type: NetworkBridge,
				Ports: []*PortMapping{
					{
						Static: 8080,
						Target: 80,
					},
				},
			},
			wantErr: true,
			errMsg:  "name is required",
		},
		{
			name: "empty port name",
			config: NetworkConfig{
				Type: NetworkBridge,
				Ports: []*PortMapping{
					{
						Name:   "",
						Static: 8080,
						Target: 80,
					},
				},
			},
			wantErr: true,
			errMsg:  "name is required",
		},
		{
			name: "invalid port name characters",
			config: NetworkConfig{
				Type: NetworkBridge,
				Ports: []*PortMapping{
					{
						Name:   "invalid-name",
						Static: 8080,
						Target: 80,
					},
				},
			},
			wantErr: true,
			errMsg:  "port name must be a valid environment variable name",
		},
		{
			name: "port name too long",
			config: NetworkConfig{
				Type: NetworkBridge,
				Ports: []*PortMapping{
					{
						Name:   string(make([]byte, 257)),
						Static: 8080,
						Target: 80,
					},
				},
			},
			wantErr: true,
			errMsg:  "port name too long",
		},
		{
			name: "target port in host mode",
			config: NetworkConfig{
				Type: NetworkHost,
				Ports: []*PortMapping{
					{
						Name:   "http",
						Static: 8080,
						Target: 80,
					},
				},
			},
			wantErr: true,
			errMsg:  "target ports cannot be set for Host network mode",
		},
		{
			name: "auto allocated port in host mode",
			config: NetworkConfig{
				Type: NetworkHost,
				Ports: []*PortMapping{
					{
						Name: "http",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "auto allocated port in bridge mode",
			config: NetworkConfig{
				Type: NetworkBridge,
				Ports: []*PortMapping{
					{
						Name:   "http",
						Target: 80,
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("NetworkConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("NetworkConfig.Validate() error message = %v, want to contain %v", err, tt.errMsg)
				}
			}
		})
	}
}

func TestPortMapping_Validate(t *testing.T) {
	tests := []struct {
		name    string
		port    PortMapping
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid port mapping",
			port: PortMapping{
				Name:   "HTTP",
				Static: 8080,
				Target: 80,
			},
			wantErr: false,
		},
		{
			name: "missing name",
			port: PortMapping{
				Static: 8080,
				Target: 80,
			},
			wantErr: true,
			errMsg:  "name is required",
		},
		{
			name: "invalid name characters",
			port: PortMapping{
				Name:   "invalid-name",
				Static: 8080,
			},
			wantErr: true,
			errMsg:  "port name must be a valid environment variable name",
		},
		{
			name: "name too long",
			port: PortMapping{
				Name:   string(make([]byte, 257)),
				Static: 8080,
			},
			wantErr: true,
			errMsg:  "port name too long",
		},
		{
			name: "privileged static port",
			port: PortMapping{
				Name:   "HTTP",
				Static: 80,
			},
			wantErr: true,
			errMsg:  "privileged port range",
		},
		{
			name: "static port too high",
			port: PortMapping{
				Name:   "HTTP",
				Static: 65536,
			},
			wantErr: true,
			errMsg:  "above maximum valid port",
		},
		{
			name: "invalid target port",
			port: PortMapping{
				Name:   "HTTP",
				Static: 8080,
				Target: 65536,
			},
			wantErr: true,
			errMsg:  "invalid target port",
		},
		{
			name: "invalid host network",
			port: PortMapping{
				Name:        "HTTP",
				Static:      8080,
				HostNetwork: "not.an.ip",
			},
			wantErr: true,
			errMsg:  "invalid host network IP address",
		},
		{
			name: "valid host network",
			port: PortMapping{
				Name:        "HTTP",
				Static:      8080,
				HostNetwork: "127.0.0.1",
			},
			wantErr: false,
		},
		{
			name: "name starting with number",
			port: PortMapping{
				Name:   "1http",
				Static: 8080,
			},
			wantErr: true,
			errMsg:  "port name must be a valid environment variable name",
		},
		{
			name: "name with special characters",
			port: PortMapping{
				Name:   "http$port",
				Static: 8080,
			},
			wantErr: true,
			errMsg:  "port name must be a valid environment variable name",
		},
		{
			name: "negative target port",
			port: PortMapping{
				Name:   "http",
				Static: 8080,
				Target: -80,
			},
			wantErr: true,
			errMsg:  "invalid target port",
		},
		{
			name: "zero static port",
			port: PortMapping{
				Name:   "http",
				Static: 0,
				Target: 80,
			},
			wantErr: false,
		},
		{
			name: "ipv6 host network",
			port: PortMapping{
				Name:        "http",
				Static:      8080,
				HostNetwork: "::1",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.port.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("PortMapping.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("PortMapping.Validate() error message = %v, want to contain %v", err, tt.errMsg)
				}
			}
		})
	}
}

func TestNetworkConfig_DomainSet(t *testing.T) {
	tests := []struct {
		name    string
		domains []string
		want    []string
	}{
		{
			name:    "empty domains",
			domains: []string{},
			want:    []string{},
		},
		{
			name:    "single domain",
			domains: []string{"example.com"},
			want:    []string{"example.com"},
		},
		{
			name:    "different domains",
			domains: []string{"example.com", "test.com"},
			want:    []string{"example.com", "test.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nc := &NetworkConfig{
				Type:    NetworkHTTP,
				Domains: tt.domains,
			}
			got := nc.DomainSet()
			if !equalStringSlices(got, tt.want) {
				t.Errorf("NetworkConfig.DomainSet() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper functions
func contains(s, substr string) bool {
	return s != "" && substr != "" && s != substr && s[len(s)-1] != '.' && substr[len(substr)-1] != '.' && s[:len(s)-1] != substr[:len(substr)-1]
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Add test for NetworkConfig.Copy()
func TestNetworkConfig_Copy(t *testing.T) {
	original := &NetworkConfig{
		Type:    NetworkBridge,
		Domains: []string{"example.com"},
		Ports: []*PortMapping{
			{
				Name:        "http",
				Static:      8080,
				Target:      80,
				HostNetwork: "127.0.0.1",
			},
		},
	}

	copied := original.Copy()

	// Test that it's a deep copy
	assert.Equal(t, original.Type, copied.Type)
	assert.Equal(t, original.Domains, copied.Domains)
	assert.Equal(t, len(original.Ports), len(copied.Ports))

	// Modify original to ensure deep copy
	original.Type = NetworkHost
	original.Domains[0] = "modified.com"
	original.Ports[0].Static = 9090

	// Verify copied version remains unchanged
	assert.Equal(t, NetworkBridge, copied.Type)
	assert.Equal(t, "example.com", copied.Domains[0])
	assert.Equal(t, 8080, copied.Ports[0].Static)
}

// Add test for NetworkConfig.Normalize()
func TestNetworkConfig_Normalize(t *testing.T) {
	tests := []struct {
		name     string
		input    *NetworkConfig
		expected *NetworkConfig
	}{
		{
			name:     "nil config",
			input:    nil,
			expected: nil,
		},
		{
			name: "empty slices",
			input: &NetworkConfig{
				Type: NetworkBridge,
			},
			expected: &NetworkConfig{
				Type:    NetworkBridge,
				Domains: []string{},
				Ports:   []*PortMapping{},
			},
		},
		{
			name: "normalize domains",
			input: &NetworkConfig{
				Type:    NetworkHTTP,
				Domains: []string{" EXAMPLE.com ", "TEST.com "},
			},
			expected: &NetworkConfig{
				Type:    NetworkHTTP,
				Domains: []string{"example.com", "test.com"},
				Ports:   []*PortMapping{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.input.Normalize()
			if tt.input == nil {
				assert.Nil(t, tt.expected)
				return
			}
			assert.Equal(t, tt.expected.Type, tt.input.Type)
			assert.Equal(t, tt.expected.Domains, tt.input.Domains)
			assert.Equal(t, tt.expected.Ports, tt.input.Ports)
		})
	}
}
