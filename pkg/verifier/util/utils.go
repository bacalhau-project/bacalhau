package util

import (
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	"github.com/filecoin-project/bacalhau/pkg/verifier/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/verifier/noop"
)

func NewIPFSVerifiers(cm *system.CleanupManager, ipfsMultiAddress string) (map[verifier.VerifierType]verifier.Verifier, error) {
	noopVerifier, err := noop.NewVerifier()
	if err != nil {
		return nil, err
	}

	ipfsVerifier, err := ipfs.NewVerifier(cm, ipfsMultiAddress)
	if err != nil {
		return nil, err
	}

	return map[verifier.VerifierType]verifier.Verifier{
		verifier.VerifierNoop: noopVerifier,
		verifier.VerifierIpfs: ipfsVerifier,
	}, nil
}
