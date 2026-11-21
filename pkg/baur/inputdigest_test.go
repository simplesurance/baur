package baur

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v5/internal/digest/sha384"
)

// The golden values below were computed independently with sha384sum.
// If one of these tests fails, the digest of existing inputs changed.
// All cached task runs recorded in the database are invalidated by such a change.
const (
	goldenEnvVarFooBar = "sha384:0d19c2ca27f1f11c01aad5d70f7ed8727e077ca63926ef8139221fd0bbbdfcffea4cf1a0962c16fccf4d2fec5e8e8315" // "ENV: FOO=bar"
	goldenStringHello  = "sha384:59e1748777448c69de6b800d7a33bbfb9ff1b463e44354c3553bcdb9c666fa90125a3c79f90397bdf5f6a13de828684f" // "hello"

	// "P:app/f.txt" "R:" "E:0" "C:" + sha384("hello")
	goldenFile = "sha384:7aacb23ef9f751c09b125d366011ec07aec138d426344c8ab41cc1f964e341d0bfc2d21e7524320fb3e97718cdffe5f3"
	// same as goldenFile with "E:1"
	goldenFileExec = "sha384:014a0ecbdae1c1adf0ea6a6ee19df093c4dead80d036cd3e8ac4e2febdbbe09bc8ae770272ba497b5c4ee422f59ad35e"
	// same as goldenFile with "R:real/f.txt"
	goldenFileRealPath = "sha384:fa0456b20d4e74e9b41b37561d16c85142393bde368b95024e730719449b418b0072eb58b1648c00d5c10e9c42aff71b"

	goldenTotalEmpty         = "sha384:38b060a751ac96384cd9327eb1b1e36a21fdb71114be07434c0cc7bf63f6e1da274edebfe76f65fbd51ad2f14898b95b"
	goldenTotalEnvOnly       = "sha384:b28ec4932f889c33fa12da8e13959a98bf9d9984d79175a77541cb1e353103ce3e74a4947af75c0deb833fd1be54e24e"
	goldenTotalEnvStringFile = "sha384:519e386041eb74e984e4c7efaf36361b714472f7c990be71afb420db89a87fc295013b2f73b912ca7185bc08d05dcaae"
)

func mustDigest(t *testing.T, in Input) string {
	t.Helper()

	d, err := in.Digest()
	require.NoError(t, err)

	return d.String()
}

func newTestFile(t *testing.T, relPath, content string, exec bool, opts ...InputFileOpt) *InputFile {
	t.Helper()

	cd := sha384.New()
	require.NoError(t, cd.AddBytes([]byte(content)))

	opts = append([]InputFileOpt{WithContentDigest(cd.Digest())}, opts...)

	return NewInputFile("/abs/"+relPath, relPath, exec, opts...)
}

func TestInputEnvVar(t *testing.T) {
	t.Run("golden digest", func(t *testing.T) {
		assert.Equal(t, goldenEnvVarFooBar, mustDigest(t, NewInputEnvVar("FOO", "bar")))
	})

	t.Run("digest is stable across calls", func(t *testing.T) {
		in := NewInputEnvVar("FOO", "bar")
		first := mustDigest(t, in)
		second := mustDigest(t, in)
		assert.Equal(t, first, second)
	})

	t.Run("name and value both influence the digest", func(t *testing.T) {
		base := mustDigest(t, NewInputEnvVar("FOO", "bar"))
		assert.NotEqual(t, base, mustDigest(t, NewInputEnvVar("FOO", "baz")))
		assert.NotEqual(t, base, mustDigest(t, NewInputEnvVar("FOO2", "bar")))
	})

	t.Run("String and Name", func(t *testing.T) {
		in := NewInputEnvVar("FOO", "bar")
		assert.Equal(t, "$FOO", in.String())
		assert.Equal(t, "FOO", in.Name())
	})
}

func TestInputString(t *testing.T) {
	t.Run("golden digest", func(t *testing.T) {
		assert.Equal(t, goldenStringHello, mustDigest(t, NewInputString("hello")))
	})

	t.Run("different values give different digests", func(t *testing.T) {
		assert.NotEqual(t, mustDigest(t, NewInputString("a")), mustDigest(t, NewInputString("b")))
	})

	t.Run("String and Value", func(t *testing.T) {
		in := NewInputString("hello")
		assert.Equal(t, "string:hello", in.String())
		assert.Equal(t, "hello", in.Value())
	})

	t.Run("AsInputStrings keeps order and values", func(t *testing.T) {
		got := AsInputStrings("a", "b")
		require.Len(t, got, 2)
		assert.Equal(t, "string:a", got[0].String())
		assert.Equal(t, "string:b", got[1].String())
		assert.Empty(t, AsInputStrings())
	})
}

func TestInputFileDigestGolden(t *testing.T) {
	assert.Equal(t, goldenFile, mustDigest(t, newTestFile(t, "app/f.txt", "hello", false)))
	assert.Equal(t, goldenFileExec, mustDigest(t, newTestFile(t, "app/f.txt", "hello", true)))
	assert.Equal(t, goldenFileRealPath,
		mustDigest(t, newTestFile(t, "app/f.txt", "hello", false, WithRealpath("real/f.txt"))))
}

