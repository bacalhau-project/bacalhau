package license

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testJWKS = json.RawMessage(`{
	"keys": [
		{
		  "alg": "RS256",
		  "e": "AQAB",
		  "kid": "key_1",
		  "kty": "RSA",
		  "n": "nKu2xtki1b0qIKboDsT-DvFUFzzzqVFFbuSBUBPp3bYm6v1Y-KM4eSJxp8Cg1V5gpwGtpTNQkvS_wa_098LrsGxEmcSGCns0n32WtOCHAMof_lUQndR_r4AF9utCJrBEJZ3lRqXUOgvLZMasB3G4m9YXjpkZpiJFnmiK8xkbYKj4acPE0zGDb2OW1RRgDIfUwqy3JTbm2eEDpXnNGxo8GDNlGcipweXZfalWo8tGBUiTnVCWTRRt6VL3Uf8-kwg2_MjsicJRH_KfxMGVUkVqZEMdvKfSktwlcqLucy5acMNjHLD3_P7dOBPR3pFnEFJzSNPNb4d1iKw839_ZAMK6sQ",
		  "use": "sig"
		},
		{
		  "alg": "RS256",
		  "e": "AQAB",
		  "kid": "key_2",
		  "kty": "RSA",
		  "n": "qhXY1QjpKi6lFiK8ueznIRm1APiPLtOplOkAbRlma5KtFNFONUdGI7Ua2Q0UbS-akKKKtcqbKP3eDP41ytgFnG_FvlvuhQLZ9YaY1_nKarcDxxYx8VcRsFtDcfx3ekH-ipujuRcetlPto7oSkPu3ZYWRHHX0MQQ5wiy-wn1maL09uUeVPJoUH148WsUodWjWxqlIVjXU63u_126oD8C7PTedny1tZrmX-_5EnqP-zVrVhFC2AUsnVsxcqXun5ljYiYTfi62E0uH1XIKqpUn-ZhY1XHXWTm8DS6AJsgPsW7Gup5fvL3lAfMhm-yFMBL3PT1c5OwGMjNYqg31D4VIGRw",
		  "use": "sig"
		}
	]
}`)

// Test tokens
const (
	validRSASignedJWTToken1 = "eyJhbGciOiJSUzI1NiIsImtpZCI6ImtleV8xIiwidHlwIjoiSldUIn0.eyJjdXN0b21lcl9pZCI6ImN1c3RvbWVyXzEiLCJleHAiOjEwMzc2MjQwMTA3LCJmZWF0dXJlcyI6WyJmZWF0dXJlXzEiLCJmZWF0dXJlXzExIl0sImlhdCI6MTczNjI0MzcwNywibGljZW5zZV9pZCI6ImxpY2Vuc2VfMSIsImxpY2Vuc2VfdHlwZSI6InByb2RfdGllcl8xIiwibGljZW5zZV92ZXJzaW9uIjoidl8xIiwibGltaXRhdGlvbnMiOnsidGllcl8xIjoibm8ifSwibWV0YWRhdGEiOnsic29tZXRoaW5nMSI6InNvbWV0aGluZzFfdmFsdWUifSwicHJvZHVjdCI6IkJhY2FsaGF1In0.XyHdItfNma4zkwwwB_M_xgHGqRhTtNmsdPx491msaalfEKAKDYqCMsE6DhL6cKWRqKsXGx27kaBCun1chiYf_yz1rSfMZny-XdakqIg_ENburNFrNSePn-kGhUPmQLzK9JV4Iph2hTWB6dJ8rFqYewDiJ6yfX_AVymmst4OziPmBiPeDcEtjjSR8MEQynRiKUup76fKVgsgXvT-eUHURXOWBcADEw-UvbyKgEt7FB-baZSryReJTyStpA7E64OFB4fNwfk3h70WxVeTrNIBvvg94PHCoZ5MxVfiUq-G3BJc_ltDpRpruv4x_eECyUN8yAkZ2SVfIYkPP0eLyFSRXJA"
	validRSASignedJWTToken2 = "eyJhbGciOiJSUzI1NiIsImtpZCI6ImtleV8yIiwidHlwIjoiSldUIn0.eyJjdXN0b21lcl9pZCI6ImN1c3RvbWVyXzIiLCJleHAiOjEwMzc2MjQwMTA3LCJmZWF0dXJlcyI6WyJmZWF0dXJlXzIiLCJmZWF0dXJlXzIyIl0sImlhdCI6MTczNjI0MzcwNywibGljZW5zZV9pZCI6ImxpY2Vuc2VfMiIsImxpY2Vuc2VfdHlwZSI6InByb2RfdGllcl8yIiwibGljZW5zZV92ZXJzaW9uIjoidl8yIiwibGltaXRhdGlvbnMiOnsidGllcl8yIjoibm8ifSwibWV0YWRhdGEiOnsic29tZXRoaW5nMiI6InNvbWV0aGluZzJfdmFsdWUifSwicHJvZHVjdCI6IkJhY2FsaGF1In0.RlcHy_Vo7YK5Xk-NF-9B2JAMsQVQ1oiyhmCffbuBUHB68UVRtDcjF7noTsYLAagSaVypBds-qE9u0gyTpfVugN-8XaJ7AS7ebYy1tFh8z8hjNhk3HpTQroc4jlBfjT__23zyqIW5p3nbwXwd9eFm5k3pQKEqu0Xfg9Cj266JOo7knvb749PZNxyk6tFn7QstnG3eQuEzyB_S_52PoV4brWaHTcctUjyG3_LRFzfl_FRxpJd4SvcRwsHT9fqP5FvAYmtWNwV_zLS9J9_231vA-Vu9Ss8cZ6xQoem6xGCdxml6oLIRpPgn7_ZlA7PMIzlJDiYdjWZBBj-wYaBZF5P67A"
)

