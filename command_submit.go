package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var submitCmd = &cobra.Command{
	Use:   "submit",
	Short: "Submit a job to the network",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("hello world\n")
		os.Exit(0)
		return nil
	},
}
