package types

// AuthenticatorConfig is config for a specific named authentication method,
// specifying the type of authentication and the path to a policy file that
// controls the method. Some implementation types may require policies that meet
// a certain interface beyond the default – see the documentation on that type
// for more info.
type AuthenticatorConfig struct {
	Type       string `yaml:"Type,omitempty"`
	PolicyPath string `yaml:"PolicyPath,omitempty"`
}

// AuthConfig is config that controls user authentication and authorization.
type AuthConfig struct {
	// TokensPath is the location where a state file of tokens will be stored.
	// By default it will be local to the Bacalhau repo, but can be any location
	// in the host filesystem. Tokens are sensitive and should be stored in a
	// location that is only readable to the current user.
	TokensPath string `yaml:"TokensPath,omitempty"`

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
	Methods map[string]AuthenticatorConfig `yaml:"Methods,omitempty"`

	// AccessPolicyPath is the path to a file or directory that will be loaded as
	// the policy to apply to all inbound API requests. If unspecified, a policy
	// that permits access to all API endpoints to both authenticated and
	// unauthenticated users (the default as of v1.2.0) will be used.
	AccessPolicyPath string `yaml:"AccessPolicyPath,omitempty"`
}
