package auth

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/credsecurity"
	"github.com/spf13/cobra"
)

// cSpell:disable
// mockStdin replaces os.Stdin with a pipe for testing
func mockStdin(t *testing.T, input string) func() {
	t.Helper()

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("couldn't create pipe: %v", err)
	}

	origStdin := os.Stdin
	os.Stdin = reader

	// Write input asynchronously
	go func() {
		defer func() { _ = writer.Close() }()
		io.WriteString(writer, input)
	}()

	// Return function to restore original stdin
	return func() {
		os.Stdin = origStdin
	}
}

// getMockCommand returns a command with a buffer for capturing output
func getMockCommand(t *testing.T) (*cobra.Command, *bytes.Buffer) {
	t.Helper()

	cmd := &cobra.Command{Use: "test"}
	outputBuf := &bytes.Buffer{}
	cmd.SetOut(outputBuf)
	cmd.SetErr(outputBuf)

	return cmd, outputBuf
}

// Helper function that directly tests the password hashing functionality
// This avoids dealing with terminal input mocking issues
func testPasswordHashing(t *testing.T, password string) (string, error) {
	t.Helper()

	// Create a manager and directly hash the password
	bcryptManager := credsecurity.NewDefaultBcryptManager()
	return bcryptManager.HashPassword(password)
}

// mockTerminal sets up terminal mocking and returns a cleanup function
func mockTerminal(isTTY bool, password string) func() {
	// Save original functions
	originalIsTerminal := isTerminalCheck
	originalReadPassword := readPasswordFunc

	// Set up mocks
	isTerminalCheck = func(fd int) bool {
		return isTTY
	}

	readPasswordFunc = func(fd int) ([]byte, error) {
		return []byte(password), nil
	}

	// Return cleanup function
	return func() {
		isTerminalCheck = originalIsTerminal
		readPasswordFunc = originalReadPassword
	}
}

// Test basic password hashing functionality
func TestHashPasswordBasicFunctionality(t *testing.T) {
	// Test with valid password
	t.Run("ValidPassword", func(t *testing.T) {
		password := "validpassword"
		hash, err := testPasswordHashing(t, password)

		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if !strings.HasPrefix(hash, "$2a$") {
			t.Errorf("Expected bcrypt hash starting with $2a$, got: %s", hash)
		}

		// Verify the hash is valid
		bcryptManager := credsecurity.NewDefaultBcryptManager()
		err = bcryptManager.VerifyPassword(password, hash)
		if err != nil {
			t.Errorf("Generated hash didn't verify correctly: %v", err)
		}
	})

	// Test with empty password
	t.Run("EmptyPassword", func(t *testing.T) {
		_, err := testPasswordHashing(t, "")

		if err == nil {
			t.Error("Expected error for empty password, got none")
		}
	})

	// Test with password that's too long
	t.Run("TooLongPassword", func(t *testing.T) {
		// We need to test the validation logic in the command itself
		// since the BCryptManager doesn't have a length limit

		options := NewHashPasswordOptions()
		cmd, _ := getMockCommand(t)

		// Create a long password (101 characters) and add a newline
		longPassword := strings.Repeat("a", 101) + "\n"

		// Mock stdin to provide the long password
		restoreStdin := mockStdin(t, longPassword)
		defer restoreStdin()

		// Mock terminal to be non-TTY
		restoreTerminal := mockTerminal(false, "")
		defer restoreTerminal()

		// The command will read from stdin, but we're not actually mocking the terminal
		// functions. This will only test the password length check, not the TTY detection.
		err := options.runHashPassword(cmd)

		if err == nil {
			t.Error("Expected error for password exceeding max length, got none")
		}

		if !strings.Contains(err.Error(), "exceeds maximum length") {
			t.Errorf("Expected error about password length, got: %s", err.Error())
		}
	})
}

