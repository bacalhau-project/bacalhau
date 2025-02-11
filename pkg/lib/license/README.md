## How to verify a license:

```go
// Initialize validator with embedded JWKS
validator, err := license.NewLicenseValidatorFromJSON(embeddedJWKS)
if err != nil {
    log.Fatalf("Failed to create validator: %v", err)
}

// Validate a license token
claims, err := validator.ValidateToken(licenseToken)
if err != nil {
    log.Fatalf("License validation failed: %v", err)
}

// Access license features and limitations
fmt.Printf("Licensed features: %v\n", claims.Features)
fmt.Printf("Usage limitations: %v\n", claims.Limitations)
```

## How to generate JWKS and JWT tokens for tests:

```python
from jwcrypto import jwk, jwt
import json
import os
from datetime import datetime, timedelta

key_name = 'my_key'

try:
    # Generate four RSA key pairs
    rsa_key_1 = jwk.JWK.generate(kty='RSA', size=2048, alg='RS256', use='sig', kid=key_name)
    rsa_key_1_public_key = json.loads(rsa_key_1.export_public())
    rsa_key_1_private_key = json.loads(rsa_key_1.export_private())

    # Save all key pairs
    with open(f'rsa_public_key_{key_name}.json', 'w') as f:
        json.dump(rsa_key_1_public_key, f, indent=2)
    with open(f'rsa_private_key_{key_name}.json', 'w') as f:
        json.dump(rsa_key_1_private_key, f, indent=2)

    # Set file permissions on Unix-like systems
    os.chmod(f'rsa_private_key_{key_name}.json', 0o600)
    os.chmod(f'rsa_public_key_{key_name}.json', 0o644)

    # Generate JWT tokens
    current_time = int(datetime.utcnow().timestamp())
    expiration_time = int((datetime.utcnow() + timedelta(days=100000)).timestamp())
    no_valid_before_time = int((datetime.utcnow() - timedelta(days=2)).timestamp())

    # Token 1 with first private key
    token1 = jwt.JWT(
        header={
            "alg": "RS256",
            "typ": "JWT",
            "kid": key_name
        },
        claims={
            "product": "Bacalhau",
            "license_id": "license_1",
            "license_type": "prod_tier_1",
            "license_version": "v1",
            "customer_id": "customer_1",
            "capabilities": {"tier_1": "no"},
            "metadata": {"something1": "something1_value"},
            "iat": current_time,
            "exp": expiration_time,
            "nbf": no_valid_before_time,
            "iss": "https://expanso.io/",
            "sub": "customer_1",
            "jti": "license_1"
        }
    )
    token1.make_signed_token(rsa_key_1)

    # Save tokens to files
    with open(f'jwt_token_{key_name}.txt', 'w') as f:
        f.write(token1.serialize())

    print("Keys and tokens have been saved to files successfully")

except Exception as e:
    print(f"An error occurred: {str(e)}")

# To verify tokens
try:
    # Read and verify all tokens
    with open(f'jwt_token_{key_name}.txt', 'r') as f:
        token_str = f.read()
    verified_token = jwt.JWT(key=rsa_key_1, jwt=token_str)
    print(f"\nToken {key_name} verified successfully")
    print(f"Token {key_name} claims:", verified_token.claims)

except Exception as e:
    print(f"Verification error: {str(e)}")
```