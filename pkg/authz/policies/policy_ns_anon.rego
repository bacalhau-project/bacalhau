package bacalhau.authz
import rego.v1

default allow = false

job_endpoint := ["api", "v1", "orchestrator", "jobs"]

# https://developer.mozilla.org/en-US/docs/Glossary/Safe/HTTP
http_safe_methods := ["GET", "HEAD", "OPTIONS"]
http_unsafe_methods := ["PUT", "DELETE", "POST"]


is_legacy_api if {
    input.http.path[2] == "requester"
}

# Allow writing jobs if the access token has namespace write access
allow if {
    input.http.path == job_endpoint
    input.http.method in http_unsafe_methods

    namespace_writable(job_namespace_perms)
}

# Allow reading jobs if the access token has namespace read access
allow if {
    input.http.path == job_endpoint
    input.http.method in http_safe_methods

    namespace_readable(job_namespace_perms)
}

# Allow reading all other endpoints, inclduing by users who don't have a token
allow if {
    input.http.path != job_endpoint
    not is_legacy_api
    input.http.method in http_safe_methods
}

# Allow access to legacy job APIs which will do authz internally
allow if {
    is_legacy_api
    input.http.path[3] in ["submit", "cancel"]

    token_namespaces
}

# Allow reading other non-job V1 APIs without a token
allow if {
    is_legacy_api
    not input.http.path[3] in ["submit", "cancel"]
}

# Allow posting to auth endpoints, neccessary to get a token in the first place
allow if {
    input.http.path[2] == "auth"
}


# The permissions the access token grants on the job namespace
job_namespace_perms := bits.or(token_namespaces[job_namespace], token_namespaces["*"]) if {
    token_namespaces[job_namespace]
    token_namespaces["*"]
}

job_namespace_perms := token_namespaces[job_namespace] if {
    token_namespaces[job_namespace]
}

job_namespace_perms := token_namespaces["*"] if {
    token_namespaces["*"]
}

# The namespace that the submitted job is going into
default job_namespace := ""
job_namespace := ns if {
    jobRequest := yaml.unmarshal(input.http.body)
    ns := jobRequest["namespace"]
}

# The list of namespaces from the verified access token
token_namespaces := ns if {
    authHeader := input.http.headers["Authorization"][0]
    startswith(authHeader, "Bearer ")
    accessToken := trim_prefix(authHeader, "Bearer ")

    # TODO: verify signature
    [header, claims, sig] := io.jwt.decode(accessToken)
    ns := claims["ns"]
}

namespace_readable(namespace)     if { bits.and(namespace, 1) != 0 }
namespace_writable(namespace)     if { bits.and(namespace, 2) != 0 }
namespace_downloadable(namespace) if { bits.and(namespace, 4) != 0 }
namespace_cancelable(namespace)   if { bits.and(namespace, 8) != 0 }
