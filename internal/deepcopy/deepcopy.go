package deepcopy

import (
	"bytes"
	"encoding/gob"
)

func Copy(from, to interface{}) error {
	var buf bytes.Buffer

	err := gob.NewEncoder(&buf).Encode(from)
	if err != nil {
		return err
	}

	err = gob.NewDecoder(&buf).Decode(to)
	if err != nil {
		return err
	}

	return nil
}

func MustCopy(from, to interface{}) {
	err := Copy(from, to)
	if err != nil {
		panic(err)
	}
}
