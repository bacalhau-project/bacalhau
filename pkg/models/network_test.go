//go:build unit || !integration

package models

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type NetworkTestSuite struct {
	suite.Suite
}

func TestNetworkSuite(t *testing.T) {
	suite.Run(t, new(NetworkTestSuite))
}

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

func (s *NetworkTestSuite) TestNetworkConfigValidation() {
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
			name: "ports with none network",
			config: NetworkConfig{
				Type: NetworkNone,
				Ports: []*Port{
					{
						Name:   "http",
						Static: 8080,
					},
				},
			},
			wantErr: true,
			errMsg:  "ports can only be set for Host or Bridge network modes",
		},
		{
			name: "ports with http network",
			config: NetworkConfig{
				Type: NetworkHTTP,
				Ports: []*Port{
					{
						Name:   "http",
						Static: 8080,
					},
				},
			},
			wantErr: true,
			errMsg:  "ports can only be set for Host or Bridge network modes",
		},
		{
			name: "duplicate static ports when specified",
			config: NetworkConfig{
				Type: NetworkBridge,
				Ports: []*Port{
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
				Ports: []*Port{
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
				Ports: []*Port{
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
				Ports: []*Port{
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
				Ports: []*Port{
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
				Ports: []*Port{
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
				Ports: []*Port{
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
				Ports: []*Port{
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
				Ports: []*Port{
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
				Ports: []*Port{
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
			name: "target port in host mode",
			config: NetworkConfig{
				Type: NetworkHost,
				Ports: []*Port{
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
				Ports: []*Port{
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
				Ports: []*Port{
					{
						Name:   "http",
						Target: 80,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "port name at max length",
			config: NetworkConfig{
				Type: NetworkBridge,
				Ports: []*Port{
					{
						Name:   "p" + strings.Repeat("o", maxPortName-1),
						Static: 8080,
						Target: 80,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "port name too long",
			config: NetworkConfig{
				Type: NetworkBridge,
				Ports: []*Port{
					{
						Name:   "p" + strings.Repeat("o", maxPortName),
						Static: 8080,
						Target: 80,
					},
				},
			},
			wantErr: true,
			errMsg:  "port name too long",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			err := tt.config.Validate()
			if tt.wantErr {
				s.Error(err)
				if tt.errMsg != "" {
					s.Contains(err.Error(), tt.errMsg)
				}
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *NetworkTestSuite) TestDomainSet() {
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
		s.Run(fmt.Sprintf("%v->%v", test.input, test.output), func() {
			set := NetworkConfig{Domains: test.input}
			s.ElementsMatch(test.output, set.DomainSet())
		})
	}
}

func (s *NetworkTestSuite) TestDomainMatching() {
	tests := []struct {
		assertion string
		left      string
		right     string
		wantEqual bool
	}{
		// Equal cases
		{assertion: "equal", left: "foo.com", right: "foo.com", wantEqual: true},
		{assertion: "equal", left: ".foo.com", right: "foo.com", wantEqual: true},
		{assertion: "equal", left: "foo.com", right: ".foo.com", wantEqual: true},
		{assertion: "equal", left: " .foo.com", right: ".foo.com", wantEqual: true},
		{assertion: "equal", left: "x.foo.com", right: ".foo.com", wantEqual: true},
		{assertion: "equal", left: "y.x.foo.com", right: ".foo.com", wantEqual: true},
		{assertion: "equal", left: "FOO.com", right: "foo.COM", wantEqual: true},

		// Not equal cases
		{assertion: "notEqual", left: "x.foo.com", right: "foo.com", wantEqual: false},
		{assertion: "notEqual", left: "foo.com", right: "x.foo.com", wantEqual: false},
		{assertion: "notEqual", left: "bar.com", right: "foo.com", wantEqual: false},
		{assertion: "notEqual", left: ".bar.com", right: "foo.com", wantEqual: false},
		{assertion: "notEqual", left: ".bar.com", right: ".foo.com", wantEqual: false},
		{assertion: "notEqual", left: "bar.com", right: ".foo.com", wantEqual: false},

		// Ordering cases
		{assertion: "less", left: "zzz.com", right: "foo.com", wantEqual: false},
		{assertion: "greater", left: "aaa.com", right: "foo.com", wantEqual: false},
		{assertion: "less", left: "bFoo.com", right: "aFoo.com", wantEqual: false},
		{assertion: "greater", left: "aFoo.com", right: "bFoo.com", wantEqual: false},
		{assertion: "less", left: "x-foo.com", right: ".foo.com", wantEqual: false},
	}

	for _, tt := range tests {
		s.Run(fmt.Sprintf("%s:%s<=>%s", tt.assertion, tt.left, tt.right), func() {
			result := matchDomain(tt.left, tt.right)
			if tt.wantEqual {
				s.Zero(result)
			} else {
				s.NotZero(result)
			}
		})
	}
}

func TestPortMapping_Validate(t *testing.T) {
	tests := []struct {
		name    string
		port    Port
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid port mapping",
			port: Port{
				Name:   "HTTP",
				Static: 8080,
				Target: 80,
			},
			wantErr: false,
		},
		{
			name: "missing name",
			port: Port{
				Static: 8080,
				Target: 80,
			},
			wantErr: true,
			errMsg:  "name is required",
		},
		{
			name: "invalid name characters",
			port: Port{
				Name:   "invalid-name",
				Static: 8080,
			},
			wantErr: true,
			errMsg:  "port name must be a valid environment variable name",
		},
		{
			name: "name too long",
			port: Port{
				Name:   string(make([]byte, 257)),
				Static: 8080,
			},
			wantErr: true,
			errMsg:  "port name too long",
		},
		{
			name: "privileged static port",
			port: Port{
				Name:   "HTTP",
				Static: 80,
			},
			wantErr: true,
			errMsg:  "privileged port range",
		},
		{
			name: "static port too high",
			port: Port{
				Name:   "HTTP",
				Static: 65536,
			},
			wantErr: true,
			errMsg:  "above maximum valid port",
		},
		{
			name: "invalid target port",
			port: Port{
				Name:   "HTTP",
				Static: 8080,
				Target: 65536,
			},
			wantErr: true,
			errMsg:  "invalid target port",
		},
		{
			name: "invalid host network",
			port: Port{
				Name:        "HTTP",
				Static:      8080,
				HostNetwork: "not.an.ip",
			},
			wantErr: true,
			errMsg:  "invalid host network IP address",
		},
		{
			name: "valid host network",
			port: Port{
				Name:        "HTTP",
				Static:      8080,
				HostNetwork: "127.0.0.1",
			},
			wantErr: false,
		},
		{
			name: "name starting with number",
			port: Port{
				Name:   "1http",
				Static: 8080,
			},
			wantErr: true,
			errMsg:  "port name must be a valid environment variable name",
		},
		{
			name: "name with special characters",
			port: Port{
				Name:   "http$port",
				Static: 8080,
			},
			wantErr: true,
			errMsg:  "port name must be a valid environment variable name",
		},
		{
			name: "negative target port",
			port: Port{
				Name:   "http",
				Static: 8080,
				Target: -80,
			},
			wantErr: true,
			errMsg:  "invalid target port",
		},
		{
			name: "zero static port",
			port: Port{
				Name:   "http",
				Static: 0,
				Target: 80,
			},
			wantErr: false,
		},
		{
			name: "ipv6 host network",
			port: Port{
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
				t.Errorf("Port.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("Port.Validate() error message = %v, want to contain %v", err, tt.errMsg)
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
func (s *NetworkTestSuite) TestNetworkConfigCopy() {
	original := &NetworkConfig{
		Type:    NetworkBridge,
		Domains: []string{"example.com"},
		Ports: []*Port{
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
	s.Equal(original.Type, copied.Type)
	s.Equal(original.Domains, copied.Domains)
	s.Equal(len(original.Ports), len(copied.Ports))

	// Modify original to ensure deep copy
	original.Type = NetworkHost
	original.Domains[0] = "modified.com"
	original.Ports[0].Static = 9090

	// Verify copied version remains unchanged
	s.Equal(NetworkBridge, copied.Type)
	s.Equal("example.com", copied.Domains[0])
	s.Equal(8080, copied.Ports[0].Static)
}

// Add test for NetworkConfig.Normalize()
func (s *NetworkTestSuite) TestNetworkConfigNormalize() {
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
				Ports:   PortMap{},
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
				Ports:   PortMap{},
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.input.Normalize()
			if tt.input == nil {
				s.Nil(tt.expected)
				return
			}
			s.Equal(tt.expected.Type, tt.input.Type)
			s.Equal(tt.expected.Domains, tt.input.Domains)
			s.Equal(tt.expected.Ports, tt.input.Ports)
		})
	}
}