func TestNewLicenseValidatorFromJSON(t *testing.T) {
	tests := []struct {
		name      string
		jwks      json.RawMessage
		wantErr   bool
		errString string
	}{
		{
			name:    "Valid JWKS",
			jwks:    testJWKS,
			wantErr: false,
		},
		{
			name:      "Invalid JSON",
			jwks:      json.RawMessage(`{invalid json}`),
			wantErr:   true,
			errString: "invalid JWKS JSON",
		},
		{
			name:      "Empty JSON string",
			jwks:      json.RawMessage(`{}`),
			wantErr:   true,
			errString: "missing 'keys' array in JWKS",
		},
		{
			name:      "Empty JSON raw message",
			jwks:      json.RawMessage(``),
			wantErr:   true,
			errString: "empty JWKS JSON",
		},
		{
			name:      "Empty keys array",
			jwks:      json.RawMessage(`{"keys": []}`),
			wantErr:   true,
			errString: "empty 'keys' array in JWKS",
		},
		{
			name:      "Null keys array",
			jwks:      json.RawMessage(`{"keys": null}`),
			wantErr:   true,
			errString: "missing 'keys' array in JWKS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator, err := NewLicenseValidatorFromJSON(tt.jwks)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, validator)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, validator)
			}
		})
	}
}

