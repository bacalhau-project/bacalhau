package util

import (
	"crypto/rsa"
	"fmt"

	"github.com/spf13/cobra"
)

func GetPublicKey(cmd *cobra.Command) (*rsa.PublicKey, string, error) {
	client := GetAPIClient(cmd.Context())
	ctx := cmd.Context()

	pubkey, keyid, err := client.PublicKey(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("failed to retrieve node's public key: %s", err)
	}

	return pubkey, keyid, nil
}
