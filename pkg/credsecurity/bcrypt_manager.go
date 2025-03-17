package credsecurity

import (
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// BcryptManager provides functionality for hashing and verifying passwords using bcrypt
type BcryptManager struct {
	// cost determines the computational cost of hashing passwords
	// higher cost means more secure but slower
	cost int
}

// NewDefaultBcryptManager creates a new BcryptManager with a reasonable default cost value
// This uses a cost of 12, which provides good security while maintaining acceptable performance
func NewDefaultBcryptManager() *BcryptManager {
	return NewBcryptManager(12)
}

// NewBcryptManager creates a new BcryptManager with the specified cost
// If cost is less than bcrypt.MinCost, the default cost (bcrypt.DefaultCost) will be used
func NewBcryptManager(cost int) *BcryptManager {
	if cost < bcrypt.MinCost {
		cost = bcrypt.DefaultCost
	}
	return &BcryptManager{
		cost: cost,
	}
}

// HashPassword generates a bcrypt hash from a password
func (bm *BcryptManager) HashPassword(password string) (string, error) {
	if password == "" {
		return "", errors.New("password cannot be empty")
	}

	// Generate the bcrypt hash
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bm.cost)
	if err != nil {
		return "", err
	}

	return string(hashedBytes), nil
}

// VerifyPassword checks if the provided password matches the stored hash
func (bm *BcryptManager) VerifyPassword(password, hashedPassword string) error {
	if password == "" {
		return errors.New("password cannot be empty")
	}
	if hashedPassword == "" {
		return errors.New("hashed password cannot be empty")
	}

	// Compare the provided password with the hashed password
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

// IsBcryptHash checks if a string is a valid bcrypt hash
func (bm *BcryptManager) IsBcryptHash(hash string) bool {
	if hash == "" {
		return false
	}

	// Check if the hash has the bcrypt format
	// Bcrypt hashes start with $2a$, $2b$, or $2y$ followed by the cost and a 22-character salt and 31-character hash
	validPrefixes := []string{"$2a$", "$2b$", "$2y$"}

	hasValidPrefix := false
	for _, prefix := range validPrefixes {
		if strings.HasPrefix(hash, prefix) {
			hasValidPrefix = true
			break
		}
	}

	if !hasValidPrefix {
		return false
	}

	// Basic length check (prefix + cost digits + $ + salt + hash)
	// Minimum length: $2a$ (4) + 2 digits + $ (1) + 22 chars salt + 31 chars hash = 60
	if len(hash) < 60 {
		return false
	}

	// Try to verify with a dummy password to see if it's a valid hash
	// This will return an error for invalid hashes or if the password doesn't match (which we expect)
	// We're only checking if the error is of type bcrypt.HashError which indicates invalid format
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte("dummy"))
	if err != nil && errors.Is(err, bcrypt.ErrHashTooShort) {
		return false
	}

	return true
}
