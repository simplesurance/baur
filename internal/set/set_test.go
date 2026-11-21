package set

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFromRemovesDuplicates(t *testing.T) {
	s := From([]string{"a", "b", "a"})

	assert.Len(t, s, 2)
	assert.True(t, s.Contains("a"))
	assert.True(t, s.Contains("b"))
	assert.False(t, s.Contains("c"))
}

func TestFromNilSliceReturnsUsableEmptySet(t *testing.T) {
	s := From[string](nil)

	assert.Empty(t, s)
	assert.False(t, s.Contains("a"))

	s.Add("a")
	assert.True(t, s.Contains("a"))
}

func TestAddIsIdempotent(t *testing.T) {
	s := Set[int]{}
	s.Add(1)
	s.Add(1)

	assert.Len(t, s, 1)
}

func TestSlice(t *testing.T) {
	assert.ElementsMatch(t, []int{1, 2, 3}, From([]int{3, 1, 2, 1}).Slice())
	assert.Empty(t, Set[int]{}.Slice())
}
