package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"

	"github.com/bacalhau-project/bacalhau/pkg/lib/policy"
	"golang.org/x/term"
)

const saltLength = 32

func main() {
	_, _ = fmt.Fprintf(os.Stderr, "Password: ")

	password, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	salt := make([]byte, saltLength)
	_, err = rand.Read(salt)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	hash, err := policy.Scrypt(password, salt)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	output := [][]byte{hash, salt}
	err = json.NewEncoder(os.Stdout).Encode(output)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
