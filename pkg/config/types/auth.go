package types

import "github.com/bacalhau-project/bacalhau/pkg/authn"

// AuthenticationConfig is config for a specific named authentication method,
// specifying the type of authentication and the path to a policy file that
// controls the method. Some implementation types may require policies that meet
// a certain interface beyond the default – see the documentation on that type
// for more info.
type AuthenticatorConfig struct {
	Type       authn.MethodType `yaml:"Type"`
	PolicyPath string           `yaml:"PolicyPath,omitempty"`
}

// AuthConfig is config that controls the authentication and authorization
// process for servers. It is not used for clients.
type AuthConfig struct {
	TokensPath string                         `yaml:"TokensPath"`
	Methods    map[string]AuthenticatorConfig `yaml:"Methods"`
}