// Test that the hash verification works
func TestPasswordVerification(t *testing.T) {
	// Test a password and its verification
	t.Run("CorrectPassword", func(t *testing.T) {
		password := "correct_password456"
		hash, err := testPasswordHashing(t, password)
		if err != nil {
			t.Fatalf("Failed to hash password: %v", err)
		}

		bcryptManager := credsecurity.NewDefaultBcryptManager()
		err = bcryptManager.VerifyPassword(password, hash)
		if err != nil {
			t.Errorf("Expected successful verification, got error: %v", err)
		}
	})

	// Test with wrong password
	t.Run("IncorrectPassword", func(t *testing.T) {
		password := "original_password789"
		wrongPassword := "wrong_password789"

		hash, err := testPasswordHashing(t, password)
		if err != nil {
			t.Fatalf("Failed to hash password: %v", err)
		}

		bcryptManager := credsecurity.NewDefaultBcryptManager()
		err = bcryptManager.VerifyPassword(wrongPassword, hash)
		if err == nil {
			t.Error("Expected error for incorrect password, got none")
		}
	})
}

// Test TTY input path
func TestHashPasswordTTYInput(t *testing.T) {
	t.Run("TTYPasswordInput", func(t *testing.T) {
		// Mock TTY mode and password input
		mockPassword := "passwordFromTTY"
		restoreTerminal := mockTerminal(true, mockPassword)
		defer restoreTerminal()

		cmd, outputBuf := getMockCommand(t)

		// Execute
		options := NewHashPasswordOptions()
		err := options.runHashPassword(cmd)

		// Assert
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// Output should contain prompt and hash
		output := outputBuf.String()
		if !strings.Contains(output, "Enter password:") {
			t.Error("Output should contain password prompt")
		}

		// Extract the hash (after the newline)
		lines := strings.Split(strings.TrimSpace(output), "\n")
		hashLine := lines[len(lines)-1]

		// Verify it's a valid hash format
		if !strings.HasPrefix(hashLine, "$2a$") {
			t.Errorf("Expected bcrypt hash starting with $2a$, got: %s", hashLine)
		}

		// Verify hash matches the input
		bcryptManager := credsecurity.NewDefaultBcryptManager()
		err = bcryptManager.VerifyPassword(mockPassword, hashLine)
		if err != nil {
			t.Errorf("Generated hash didn't verify correctly: %v", err)
		}
	})
}

