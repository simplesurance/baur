package digest

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	hex256 = strings.Repeat("ab", 32)
	hex384 = strings.Repeat("cd", 48)
)

func TestAlgorithmString(t *testing.T) {
	assert.Equal(t, "sha256", SHA256.String())
	assert.Equal(t, "sha384", SHA384.String())
	assert.Equal(t, "gitobjectid", GitObjectID.String())
	assert.Equal(t, "undefined", Algorithm(0).String())
	assert.Equal(t, "undefined", Algorithm(99).String())
}

func TestFromStringRoundTrip(t *testing.T) {
	for _, in := range []string{"sha256:" + hex256, "sha384:" + hex384} {
		t.Run(in[:6], func(t *testing.T) {
			d, err := FromString(in)
			require.NoError(t, err)

			assert.Equal(t, in, d.String())
		})
	}
}

func TestFromStringAcceptsSurroundingWhitespaceAndMixedCaseAlgorithm(t *testing.T) {
	d, err := FromString("  SHA256:" + hex256 + "\n")
	require.NoError(t, err)

	assert.Equal(t, SHA256, d.Algorithm)
	assert.Len(t, d.Sum, 32)
}

func TestFromStringAcceptsUppercaseHexAndNormalizesIt(t *testing.T) {
	d, err := FromString("sha256:" + strings.ToUpper(hex256))
	require.NoError(t, err)

	assert.Equal(t, "sha256:"+hex256, d.String())
}

func TestFromStringErrors(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", "exactly 1 ':'"},
		{"no colon", "sha256" + hex256, "exactly 1 ':'"},
		{"two colons", "sha256:" + hex256 + ":x", "exactly 1 ':'"},
		{"unsupported algorithm", "md5:" + hex256, `unsupported format "md5"`},
		{"gitobjectid can not be parsed", "gitobjectid:" + strings.Repeat("a", 40), "unsupported format"},
		{"sha256 too short", "sha256:" + hex256[:62], "hash length is 62, expected length 64"},
		{"sha256 with sha384 length", "sha256:" + hex384, "expected length 64"},
		{"sha384 too short", "sha384:" + hex384[:94], "hash length is 94, expected length 96"},
		{"not hex", "sha256:" + strings.Repeat("zz", 32), "converting string sum to hex failed"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d, err := FromString(tc.in)

			require.Error(t, err)
			assert.Nil(t, d)
			assert.Contains(t, err.Error(), tc.want)
		})
	}
}

func TestFromStrDigest(t *testing.T) {
	d, err := FromStrDigest("00ff", GitObjectID)
	require.NoError(t, err)

	assert.Equal(t, []byte{0x00, 0xff}, d.Sum)
	assert.Equal(t, GitObjectID, d.Algorithm)
	assert.Equal(t, "gitobjectid:00ff", d.String())
}

func TestFromStrDigestInvalidHex(t *testing.T) {
	_, err := FromStrDigest("xyz", SHA256)

	require.ErrorContains(t, err, "converting digest")
}
