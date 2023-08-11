package cmd

import (
	"os"
	"strings"

	"github.com/multiformats/go-multiaddr"
	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/pkg/system"
)

func getCommandLineExecutable() string {
	return os.Args[0]
}

func FatalErrorHandler(cmd *cobra.Command, msg string, code int) {
	if len(msg) > 0 {
		// add newline if needed
		if !strings.HasSuffix(msg, "\n") {
			msg += "\n"
		}
		cmd.Print(msg)
	}
	os.Exit(code)
}

func getPeers(peerConnect string) ([]multiaddr.Multiaddr, error) {
	var peersStrings []string
	if peerConnect == "none" {
		peersStrings = []string{}
	} else if peerConnect == "" {
		peersStrings = system.Envs[system.EnvironmentProd].BootstrapAddresses
	} else {
		peersStrings = strings.Split(peerConnect, ",")
	}

	peers := make([]multiaddr.Multiaddr, 0, len(peersStrings))
	for _, peer := range peersStrings {
		parsed, err := multiaddr.NewMultiaddr(peer)
		if err != nil {
			return nil, err
		}
		peers = append(peers, parsed)
	}
	return peers, nil
}
