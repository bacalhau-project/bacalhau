package bacalhau.authz
import rego.v1

default allow = false

# Anyone is able to query the auth endpoint to see what is suppoerted.
# This is the only anon-supported endpoint.
allow if {
    # TODO(forrest) [fixme] maybe make this more explicit.
    input.http.path[2] == "auth"
}

# You are only allowed access if you have a token. Token == root
allow if {
    token_namespaces
}

# The list of namespaces from the verified access token
token_namespaces := ns if {
    authHeader := input.http.headers["Authorization"][0]
    startswith(authHeader, "Bearer ")
    accessToken := trim_prefix(authHeader, "Bearer ")

    # TODO(simon): [fixme] verify signature
    [header, claims, sig] := io.jwt.decode(accessToken)
    ns := claims["ns"]
}