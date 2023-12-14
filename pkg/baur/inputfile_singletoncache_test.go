package baur

import (
	"path/filepath"
	"testing"

	"github.com/simplesurance/baur/v3/internal/digest/sha384"

	"github.com/stretchr/testify/require"
)

func TestInputFileSingletonCache(t *testing.T) {
	c := NewInputFileSingletonCache(sha384.File)
	f1Path := filepath.Join("etc", "issue")
	f2Path := filepath.Join("etc", "motd")
	f1 := c.CreateOrGetInputFile(f1Path, "issue")
	f2 := c.CreateOrGetInputFile(f1Path, "issue")
	f3 := c.CreateOrGetInputFile(f2Path, "motd")

	require.Same(t, f1, f2)
	require.NotSame(t, f1, f3)
}
