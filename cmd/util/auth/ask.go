package auth

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

// Returns a responder that responds to authentication requirements of type
// `authn.MethodTypeAsk`. Reads the JSON Schema returned by the `ask` endpoint
// and uses it to ask appropriate questions to the user on their terminal, and
// then returns their response as serialized JSON.
func askResponder(cmd *cobra.Command) responder {
	return func(request *json.RawMessage) ([]byte, error) {
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

			fmt.Fprintf(cmd.ErrOrStderr(), "%s: ", name)

			var input []byte
			var err error

			// If the property is marked as write only, assume it is a sensitive
			// value and make sure we don't display it in the terminal
			if subschema.WriteOnly {
				input, err = term.ReadPassword(int(os.Stdin.Fd()))
				fmt.Fprintln(cmd.ErrOrStderr())
			} else {
				reader := bufio.NewScanner(cmd.InOrStdin())
				if reader.Scan() {
					input = reader.Bytes()
				}
				err = reader.Err()
			}

			if err != nil {
				return nil, err
			}
			response[name] = string(input)
		}

		respBytes, err := json.Marshal(response)
		if err != nil {
			return nil, err
		}

		return respBytes, schema.Validate(response)
	}
}
