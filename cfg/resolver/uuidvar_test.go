package resolver

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUUIDVar(t *testing.T) {
	r := &UUIDVar{Old: "$UUID"}

	result, err := r.Resolve("$UUID")
	assert.NoError(t, err)

	matched, err := regexp.MatchString(`(?i)^[0-9A-F]{8}-[0-9A-F]{4}-[4][0-9A-F]{3}-[89AB][0-9A-F]{3}-[0-9A-F]{12}$`, result)
	assert.NoError(t, err)
	assert.True(t, matched)
}
