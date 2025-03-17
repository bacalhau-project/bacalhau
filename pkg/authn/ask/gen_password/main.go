package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"

	"github.com/bacalhau-project/bacalhau/pkg/lib/policy"
)

const saltLength = 32

// verifyPassword checks if the provided password matches the stored hash using the given salt
func verifyPassword(password []byte, storedHash []byte, salt []byte) (bool, error) {
	computedHash, err := policy.Scrypt(password, salt)
	if err != nil {
		return false, fmt.Errorf("failed to compute hash: %w", err)
	}

	// Compare the computed hash with the stored hash
	if len(computedHash) != len(storedHash) {
		return false, nil
	}

	// Use a constant-time comparison to prevent timing attacks
	match := true
	for i := range computedHash {
		if computedHash[i] != storedHash[i] {
			match = false
		}
	}
	return match, nil
}

func main() {
	//fmt.Fprintf(os.Stderr, "Password: ")
	//
	//password, err := term.ReadPassword(int(os.Stdin.Fd()))
	//if err != nil {
	//	fmt.Fprintln(os.Stderr, err.Error())
	//	os.Exit(1)
	//}
	password := "vroom"
	salt := make([]byte, saltLength)
	_, err := rand.Read(salt)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	hash, err := policy.Scrypt([]byte(password), salt)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	// Test password verification
	match, err := verifyPassword([]byte(password), hash, salt)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error verifying password:", err.Error())
		os.Exit(1)
	}

	if match {
		fmt.Println("Password verification successful!")
	} else {
		fmt.Println("Password verification failed!")
	}

	// Original output
	output := [][]byte{hash, salt}
	err = json.NewEncoder(os.Stdout).Encode(output)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
