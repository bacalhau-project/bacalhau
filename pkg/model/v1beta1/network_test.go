package v1beta1

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
			if err := n.IsValid(); tt.wantErr {
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
		{require.Less, "bfoo.com", "afoo.com"},
		{require.Greater, "afoo.com", "bfoo.com"},
		{require.Less, "x-foo.com", ".foo.com"},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%s<=>%s", test.left, test.right), func(t *testing.T) {
			test.require(t, 0, matchDomain(test.left, test.right))
		})
	}
}
