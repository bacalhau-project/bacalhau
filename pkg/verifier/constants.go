package verifier

type verifierType string

const VERIFIER_IPFS verifierType = "ipfs"
const VERIFIER_NOOP verifierType = "noop"

var VERIFIERS = []string{
	string(VERIFIER_IPFS),
	string(VERIFIER_NOOP),
}
