package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/choose"
	"github.com/bacalhau-project/bacalhau/pkg/authn"
	"github.com/bacalhau-project/bacalhau/pkg/authn/challenge"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"golang.org/x/exp/maps"
)

type responder = func(request *json.RawMessage) (response []byte, err error)

func RunAuthenticationFlow(cmd *cobra.Command) (string, error) {
	supportedMethods := map[authn.MethodType]responder{
		authn.MethodTypeChallenge: challenge.Respond,
		authn.MethodTypeAsk:       askResponder(cmd),
	}

	client := util.GetAPIClientV2(cmd.Context())
	methods, err := client.Auth().Methods(&apimodels.ListAuthnMethodsRequest{})
	if err != nil {
		return "", err
	}

	filteredMethods := make(map[string]authn.Requirement, len(methods.Methods))
	clientTypes := maps.Keys(supportedMethods)
	for name, req := range methods.Methods {
		if lo.Contains(clientTypes, req.Type) {
			filteredMethods[name] = req
		}
	}

	if len(filteredMethods) == 0 {
		serverTypes := lo.Map(maps.Values(methods.Methods), func(r authn.Requirement, _ int) authn.MethodType { return r.Type })
		return "", fmt.Errorf("no common authentication method: client supports %v, server supports %v", clientTypes, serverTypes)
	}

	var authentication authn.Authentication
	for !authentication.Success {
		supportedNames := maps.Keys(filteredMethods)
		chosenMethodName, err := choose.Choose(cmd, "How would you like to authenticate?", supportedNames)
		if errors.Is(err, io.EOF) {
			return "", nil
		} else if err != nil {
			return "", err
		}

		methodRequirement := methods.Methods[chosenMethodName]
		methodResponder := supportedMethods[methodRequirement.Type]
		response, err := methodResponder(methodRequirement.Params)
		if err != nil {
			return "", err
		}

		authnResponse, err := client.Auth().Authenticate(&apimodels.AuthnRequest{
			Name:       chosenMethodName,
			MethodData: response,
		})
		if err != nil {
			return "", err
		}

		authentication = authnResponse.Authentication
		if authentication.Reason != "" {
			cmd.PrintErrln(authentication.Reason)
		}
	}

	return authentication.Token, nil
}

// A Cobra pre-run hook that will run the authentication flow.
func Authenticate(cmd *cobra.Command, args []string) error {
	base := config.ClientAPIBase()

	// See if we have a token for the server we will be using.
	token, err := util.ReadToken(base)
	if err != nil {
		return err
	}
	if token != "" {
		return nil
	}

	// No token found – so eagerly run an authentication flow to try and get a
	// valid token.
	token, err = RunAuthenticationFlow(cmd)
	if err != nil {
		return err
	}
	if token != "" {
		return util.WriteToken(base, token)
	}

	// Failed to authenticate. That's ok – this server may accept
	// unauthenticated requests.
	return nil
}
