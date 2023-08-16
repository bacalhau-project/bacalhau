package config

import (
	"github.com/multiformats/go-multiaddr"
	"github.com/spf13/viper"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

func GetLibp2pConfig() (types.Libp2pConfig, error) {
	var libp2pCfg types.Libp2pConfig
	if err := ForKey(types.NodeLibp2p, &libp2pCfg); err != nil {
		return types.Libp2pConfig{}, err
	}
	return libp2pCfg, nil
}

func GetBootstrapPeers() ([]multiaddr.Multiaddr, error) {
	bootstrappers := viper.GetStringSlice(types.NodeBootstrapAddresses)
	peers := make([]multiaddr.Multiaddr, 0, len(bootstrappers))
	for _, peer := range bootstrappers {
		parsed, err := multiaddr.NewMultiaddr(peer)
		if err != nil {
			return nil, err
		}
		peers = append(peers, parsed)
	}
	return peers, nil

}
