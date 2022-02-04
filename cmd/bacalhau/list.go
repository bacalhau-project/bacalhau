package bacalhau

import (
	"fmt"
	"log"
	"net/rpc"

	"github.com/filecoin-project/bacalhau/internal"
	"github.com/spf13/cobra"
)

var listOutputFormat string

func init() {
	submitCmd.PersistentFlags().StringVar(
		&listOutputFormat, "output", "text",
		`The output format for the list of jobs (json or text)`,
	)
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List jobs on the network",
	RunE: func(cmd *cobra.Command, cmdArgs []string) error {
		/*
				:= table.NewWriter()
				SetOutputMirror(os.Stdout)
				AppendHeader(table.Row{"#", "First Name", "Last Name", "Salary"})
				AppendRows([]table.Row{
					Arya", "Stark", 3000},
					"Jon", "Snow", 2000, "You know nothing, Jon Snow!"},

				AppendSeparator()
				AppendRow([]interface{}{300, "Tyrion", "Lannister", 5000})
				AppendFooter(table.Row{"", "", "Total", 10000})
				Render()
		**/
		//make connection to rpc server
		client, err := rpc.DialHTTP("tcp", fmt.Sprintf(":%d", jsonrpcPort))
		if err != nil {
			log.Fatalf("Error in dialing. %s", err)
		}
		args := &internal.ListArgs{}
		result := &internal.ListResponse{}
		err = client.Call("JobServer.List", args, result)
		if err != nil {
			log.Fatalf("error in JobServer: %s", err)
		}
		fmt.Printf("---> results \n%+v\n", result)
		return nil
	},
}
