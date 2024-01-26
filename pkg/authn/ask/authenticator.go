package ask

import (
	"context"
	"crypto/rsa"
	"encoding/json"

	"github.com/bacalhau-project/bacalhau/pkg/authn"
	"github.com/bacalhau-project/bacalhau/pkg/lib/policy"
	"github.com/lestrrat-go/jwx/jwk"
	"github.com/pkg/errors"
	"github.com/samber/lo"
)

type policyData struct {
	SigningKey jwk.Key           `json:"signingKey"`
	NodeID     string            `json:"nodeId"`
	Ask        map[string]string `json:"ask"`
}

type requiredSchema = map[string]any

const schemaRule = "bacalhau.authn.schema"

type askAuthenticator struct {
	authnPolicy *policy.Policy
	key         jwk.Key
	nodeID      string

	validate policy.Query[policyData, string]
	schema   policy.Query[any, requiredSchema]
}

func NewAuthenticator(p *policy.Policy, key *rsa.PrivateKey, nodeID string) authn.Authenticator {
	return askAuthenticator{
		authnPolicy: p,
		key:         lo.Must(jwk.New(key)),
		nodeID:      nodeID,
		validate:    policy.AddQuery[policyData, string](p, authn.PolicyTokenRule),
		schema:      policy.AddQuery[any, requiredSchema](p, schemaRule),
	}
}

// Authenticate implements authn.Authenticator.
func (authenticator askAuthenticator) Authenticate(ctx context.Context, req []byte) (authn.Authentication, error) {
	var userInput map[string]string
	err := json.Unmarshal(req, &userInput)
	if err != nil {
		return authn.Error(errors.Wrap(err, "invalid authentication data"))
	}

	input := policyData{
		SigningKey: authenticator.key,
		NodeID:     authenticator.nodeID,
		Ask:        userInput,
	}

	token, err := authenticator.validate(ctx, input)
	if errors.Is(err, policy.ErrNoResult) {
		return authn.Failed("credentials rejected"), nil
	} else if err != nil {
		return authn.Error(err)
	}

	return authn.Authentication{Success: true, Token: token}, nil
}

func (authenticator askAuthenticator) Schema(ctx context.Context) ([]byte, error) {
	schema, err := authenticator.schema(ctx, nil)
	if err != nil {
		return nil, err
	}

	return json.Marshal(schema)
}

// IsInstalled implements authn.Authenticator.
func (authenticator askAuthenticator) IsInstalled(ctx context.Context) (bool, error) {
	schema, err := authenticator.Schema(ctx)
	return err == nil && schema != nil, err
}

// Requirement implements authn.Authenticator.
func (authenticator askAuthenticator) Requirement() authn.Requirement {
	params := lo.Must(authenticator.Schema(context.TODO()))
	return authn.Requirement{
		Type:   authn.MethodTypeAsk,
		Params: (*json.RawMessage)(&params),
	}
}
