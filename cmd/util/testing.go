package util

import (
	"github.com/bacalhau-project/bacalhau/pkg/lib/marshaller"
	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/pkg/model"
)

// FakeFatalErrorHandler captures the error for testing, responsibility of the test to handle the exit (if any)
// NOTE: If your test is not idempotent, you can cause side effects
// (the underlying function will continue to run)
// Returned as text JSON to wherever RootCmd is printing.
func FakeFatalErrorHandler(cmd *cobra.Command, msg error, code int) {
	c := model.TestFatalErrorHandlerContents{Message: msg.Error(), Code: code}
	b, _ := marshaller.JSONMarshalWithMax(c)
	cmd.Println(string(b))
}
