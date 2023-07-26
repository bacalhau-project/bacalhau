package secrets

import (
	"fmt"
	"strings"
)

type EncryptedValue struct {
	KeyID string
	Data  string
}

func NewEncryptedValue(keyID string, data string) *EncryptedValue {
	return &EncryptedValue{
		KeyID: keyID,
		Data:  data,
	}
}

func (e *EncryptedValue) String() string {
	return fmt.Sprintf("ENC[id:%s;data:%s]", e.KeyID, e.Data)
}

func ParseEncryptedValue(val string) (*EncryptedValue, error) {
	if !strings.HasPrefix(val, "ENC[") {
		return nil, fmt.Errorf("value is not encrypted")
	}

	ev := &EncryptedValue{}

	v := val[4:]
	v = v[:len(v)-1]

	segments := strings.Split(v, ";")
	for _, segment := range segments {
		parts := strings.Split(segment, ":")
		if len(parts) != 2 {
			return nil, fmt.Errorf("not enough parts to decode in encrypted value")
		}
		k, v := parts[0], parts[1]
		if k == "id" {
			ev.KeyID = v
		} else if k == "data" {
			ev.Data = v
		}
	}

	return ev, nil
}
