package ask

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

type Responder struct {
	Cmd *cobra.Command
}

// Returns a responder that responds to authentication requirements of type
// `authn.MethodTypeAsk`. Reads the JSON Schema returned by the `ask` endpoint
// and uses it to ask appropriate questions to the user on their terminal, and
// then returns their response as serialized JSON.
func (r *Responder) Respond(request *json.RawMessage) ([]byte, error) {
	compiler := jsonschema.NewCompiler()
	compiler.ExtractAnnotations = true

	if err := compiler.AddResource("", bytes.NewReader(*request)); err != nil {
		return nil, err
	}

	schema, err := compiler.Compile("")
	if err != nil {
		return nil, err
	}

	response := make(map[string]any, len(schema.Properties))
	for _, name := range schema.Required {
		subschema := schema.Properties[name]

		if len(subschema.Types) < 1 {
			return nil, fmt.Errorf("invalid schema: property %q has no type", name)
		}

		typ := subschema.Types[0]
		if typ == "object" {
			return nil, fmt.Errorf("invalid schema: property %q has non-scalar type", name)
		}

		fmt.Fprintf(r.Cmd.ErrOrStderr(), "%s: ", name)

		inputChan := make(chan []byte, 1)
		errChan := make(chan error, 1)

		// Reading a value is not a cancellable operation. So we manually
		// listen for the context here and write a final byte to the input
		// if the user cancels, to hopefully get the input read to end.
		go func(inputCh chan<- []byte, errCh chan<- error) {
			var input []byte
			var err error

			// If the property is marked as write only, assume it is a sensitive
			// value and make sure we don't display it in the terminal
			if subschema.WriteOnly && term.IsTerminal(int(os.Stdin.Fd())) {
				input, err = term.ReadPassword(int(os.Stdin.Fd()))
			} else {
				reader := bufio.NewScanner(r.Cmd.InOrStdin())
				if reader.Scan() {
					input = reader.Bytes()
				}
				err = reader.Err()
			}

			if err != nil {
				errCh <- err
			} else {
				inputCh <- input
			}
		}(inputChan, errChan)

		select {
		case input := <-inputChan:
			response[name] = string(input)
		case err = <-errChan:
			return nil, err
		case <-r.Cmd.Context().Done():
			_, _ = os.Stdin.Write([]byte{'\n'})
			return nil, r.Cmd.Context().Err()
		}
	}

	respBytes, err := json.Marshal(response)
	if err != nil {
		return nil, err
	}

	return respBytes, schema.Validate(response)
}
