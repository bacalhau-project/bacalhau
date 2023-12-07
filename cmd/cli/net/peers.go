package net

import (
	"fmt"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/output"
	"github.com/bacalhau-project/bacalhau/pkg/lib"
)

type peerListEntry struct {
	peer peer.AddrInfo
}

var listColumns = []output.TableColumn[peerListEntry]{
	{
		ColumnConfig: table.ColumnConfig{Name: "ID"},
		Value: func(s peerListEntry) string {
			return s.peer.ID.String()
		},
	},
	{
		ColumnConfig: table.ColumnConfig{Name: "Addresses"},
		Value: func(v peerListEntry) string {
			sb := strings.Builder{}
			for _, a := range v.peer.Addrs {
				sb.WriteString(a.String())
				sb.WriteString("\n")
			}
			return sb.String()
		},
	},
}

func NewPeersCmd() *cobra.Command {
	peersCmd := &cobra.Command{
		Use:   "peers",
		Short: "Get the peers connected to the host.",
		RunE: func(cmd *cobra.Command, args []string) error {
			response, err := util.GetAPIClientV2(cmd.Context()).Net().Peers()
			if err != nil {
				return err
			}
			var list []peerListEntry
			for _, p := range response.Peers {
				list = append(list, peerListEntry{peer: p})
			}
			o := output.OutputOptions{
				Format:     output.TableFormat,
				Pretty:     true,
				HideHeader: false,
				NoStyle:    false,
				Wide:       false,
			}
			if err := output.Output(cmd, listColumns, o, list); err != nil {
				return err
			}
			return nil
		},
	}
	return peersCmd
}

func NewConnectCmd() *cobra.Command {
	connectCmd := &cobra.Command{
		Use:   "connect",
		Short: "Connect to peers.",
		RunE: func(cmd *cobra.Command, args []string) error {
			pis, err := lib.ParseAddresses(cmd.Context(), args)
			if err != nil {
				return err
			}

			for _, p := range pis {
				response, err := util.GetAPIClientV2(cmd.Context()).Net().Connect(p)
				if err != nil {
					return err
				}
				cmd.Printf("Success: %t", response.Success)
			}
			return nil
		},
	}
	return connectCmd
}

func NewDisconnectCmd() *cobra.Command {
	disconnectCmd := &cobra.Command{
		Use:   "disconnect",
		Short: "Disconnect from peers.",
		RunE: func(cmd *cobra.Command, args []string) error {
			peers := make([]peer.ID, 0, len(args))
			for _, a := range args {
				p, err := peer.Decode(a)
				if err != nil {
					return fmt.Errorf("decoding peerID from string %q: %w", a, err)
				}
				peers = append(peers, p)
			}

			for _, p := range peers {
				response, err := util.GetAPIClientV2(cmd.Context()).Net().Disconnect(p)
				if err != nil {
					return err
				}
				cmd.Printf("Success: %t", response.Success)
			}
			return nil
		},
	}
	return disconnectCmd
}

func NewPingCmd() *cobra.Command {
	pingCmd := &cobra.Command{
		Use:   "ping",
		Short: "Ping peers.",
		RunE: func(cmd *cobra.Command, args []string) error {
			pis, err := lib.ParseAddresses(cmd.Context(), args)
			if err != nil {
				return err
			}

			for _, p := range pis {
				response, err := util.GetAPIClientV2(cmd.Context()).Net().Ping(p.ID)
				if err != nil {
					return err
				}
				if response.Msg != "" {
					cmd.Printf("PING %s ERROR=%s\n", p.ID, response.Msg)
				} else {
					cmd.Printf("PING %s TTL=%s", p.ID, response.TTL)
				}
			}
			return nil
		},
	}
	return pingCmd
}
