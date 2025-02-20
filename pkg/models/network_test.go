//go:build unit || !integration

package models

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"
)

type NetworkTestSuite struct {
	suite.Suite
}

func TestNetworkSuite(t *testing.T) {
	suite.Run(t, new(NetworkTestSuite))
}

func (s *NetworkTestSuite) TestNetworkConfigDomainValidation() {
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
		s.Run(tt.name, func() {
			n := NetworkConfig{
				Type:    NetworkHTTP,
				Domains: tt.domains,
			}
			err := n.Validate()
			if tt.wantErr {
				s.Error(err)
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *NetworkTestSuite) TestPortValidation() {
	tests := []struct {
		name    string
		port    Port
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid port",
			port: Port{
				Name:   "http",
				Static: 8080,
			},
			wantErr: false,
		},
		{
			name: "missing name",
			port: Port{
				Static: 8080,
			},
			wantErr: true,
			errMsg:  "port mapping name is required",
		},
		// ... other test cases ...
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			err := tt.port.Validate()
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
		{
			name:    "wildcard domains",
			domains: []string{"y.foo.com", ".foo.com", "x.foo.com"},
			want:    []string{".foo.com"},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			nc := &NetworkConfig{
				Type:    NetworkHTTP,
				Domains: tt.domains,
			}
			got := nc.DomainSet()
			s.ElementsMatch(got, tt.want)
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

func (s *NetworkTestSuite) TestNetworkConfigCopy() {
	original := &NetworkConfig{
		Type:    NetworkBridge,
		Domains: []string{"example.com"},
		Ports: PortMap{
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
