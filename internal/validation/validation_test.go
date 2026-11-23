package validation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStrID(t *testing.T) {
	valid := []string{"", "a", "my app", "app-1_x", "ünïcode", "a.b"}
	for _, id := range valid {
		t.Run("valid "+id, func(t *testing.T) {
			require.NoError(t, StrID(id))
		})
	}

	tests := []struct {
		name    string
		id      string
		wantErr string
	}{
		{"leading space", " a", "leading or trailing"},
		{"trailing space", "a ", "leading or trailing"},
		{"leading tab", "\ta", "leading or trailing"},
		{"trailing newline", "a\n", "leading or trailing"},
		{"only space", " ", "leading or trailing"},
		{"control character", "a\x00b", "non-printable"},
		{"newline inside", "a\nb", "non-printable"},
		{"tab inside", "a\tb", "non-printable"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := StrID(tc.id)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}
}