func TestValidateToken(t *testing.T) {
	validator, err := NewLicenseValidatorFromJSON(testJWKS)
	require.NoError(t, err)

	tests := []struct {
		name      string
		token     string
		wantErr   bool
		errString string
	}{
		{
			name:    "Valid RSA token",
			token:   validRSASignedJWTToken1,
			wantErr: false,
		},
		{
			name:    "Valid RSA token 2",
			token:   validRSASignedJWTToken2,
			wantErr: false,
		},
		{
			name:      "Empty token",
			token:     "",
			wantErr:   true,
			errString: "token contains an invalid number of segments",
		},
		{
			name:      "Invalid format",
			token:     "not.a.jwt",
			wantErr:   true,
			errString: "failed to parse token: token is malformed:",
		},
		{
			name:      "Unknown key ID",
			token:     "eyJhbGciOiJSUzI1NiIsImtpZCI6ImtleV8zIiwidHlwIjoiSldUIn0.eyJjdXN0b21lcl9pZCI6ImN1c3RvbWVyXzMiLCJleHAiOjEwMzc2MjQwMTA3LCJmZWF0dXJlcyI6WyJmZWF0dXJlXzMiLCJmZWF0dXJlXzMzIl0sImlhdCI6MTczNjI0MzcwNywibGljZW5zZV9pZCI6ImxpY2Vuc2VfMyIsImxpY2Vuc2VfdHlwZSI6InByb2RfdGllcl8zIiwibGljZW5zZV92ZXJzaW9uIjoidl8zIiwibGltaXRhdGlvbnMiOnsidGllcl8zIjoibm8ifSwibWV0YWRhdGEiOnsic29tZXRoaW5nMyI6InNvbWV0aGluZzNfdmFsdWUifSwicHJvZHVjdCI6IkJhY2FsaGF1In0.dzWz7FHKWM0SuVDISzxJ7lfXpXOunrJ01PeRjufvhxGv4g6bGwfKFRjiQYEuwrzst_k1zw0d5XL2VWhhjTpETew7728cubugbiA7222FgLdDk-y2hitEsf_cn-Wd3-da56huBO4tuPZifrT_NEdhbnXzB90Xd6ga3xK-oTsjXniHIj6tdLn9rH4Exp44QYLSj_YTlOm5JMUSWdD70Fnwx5SlWSST1yx5eGTJ71rRTr-tN6Y5_1tywK6a1Tf3iBmW6y4-jA-94zIfvI2wHvmZXen3KRJKra31pKpjjlLPHpqZ3_tVVV7R1sz4PME4sSlh3yhj4oIO-Ixu-eSo1yDWHw",
			wantErr:   true,
			errString: "key not found: kid \"key_3\"",
		},
		{
			name:      "Invalid signature",
			token:     "eyJhbGciOiJSUzI1NiIsImtpZCI6ImtleV8xIiwidHlwIjoiSldUIn0.eyJjdXN0b21lcl9pZCI6ImN1c3RvbWVyXzEiLCJleHAiOjEwMzc2MjQwMTA3LCJmZWF0dXJlcyI6WyJmZWF0dXJlXzEiLCJmZWF0dXJlXzExIl0sImlhdCI6MTczNjI0MzcwNywibGljZW5zZV9pZCI6ImxpY2Vuc2VfMSIsImxpY2Vuc2VfdHlwZSI6InByb2RfdGllcl8xIiwibGljZW5zZV92ZXJzaW9uIjoidl8xIiwibGltaXRhdGlvbnMiOnsidGllcl8xIjoibm8ifSwibWV0YWRhdGEiOnsic29tZXRoaW5nMSI6InNvbWV0aGluZzFfdmFsdWUifSwicHJvZHVjdCI6IkJhY2FsaGF1In0.XyHdItfNma4zkwwwB_M_xgHGqRhTtNmsdPx491msaalfEKAKDYqCMsE6DhL6cKWRqKsXGx27kaBCun1chiYf_yz1rSfMZny-XdakqIg_ENburNFrNSePn-kGhUPmQLzK9JV4Iph2hTWB6dJ8rFqYewDiJ6yfX_AVymmst4OziPmBiPeDcEtjjSR8MEQynRiKUup76fKVgsgXvT-eUHURXOWBcADEw-UvbyKgEt7FB-baZSryReJTyStpA7E64OFB4fNwfk3h70",
			wantErr:   true,
			errString: "token signature is invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := validator.ValidateToken(tt.token)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errString != "" {
					assert.Contains(t, err.Error(), tt.errString)
				}
				assert.Nil(t, claims)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, claims)
			}
		})
	}
}

func TestNewLicenseValidatorFromFile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		wantErr  bool
	}{
		{
			name:     "Non-existent file",
			filename: "non_existent.json",
			wantErr:  true,
		},
		{
			name:     "Invalid file permissions",
			filename: "/root/test.json", // Assuming no permission to access /root
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator, err := NewLicenseValidatorFromFile(tt.filename)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, validator)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, validator)
			}
		})
	}
}

func TestValidTokenClaims(t *testing.T) {
	validator, err := NewLicenseValidatorFromJSON(testJWKS)
	require.NoError(t, err)

	tests := []struct {
		name     string
		token    string
		validate func(*testing.T, *LicenseClaims)
	}{
		{
			name:  "RSA token claims",
			token: validRSASignedJWTToken1,
			validate: func(t *testing.T, claims *LicenseClaims) {
				assert.Equal(t, "license_1", claims.LicenseID)
				assert.Equal(t, "prod_tier_1", claims.LicenseType)
				assert.Equal(t, "v_1", claims.LicenseVersion)
				assert.Equal(t, "customer_1", claims.CustomerID)
				assert.Equal(t, []string{"feature_1", "feature_11"}, claims.Features)
				assert.Equal(t, map[string]string{"tier_1": "no"}, claims.Limitations)
				assert.Equal(t, map[string]string{"something1": "something1_value"}, claims.Metadata)
				assert.True(t, claims.ExpiresAt.Unix() == 10376240107)
			},
		},
		{
			name:  "RSA token claims 2",
			token: validRSASignedJWTToken2,
			validate: func(t *testing.T, claims *LicenseClaims) {
				assert.Equal(t, "license_2", claims.LicenseID)
				assert.Equal(t, "prod_tier_2", claims.LicenseType)
				assert.Equal(t, "v_2", claims.LicenseVersion)
				assert.Equal(t, "customer_2", claims.CustomerID)
				assert.Equal(t, []string{"feature_2", "feature_22"}, claims.Features)
				assert.Equal(t, map[string]string{"tier_2": "no"}, claims.Limitations)
				assert.Equal(t, map[string]string{"something2": "something2_value"}, claims.Metadata)
				assert.True(t, claims.ExpiresAt.Unix() == 10376240107)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := validator.ValidateToken(tt.token)
			require.NoError(t, err)
			require.NotNil(t, claims)
			tt.validate(t, claims)
		})
	}
}
