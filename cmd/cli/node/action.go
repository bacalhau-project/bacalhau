package node

import (
	"fmt"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/spf13/cobra"
)

type NodeActionCmd struct {
	action  string
	message string
}

func NewActionCmd(action apimodels.NodeAction) *cobra.Command {
	actionCmd := &NodeActionCmd{
		action:  string(action),
		message: "",
	}

	cmd := &cobra.Command{
		Use:   fmt.Sprintf("%s [id]", action),
		Short: action.Description(),
		Args:  cobra.ExactArgs(1),
		RunE:  actionCmd.run,
	}

	cmd.Flags().StringVarP(&actionCmd.message, "message", "m", "", "Message to include with the action")
	return cmd
}

func (n *NodeActionCmd) run(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	nodeID := args[0]

	response, err := util.GetAPIClientV2(cmd).Nodes().Put(ctx, &apimodels.PutNodeRequest{
		NodeID:  nodeID,
		Action:  n.action,
		Message: n.message,
	})
	if err != nil {
		util.Fatal(cmd, fmt.Errorf("could not %s node %s: %w", n.action, nodeID, err), 1)
	}

	if response.Success {
		cmd.Println("Ok")
	} else {
		cmd.PrintErrf("Failed to %s node %s: %s\n", n.action, nodeID, response.Error)
	}

	return nil
}
