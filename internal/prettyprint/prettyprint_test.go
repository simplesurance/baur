package prettyprint

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAsStringReturnsIndentedJSON(t *testing.T) {
	got := AsString(map[string]int{"a": 1})

	assert.Equal(t, "{\n  \"a\": 1\n}", got)
}

func TestAsStringFallsBackToGoSyntaxIfNotMarshalable(t *testing.T) {
	got := AsString(make(chan int))

	assert.Contains(t, got, "0x", "falls back to %+v formatting")
}

func TestTruncatedStrSlice(t *testing.T) {
	sl := []string{"a", "b", "c"}

	assert.Equal(t, "a, b, c", TruncatedStrSlice(sl, 3), "len == max is not truncated")
	assert.Equal(t, "a, b, c", TruncatedStrSlice(sl, 10))
	assert.Equal(t, "a, b, [...]", TruncatedStrSlice(sl, 2))
	assert.Equal(t, ", [...]", TruncatedStrSlice(sl, 0))
	assert.Empty(t, TruncatedStrSlice(nil, 2))
}
