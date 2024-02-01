package model

import (
	"encoding/base64"
)

type PublicKey []byte

func (pk PublicKey) MarshalText() ([]byte, error) {
	return []byte(base64.StdEncoding.EncodeToString(pk)), nil
}

func (pk *PublicKey) UnmarshalText(text []byte) error {
	ba, err := base64.StdEncoding.DecodeString(string(text))
	if err != nil {
		return err
	}
	*pk = ba
	return nil
}

func (pk PublicKey) String() string {
	return base64.StdEncoding.EncodeToString(pk)
}
