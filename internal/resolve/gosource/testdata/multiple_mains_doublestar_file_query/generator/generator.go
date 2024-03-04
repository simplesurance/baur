package generator

import (
	"math/rand"

	"github.com/google/uuid"
)

// RandomNumber returns a random number
func RandomNumber() int {
	return rand.Int()
}

func UUID() string {
	return uuid.New().String()
}
