package credsecurity

import (
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestNewBcryptManager(t *testing.T) {
	t.Run("WithValidCost", func(t *testing.T) {
		expectedCost := 11
		manager := NewBcryptManager(expectedCost)

		if manager.cost != expectedCost {
			t.Errorf("Expected cost to be %d, got %d", expectedCost, manager.cost)
		}
	})

	t.Run("WithInvalidCost", func(t *testing.T) {
		// Test with cost below minimum
		manager := NewBcryptManager(bcrypt.MinCost - 1)

		if manager.cost != bcrypt.DefaultCost {
			t.Errorf("Expected cost to default to %d, got %d", bcrypt.DefaultCost, manager.cost)
		}
	})
}

func TestNewDefaultBcryptManager(t *testing.T) {
	manager := NewDefaultBcryptManager()

	// The default cost should be 12 as specified in the implementation
	expectedCost := 12
	if manager.cost != expectedCost {
		t.Errorf("Expected default cost to be %d, got %d", expectedCost, manager.cost)
	}
}

func TestHashPassword(t *testing.T) {
	t.Run("ValidPassword", func(t *testing.T) {
		manager := NewBcryptManager(bcrypt.MinCost) // Use min cost for faster tests
		password := "secure_password123"

		hash, err := manager.HashPassword(password)

		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if hash == "" {
			t.Error("Expected non-empty hash")
		}

		// Verify the hash starts with the bcrypt identifier
		if !strings.HasPrefix(hash, "$2a$") {
			t.Errorf("Hash doesn't start with bcrypt identifier: %s", hash)
		}
	})

	t.Run("EmptyPassword", func(t *testing.T) {
		manager := NewBcryptManager(bcrypt.MinCost)

		hash, err := manager.HashPassword("")

		if err == nil {
			t.Error("Expected error for empty password, got none")
		}

		if hash != "" {
			t.Errorf("Expected empty hash, got: %s", hash)
		}
	})
}

func TestVerifyPassword(t *testing.T) {
	t.Run("CorrectPassword", func(t *testing.T) {
		manager := NewBcryptManager(bcrypt.MinCost) // Use min cost for faster tests
		password := "correct_password456"

		// First hash the password
		hash, err := manager.HashPassword(password)
		if err != nil {
			t.Fatalf("Failed to hash password: %v", err)
		}

		// Then verify with the same password
		err = manager.VerifyPassword(password, hash)

		if err != nil {
			t.Errorf("Expected successful verification, got error: %v", err)
		}
	})

	t.Run("IncorrectPassword", func(t *testing.T) {
		manager := NewBcryptManager(bcrypt.MinCost)
		password := "original_password789"
		wrongPassword := "wrong_password789"

		// Hash the original password
		hash, err := manager.HashPassword(password)
		if err != nil {
			t.Fatalf("Failed to hash password: %v", err)
		}

		// Try to verify with the wrong password
		err = manager.VerifyPassword(wrongPassword, hash)

		if err == nil {
			t.Error("Expected error for incorrect password, got none")
		}
	})

	t.Run("EmptyPassword", func(t *testing.T) {
		manager := NewBcryptManager(bcrypt.MinCost)
		validHash := "$2a$04$NQvJJHvbPMiQqMP3LbRaJeTbVFLLe/c.4WhTDXYFdHnnXhJ44/fQ2" // Example hash

		err := manager.VerifyPassword("", validHash)

		if err == nil {
			t.Error("Expected error for empty password, got none")
		}
	})

	t.Run("EmptyHash", func(t *testing.T) {
		manager := NewBcryptManager(bcrypt.MinCost)
		password := "some_password"

		err := manager.VerifyPassword(password, "")

		if err == nil {
			t.Error("Expected error for empty hash, got none")
		}
	})
}

func TestIsBcryptHash(t *testing.T) {
	manager := NewBcryptManager(bcrypt.MinCost)

	t.Run("ValidHash", func(t *testing.T) {
		// Create a valid hash
		password := "test_password"
		validHash, err := manager.HashPassword(password)
		if err != nil {
			t.Fatalf("Failed to create test hash: %v", err)
		}

		result := manager.IsBcryptHash(validHash)

		if !result {
			t.Errorf("Expected true for valid hash, got false. Hash: %s", validHash)
		}
	})

	t.Run("ValidHashExamples", func(t *testing.T) {
		// Test with known valid hash formats
		validHashes := []string{
			"$2a$04$p0mOrE8nAY.RjOehZJRUCOjToyfiVOPUbn8RMwTHOQTiSHsUJdUXa",
		}

		for _, hash := range validHashes {
			result := manager.IsBcryptHash(hash)
			if !result {
				t.Errorf("Expected true for valid hash, got false. Hash: %s", hash)
			}
		}
	})

	t.Run("EmptyString", func(t *testing.T) {
		result := manager.IsBcryptHash("")

		if result {
			t.Error("Expected false for empty string, got true")
		}
	})

	t.Run("InvalidHash", func(t *testing.T) {
		invalidHashes := []string{
			"not-a-hash",
			"$1$abcdefgh$ijklmnopqrstuv", // MD5 hash format
			"$6$abcdefgh$ijklmnopqrstuv", // SHA-512 hash format
			"$2a$",                       // Too short
			"$2a$10abcdefghijklmnopqrstuuvwxyzabcdefghijklmnopqrstuuvwx",  // Missing $ after cost
			"$2c$10$abcdefghijklmnopqrstuuVFjDnPYNOQf/uXU5n82PnKh1eVSLGS", // Invalid prefix
		}

		for _, hash := range invalidHashes {
			result := manager.IsBcryptHash(hash)
			if result {
				t.Errorf("Expected false for invalid hash, got true. Hash: %s", hash)
			}
		}
	})
}
