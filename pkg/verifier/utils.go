package verifier

import (
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/verifier/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/verifier/noop"
)

func NewIPFSVerifiers(cm *system.CleanupManager, ipfsMultiAddress string) (
	map[string]Verifier, error) {

	noopVerifier, err := noop.NewNoopVerifier()
	if err != nil {
		return nil, err
	}

	ipfsVerifier, err := ipfs.NewIPFSVerifier(cm, ipfsMultiAddress)
	if err != nil {
		return nil, err
	}

	return map[string]Verifier{
		string(VERIFIER_NOOP): noopVerifier,
		string(VERIFIER_IPFS): ipfsVerifier,
	}, nil
}
