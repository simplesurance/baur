package sha384_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v5/internal/digest"
	"github.com/simplesurance/baur/v5/internal/digest/sha384"
)

// Golden values were computed independently with sha384sum.
const (
	helloDigest = "sha384:59e1748777448c69de6b800d7a33bbfb9ff1b463e44354c3553bcdb9c666fa90125a3c79f90397bdf5f6a13de828684f"
	byeDigest   = "sha384:2c2d6082b325bacab39aca7fd683e63f2120ada9d1a0439ec916f3670d188ec88923a0923308a81e65c7b9d7dfdd870a"

	// sha384("sha384:2c2d...870a" + "sha384:59e1...684f"), sorted by sum
	sumHelloBye = "sha384:18cd445e53f3ff2e40432c26f24733ee24d4c77043f311be30b9b94a627fbaa2547b07e1af1b0051d8a0cdb2a3e2d73e"
	// sha384("sha256:ffff..." + "sha384:59e1..."), sorted by algorithm first
	sumMixed = "sha384:dd72c3e70cce3b011f83e72379a2f5382647d2b34eb3c8cc7250032a028e2d6a9c3cc698878eb9f39fe16affa345041b"
	// sha384("")
	sumEmpty = "sha384:38b060a751ac96384cd9327eb1b1e36a21fdb71114be07434c0cc7bf63f6e1da274edebfe76f65fbd51ad2f14898b95b"
)

func mustParse(t *testing.T, s string) *digest.Digest {
	t.Helper()

	d, err := digest.FromString(s)
	require.NoError(t, err)

	return d
}

func TestSum(t *testing.T) {
	t.Run("golden value", func(t *testing.T) {
		d, err := sha384.Sum([]*digest.Digest{mustParse(t, helloDigest), mustParse(t, byeDigest)})

		require.NoError(t, err)
		assert.Equal(t, sumHelloBye, d.String())
		assert.Equal(t, digest.SHA384, d.Algorithm)
	})

	t.Run("does not depend on the order of the digests", func(t *testing.T) {
		d, err := sha384.Sum([]*digest.Digest{mustParse(t, byeDigest), mustParse(t, helloDigest)})

		require.NoError(t, err)
		assert.Equal(t, sumHelloBye, d.String())
	})

	t.Run("digests are ordered by algorithm before their sum", func(t *testing.T) {
		sha256Digest := mustParse(t, "sha256:"+"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")

		d1, err := sha384.Sum([]*digest.Digest{sha256Digest, mustParse(t, helloDigest)})
		require.NoError(t, err)
		d2, err := sha384.Sum([]*digest.Digest{mustParse(t, helloDigest), sha256Digest})
		require.NoError(t, err)

		assert.Equal(t, sumMixed, d1.String())
		assert.Equal(t, sumMixed, d2.String())
	})

	t.Run("no digests", func(t *testing.T) {
		d, err := sha384.Sum(nil)

		require.NoError(t, err)
		assert.Equal(t, sumEmpty, d.String())
	})

	t.Run("a different set of digests gives a different sum", func(t *testing.T) {
		d, err := sha384.Sum([]*digest.Digest{mustParse(t, helloDigest)})

		require.NoError(t, err)
		assert.NotEqual(t, sumHelloBye, d.String())
	})

	t.Run("the passed slice is sorted in place", func(t *testing.T) {
		in := []*digest.Digest{mustParse(t, helloDigest), mustParse(t, byeDigest)}

		_, err := sha384.Sum(in)
		require.NoError(t, err)

		assert.Equal(t, byeDigest, in[0].String(), "characterization: Sum reorders the caller's slice")
	})
}

func TestFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "f")
	require.NoError(t, os.WriteFile(p, []byte("hello"), 0o600))

	d, err := sha384.File(p)

	require.NoError(t, err)
	assert.Equal(t, helloDigest, d.String())

	_, err = sha384.File(filepath.Join(t.TempDir(), "missing"))
	require.ErrorIs(t, err, os.ErrNotExist)
}
