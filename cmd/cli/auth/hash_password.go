package auth

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/bacalhau-project/bacalhau/pkg/credsecurity"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// Variables for terminal functions that can be overridden in tests
var (
	isTerminalCheck = func(fd int) bool {
		return term.IsTerminal(fd)
	}
	readPasswordFunc = func(fd int) ([]byte, error) {
		return term.ReadPassword(fd)
	}
)

// HashPasswordOptions contains options for the hash-password command
type HashPasswordOptions struct {
	// No options needed now
}

// NewHashPasswordOptions returns initialized HashPasswordOptions
func NewHashPasswordOptions() *HashPasswordOptions {
	return &HashPasswordOptions{}
}

// NewHashPasswordCmd returns a cobra command for hashing passwords
func NewHashPasswordCmd() *cobra.Command {
	o := NewHashPasswordOptions()
	cmd := &cobra.Command{
		Use:   "hash-password",
		Short: "Hash a password using bcrypt",
		Long:  "Generate a secure bcrypt hash from a password provided via stdin",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.runHashPassword(cmd)
		},
	}

	return cmd
}

// runHashPassword executes the hash-password command
func (o *HashPasswordOptions) runHashPassword(cmd *cobra.Command) error {
	// Create a bcrypt manager with default cost (12)
	bcryptManager := credsecurity.NewDefaultBcryptManager()

	// Determine if we're reading from a TTY
	// int conversion is needed for windows architecture
	stdInIsTTY := isTerminalCheck(int(syscall.Stdin)) //nolint:unconvert

	var password string
	var err error

	if stdInIsTTY {
		// If we have a TTY, prompt for password with no echo
		_, _ = fmt.Fprint(cmd.OutOrStdout(), "Enter password: ")
		// int conversion is needed for windows architecture
		passwordBytes, err := readPasswordFunc(int(syscall.Stdin)) //nolint:unconvert
		if err != nil {
			log.Debug().Err(err).Msg("failed to read password")
			return fmt.Errorf("failed to read password: %w", err)
		}
		password = string(passwordBytes)
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "") // Print a newline after the password input
	} else {
		// If we're not on a TTY, read from stdin
		reader := bufio.NewReader(os.Stdin)
		password, err = reader.ReadString('\n')
		if err != nil {
			log.Debug().Err(err).Msg("failed to read password from stdin")
			return fmt.Errorf("failed to read password from stdin: %w", err)
		}
		password = strings.TrimSpace(password)
	}

	// Check if password is empty
	if password == "" {
		return fmt.Errorf("password cannot be empty")
	}

	// Check if password exceeds maximum length
	const maxPasswordLength = 100
	if len(password) > maxPasswordLength {
		return fmt.Errorf("password exceeds maximum length of %d characters", maxPasswordLength)
	}

	// Hash the password
	hashedPassword, err := bcryptManager.HashPassword(password)
	if err != nil {
		log.Debug().Err(err).Msg("failed to hash password")
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Output the hash
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), hashedPassword)
	return nil
}
