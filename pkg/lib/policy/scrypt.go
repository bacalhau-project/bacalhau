package policy

import (
	"encoding/base64"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/rego"
	"github.com/open-policy-agent/opa/v1/types"
	"golang.org/x/crypto/scrypt"
)

// See https://pkg.go.dev/golang.org/x/crypto/scrypt
const (
	n      = 32768
	r      = 8
	p      = 1
	keyLen = 32
)

func Scrypt(password, salt []byte) ([]byte, error) {
	return scrypt.Key(password, salt, n, r, p, keyLen)
}

// scryptFn exposes the `scrypt` password hashing primitive to Rego.
var scryptFn = rego.Function2(
	&rego.Function{
		Name:             "scrypt",
		Description:      "Run the scrypt key derivation function",
		Decl:             types.NewFunction(types.Args(types.S, types.S), types.S),
		Memoize:          true,
		Nondeterministic: false,
	},
	func(bCtx rego.BuiltinContext, passwordTerm, saltTerm *ast.Term) (*ast.Term, error) {
		var password, salt string
		if err := ast.As(passwordTerm.Value, &password); err != nil {
			return nil, err
		}
		if err := ast.As(saltTerm.Value, &salt); err != nil {
			return nil, err
		}

		saltBytes, err := base64.StdEncoding.DecodeString(salt)
		if err != nil {
			return nil, err
		}

		passwordBytes := []byte(password)
		hash, err := Scrypt(passwordBytes, saltBytes)
		if err != nil {
			return nil, err
		}

		value, err := ast.InterfaceToValue(base64.StdEncoding.EncodeToString(hash))
		if err != nil {
			return nil, err
		}

		return ast.NewTerm(value), nil
	},
)
