package jobstore

import (
	"bytes"
	"encoding/gob"
)

type MarshalFunc func(v any) ([]byte, error)
type UnmarshalFunc func(data []byte, obj interface{}) error

func BinaryMarshal(obj any) ([]byte, error) {
	var b bytes.Buffer
	e := gob.NewEncoder(&b)

	if err := e.Encode(obj); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

func BinaryUnmarshal(data []byte, target interface{}) error {
	b := bytes.NewBuffer(data)
	d := gob.NewDecoder(b)
	if err := d.Decode(target); err != nil {
		return err
	}
	return nil
}
