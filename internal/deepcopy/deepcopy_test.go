package deepcopy

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type sample struct {
	Name  string
	List  []string
	Inner map[string]*inner
}

type inner struct{ V int }

func TestCopyIsDeep(t *testing.T) {
	from := sample{
		Name:  "a",
		List:  []string{"x", "y"},
		Inner: map[string]*inner{"k": {V: 1}},
	}

	var to sample
	require.NoError(t, Copy(&from, &to))
	assert.Equal(t, from, to)

	to.List[0] = "changed"
	to.Inner["k"].V = 99

	assert.Equal(t, "x", from.List[0], "slice must not be shared")
	assert.Equal(t, 1, from.Inner["k"].V, "pointer targets must not be shared")
}

func TestCopyReturnsErrorIfTargetIsNotAPointer(t *testing.T) {
	require.Error(t, Copy(sample{Name: "a"}, sample{}))
}

func TestCopyReturnsErrorOnUnencodableValue(t *testing.T) {
	require.Error(t, Copy(make(chan int), new(chan int)))
}

func TestMustCopy(t *testing.T) {
	var to sample
	MustCopy(&sample{Name: "a"}, &to)
	assert.Equal(t, "a", to.Name)

	assert.Panics(t, func() { MustCopy(make(chan int), new(chan int)) })
}
