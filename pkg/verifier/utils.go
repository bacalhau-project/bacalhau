package verifier

import (
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/verifier/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/verifier/noop"
)

func NewIPFSVerifiers(
	cancelContext *system.CancelContext,
	ipfsMultiAddress string,
) (map[string]Verifier, error) {
	verifiers := map[string]Verifier{}

	noopVerifier, err := noop.NewNoopVerifier()
	if err != nil {
		return verifiers, err
	}

	verifiers[string(VERIFIER_NOOP)] = noopVerifier

	ipfsVerifier, err := ipfs.NewIPFSVerifier(cancelContext, ipfsMultiAddress)
	if err != nil {
		return verifiers, err
	}

	verifiers[string(VERIFIER_IPFS)] = ipfsVerifier

	return verifiers, nil
}
