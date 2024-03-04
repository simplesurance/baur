package ostest

import (
	"os"
	"testing"
)

// Chdir changes the current working dir to dir.
// It registers a t.Cleanup function to change the working directory back to
// previous one.
func Chdir(t *testing.T, dir string) {
	t.Helper()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("could not get current working directory: %s", err)
	}
	t.Cleanup(func() {
		err := os.Chdir(oldDir)
		if err != nil {
			t.Fatalf("could not change back to previous working dir: %q: %s", oldDir, err)
		}
	})

	if err := os.Chdir(dir); err != nil {
		t.Fatalf("changing working directory to %q failed: %s", dir, err)
	}
}
