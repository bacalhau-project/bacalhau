package verifier

type VerifierType string

const VERIFIER_IPFS VerifierType = "ipfs"
const VERIFIER_NOOP VerifierType = "noop"

var VERIFIERS = []string{
	string(VERIFIER_IPFS),
	string(VERIFIER_NOOP),
}