// TestFullCommandFlow tests the entire command flow and verifies the generated hashes
func TestFullCommandFlow(t *testing.T) {
	t.Run("NonTTYValidPasswordFlow", func(t *testing.T) {
		// Setup
		testPassword := "securePassword123\n"
		restoreStdin := mockStdin(t, testPassword)
		defer restoreStdin()

		restoreTerminal := mockTerminal(false, "")
		defer restoreTerminal()

		cmd, outputBuf := getMockCommand(t)

		// Execute command
		options := NewHashPasswordOptions()
		err := options.runHashPassword(cmd)

		// Check for errors
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// Get the hash from output
		hashOutput := strings.TrimSpace(outputBuf.String())

		// Verify hash format
		if !strings.HasPrefix(hashOutput, "$2a$") {
			t.Errorf("Expected bcrypt hash starting with $2a$, got: %s", hashOutput)
		}

		// Verify the hash using BcryptManager
		bcryptManager := credsecurity.NewDefaultBcryptManager()
		err = bcryptManager.VerifyPassword(strings.TrimSpace(testPassword), hashOutput)
		if err != nil {
			t.Errorf("Failed to verify generated hash with original password: %v", err)
		}

		// Verify the hash fails with wrong password
		wrongPassword := "wrongPassword123"
		err = bcryptManager.VerifyPassword(wrongPassword, hashOutput)
		if err == nil {
			t.Error("Hash verification should have failed with wrong password")
		}

		// Verify the hash using the IsBcryptHash method
		if !bcryptManager.IsBcryptHash(hashOutput) {
			t.Errorf("Output hash failed format validation: %s", hashOutput)
		}
	})

	t.Run("HashFormatValidation", func(t *testing.T) {
		// Generate several hashes and verify they all pass format validation
		bcryptManager := credsecurity.NewDefaultBcryptManager()

		testPasswords := []string{
			"simple",
			"WithNumbers123",
			"With.Special!Characters@",
			"a-longer-passphrase-that-is-more-secure",
		}

		for _, password := range testPasswords {
			t.Run(fmt.Sprintf("Password_%s", password[:5]), func(t *testing.T) {
				hash, err := bcryptManager.HashPassword(password)
				if err != nil {
					t.Fatalf("Failed to hash password: %v", err)
				}

				// Verify format using IsBcryptHash
				if !bcryptManager.IsBcryptHash(hash) {
					t.Errorf("Hash failed format validation: %s", hash)
				}

				// Verify hash can be verified with original password
				err = bcryptManager.VerifyPassword(password, hash)
				if err != nil {
					t.Errorf("Failed to verify hash with original password: %v", err)
				}

				// Verify different passwords generate different hashes
				hash2, _ := bcryptManager.HashPassword(password)
				if hash == hash2 {
					t.Error("Two hashes of the same password should be different due to salt")
				}
			})
		}
	})

	t.Run("MalformedHashVerification", func(t *testing.T) {
		bcryptManager := credsecurity.NewDefaultBcryptManager()
		password := "testpassword"

		// Test with malformed hashes
		malformedHashes := []string{
			"",                             // Empty string
			"not-a-hash",                   // Not a hash at all
			"$2a$10$",                      // Truncated hash
			"$2a$10$abcdefghijklmnopqrstu", // Incomplete hash
			"$2x$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy", // Invalid prefix
		}

		for _, badHash := range malformedHashes {
			t.Run(fmt.Sprintf("BadHash_%d", len(badHash)), func(t *testing.T) {
				// Verify it's not detected as a valid hash format
				if bcryptManager.IsBcryptHash(badHash) {
					t.Errorf("Invalid hash '%s' was incorrectly identified as valid", badHash)
				}

				// Verify password verification fails
				err := bcryptManager.VerifyPassword(password, badHash)
				if err == nil {
					t.Errorf("Verification with invalid hash '%s' should have failed", badHash)
				}
			})
		}
	})

	t.Run("CommandOutputVerification", func(t *testing.T) {
		// Set of test passwords to try
		testCases := []struct {
			name     string
			password string
		}{
			{"Simple", "simple123\n"},
			{"Complex", "C0mplex!P@ssw0rd#123\n"},
			{"WithSpaces", "this is a passphrase with spaces\n"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Setup command execution with the test password
				restoreStdin := mockStdin(t, tc.password)
				defer restoreStdin()

				restoreTerminal := mockTerminal(false, "")
				defer restoreTerminal()

				cmd, outputBuf := getMockCommand(t)
				options := NewHashPasswordOptions()

				// Execute the command
				err := options.runHashPassword(cmd)
				if err != nil {
					t.Fatalf("Command execution failed: %v", err)
				}

				// Get the hash from the command output
				commandHash := strings.TrimSpace(outputBuf.String())

				// Create a BcryptManager for verification
				bcryptManager := credsecurity.NewDefaultBcryptManager()

				// Verify the password against the generated hash
				trimmedPassword := strings.TrimSpace(tc.password) // Remove newline
				err = bcryptManager.VerifyPassword(trimmedPassword, commandHash)
				if err != nil {
					t.Errorf("Failed to verify password against command output hash: %v", err)
				}

				// Check if it's a valid bcrypt hash format
				if !bcryptManager.IsBcryptHash(commandHash) {
					t.Errorf("Command output is not a valid bcrypt hash: %s", commandHash)
				}

				// Now hash the password directly using the manager
				directHash, err := bcryptManager.HashPassword(trimmedPassword)
				if err != nil {
					t.Fatalf("Failed to directly hash password: %v", err)
				}

				// Verify both the command hash and direct hash work with the password
				err = bcryptManager.VerifyPassword(trimmedPassword, directHash)
				if err != nil {
					t.Errorf("Failed to verify password against directly generated hash: %v", err)
				}

				// Verify both hashes are valid bcrypt hashes
				if !strings.HasPrefix(commandHash, "$2a$") || !strings.HasPrefix(directHash, "$2a$") {
					t.Errorf("One or both hashes don't have proper bcrypt format")
				}

				// Verify both hashes are different (due to salt) but both work
				if directHash == commandHash {
					t.Errorf("Expected different hashes due to salt, but got identical hashes")
				}
			})
		}
	})
}
