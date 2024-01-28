package bacalhau.authn

import rego.v1

schema := {
	"type": "object",
	"properties": {"magic": {"type": "string"}},
	"required": ["magic"],
}

token := t if {
	input.magic == "open sesame"

	t := io.jwt.encode_sign(
		{
			"typ": "JWT",
			"alg": "RS256",
		},
		{
			"iss": input.nodeId,
			"sub": "aladdin",
			"aud": [input.nodeId],
			"iat": now,
			"exp": one_month,
			"ns": {
				# Read-only access to all namespaces
				"*": read_only,
				# Writable access to own namespace
				"genie": full_access,
			},
		},
		input.signingKey,
	)
}
