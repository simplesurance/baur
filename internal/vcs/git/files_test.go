package git

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsRegularFile(t *testing.T) {
	assert.False(t, ObjectTypeSymlink.IsRegularFile())
	assert.True(t, ObjectTypeExectuable.IsRegularFile())
	assert.True(t, ObjectTypeFile.IsRegularFile())
}

func TestIsSymlink(t *testing.T) {
	assert.True(t, ObjectTypeSymlink.IsSymlink())
	assert.False(t, ObjectTypeExectuable.IsSymlink())
	assert.False(t, ObjectTypeFile.IsSymlink())
}
