package resolver

import (
	"strings"

	"github.com/google/uuid"
)

type UUIDVar struct {
	Old string
}

func (r *UUIDVar) Resolve(in string) (string, error) {
	return strings.Replace(in, r.Old, uuid.New().String(), -1), nil
}
