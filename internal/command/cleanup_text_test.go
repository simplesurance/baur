package command

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCleanupDbForceFlagUsageDoesNotRepeatTheTimeUnit(t *testing.T) {
	f := newCleanupDbCmd().Flags().Lookup("force")
	require.NotNil(t, f)

	assert.Contains(t, f.Usage, cleanupDbGracetime.String())
	assert.NotContains(t, f.Usage, cleanupDbGracetime.String()+" seconds")
}