func TestInputFileDigestDependsOnEveryAttribute(t *testing.T) {
	base := mustDigest(t, newTestFile(t, "app/f.txt", "hello", false))

	tests := []struct {
		name string
		in   *InputFile
	}{
		{"content", newTestFile(t, "app/f.txt", "hellO", false)},
		{"path", newTestFile(t, "app/g.txt", "hello", false)},
		{"executable bit", newTestFile(t, "app/f.txt", "hello", true)},
		{"real path", newTestFile(t, "app/f.txt", "hello", false, WithRealpath("x"))},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.NotEqual(t, base, mustDigest(t, tc.in))
		})
	}
}

func TestInputFileDigestWithoutHasherReturnsError(t *testing.T) {
	_, err := NewInputFile("/abs/f", "f", false).Digest()
	require.Error(t, err)
}

func TestInputsDigest(t *testing.T) {
	env := func() Input { return NewInputEnvVar("FOO", "bar") }
	str := func() Input { return NewInputString("hello") }
	file := func() Input { return newTestFile(t, "app/f.txt", "hello", false) }

	t.Run("golden digests", func(t *testing.T) {
		d, err := NewInputs(nil).Digest()
		require.NoError(t, err)
		assert.Equal(t, goldenTotalEmpty, d.String())

		d, err = NewInputs([]Input{env()}).Digest()
		require.NoError(t, err)
		assert.Equal(t, goldenTotalEnvOnly, d.String())

		d, err = NewInputs([]Input{env(), str(), file()}).Digest()
		require.NoError(t, err)
		assert.Equal(t, goldenTotalEnvStringFile, d.String())
	})

	t.Run("does not depend on input order", func(t *testing.T) {
		d1, err := NewInputs([]Input{env(), str(), file()}).Digest()
		require.NoError(t, err)

		d2, err := NewInputs([]Input{file(), env(), str()}).Digest()
		require.NoError(t, err)

		assert.Equal(t, d1.String(), d2.String())
	})

	t.Run("a changed input changes the total digest", func(t *testing.T) {
		d1, err := NewInputs([]Input{env(), file()}).Digest()
		require.NoError(t, err)

		changed := newTestFile(t, "app/f.txt", "hello!", false)
		d2, err := NewInputs([]Input{env(), changed}).Digest()
		require.NoError(t, err)

		assert.NotEqual(t, d1.String(), d2.String())
	})

	t.Run("a missing input changes the total digest", func(t *testing.T) {
		d1, err := NewInputs([]Input{env(), file()}).Digest()
		require.NoError(t, err)

		d2, err := NewInputs([]Input{env()}).Digest()
		require.NoError(t, err)

		assert.NotEqual(t, d1.String(), d2.String())
	})

	t.Run("failing input digest returns an error", func(t *testing.T) {
		_, err := NewInputs([]Input{NewInputFile("/abs/f", "f", false)}).Digest()
		require.Error(t, err)
	})

	t.Run("Add invalidates the cached digest", func(t *testing.T) {
		in := NewInputs([]Input{env()})
		before, err := in.Digest()
		require.NoError(t, err)
		assert.Equal(t, goldenTotalEnvOnly, before.String())

		in.Add([]Input{str()})
		after, err := in.Digest()
		require.NoError(t, err)

		assert.NotEqual(t, before.String(), after.String())
		assert.Len(t, in.Inputs(), 2)
	})

	t.Run("Add without inputs keeps the cached digest", func(t *testing.T) {
		in := NewInputs([]Input{env()})
		_, err := in.Digest()
		require.NoError(t, err)

		in.Add(nil)
		assert.NotNil(t, in.digest)
	})

	t.Run("digest is cached", func(t *testing.T) {
		in := NewInputs([]Input{env()})
		d1, err := in.Digest()
		require.NoError(t, err)
		d2, err := in.Digest()
		require.NoError(t, err)

		assert.Same(t, d1, d2)
	})
}

func TestInputsSortOrdersByStringRepresentation(t *testing.T) {
	in := NewInputs([]Input{
		NewInputString("b"),
		NewInputEnvVar("A", "x"),
		NewInputString("a"),
	})

	in.Sort()

	got := make([]string, 0, len(in.Inputs()))
	for _, i := range in.Inputs() {
		got = append(got, i.String())
	}

	assert.Equal(t, []string{"$A", "string:a", "string:b"}, got)
}

func TestInputTask(t *testing.T) {
	inner := NewInputs([]Input{NewInputString("hello")})
	in := NewInputTask(&Task{ID: "app.build"}, inner)

	want, err := inner.Digest()
	require.NoError(t, err)

	assert.Equal(t, want.String(), mustDigest(t, in), "digest is the digest of the task's inputs")
	assert.Equal(t, "app.build", in.TaskID())
	assert.Equal(t, "task: app.build", in.String())
}
