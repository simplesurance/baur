package baur

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInputFileSingletonCache(t *testing.T) {
	c := NewInputFileSingletonCache()
	f1 := c.CreateOrGetInputFile("/etc/", "issue")
	f2 := c.CreateOrGetInputFile("/etc/", "issue")
	f3 := c.CreateOrGetInputFile("/etc/", "motd")

	require.Same(t, f1, f2)
	require.NotSame(t, f1, f3)
}
