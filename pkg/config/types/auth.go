package types

// AuthConfig is config that controls user authentication and authorization.
type AuthConfig struct {
	// Methods maps "method names" to authenticator implementations. A method
	// name is a human-readable string chosen by the person configuring the
	// system that is shown to users to help them pick the authentication method
	// they want to use. There can be multiple usages of the same Authenticator
	// *type* but with different configs and parameters, each identified with a
	// unique method name.
	//
	// For example, if an implementation wants to allow users to log in with
	// Github or Bitbucket, they might both use an authenticator implementation
	// of type "oidc", and each would appear once on this provider with key /
	// method name "github" and "bitbucket".
	//
	// By default, only a single authentication method that accepts
	// authentication via client keys will be enabled.
	Methods map[string]AuthenticatorConfig `yaml:"Methods,omitempty" json:"Methods,omitempty"`

	// AccessPolicyPath is the path to a file or directory that will be loaded as
	// the policy to apply to all inbound API requests. If unspecified, a policy
	// that permits access to all API endpoints to both authenticated and
	// unauthenticated users (the default as of v1.2.0) will be used.
	AccessPolicyPath string `yaml:"AccessPolicyPath,omitempty" json:"AccessPolicyPath,omitempty"`

	Users  []AuthUser   `yaml:"Users,omitempty" json:"Users,omitempty"`
	Oauth2 Oauth2Config `yaml:"Oauth2,omitempty" json:"Oauth2,omitempty"`
}

// AuthenticatorConfig is config for a specific named authentication method,
// specifying the type of authentication and the path to a policy file that
// controls the method. Some implementation types may require policies that meet
// a certain interface beyond the default – see the documentation on that type
// for more info.
type AuthenticatorConfig struct {
	Type       string `yaml:"Type,omitempty" json:"Type,omitempty"`
	PolicyPath string `yaml:"PolicyPath,omitempty" json:"PolicyPath,omitempty"`
}

type AuthUser struct {
	Alias        string       `yaml:"Alias,omitempty" json:"Alias,omitempty"`
	Username     string       `yaml:"Username,omitempty" json:"Username,omitempty"`
	Password     string       `yaml:"Password,omitempty" json:"Password,omitempty"`
	APIKey       string       `yaml:"APIKey,omitempty" json:"APIKey,omitempty"`
	Capabilities []Capability `yaml:"Capabilities,omitempty" json:"Capabilities,omitempty"`
}

type Capability struct {
	Actions []string `yaml:"Actions,omitempty" json:"Actions,omitempty"`
}

type Oauth2Config struct {
	ProviderID                  string   `yaml:"ProviderID,omitempty" json:"ProviderID,omitempty"`
	ProviderName                string   `yaml:"ProviderName,omitempty" json:"ProviderName,omitempty"`
	TokenEndpoint               string   `yaml:"TokenEndpoint,omitempty" json:"TokenEndpoint,omitempty"`
	JWKSUri                     string   `yaml:"JWKSUri,omitempty" json:"JWKSUri,omitempty"`
	Issuer                      string   `yaml:"Issuer,omitempty" json:"Issuer,omitempty"`
	DeviceClientID              string   `yaml:"DeviceClientID,omitempty" json:"DeviceClientID,omitempty"`
	Audience                    string   `yaml:"Audience,omitempty" json:"Audience,omitempty"`
	Scopes                      []string `yaml:"Scopes,omitempty" json:"Scopes,omitempty"`
	DeviceAuthorizationEndpoint string   `yaml:"DeviceAuthorizationEndpoint,omitempty" json:"DeviceAuthorizationEndpoint,omitempty"`
	PollingInterval             int      `yaml:"PollingInterval,omitempty" json:"PollingInterval,omitempty"`
}
