package fs

import (
	"os"
	"testing"
)

func TestPathIsInDirectories(t *testing.T) {
	tests := []struct {
		name         string
		pathArg      string
		directoryArg []string
		want         bool
	}{
		{
			"inPathLastDir",
			"/tmp/test.log",
			[]string{"/var/", "/tmp/"},
			true,
		},

		{
			"inPathFirstDir",
			"/tmp/test.log",
			[]string{"/tmp", "/var"},
			true,
		},

		{
			"DirInDir",
			"/tmp",
			[]string{"/etc", "/tmp/"},
			true,
		},

		{
			"NotInPaths",
			"/tmp/test.log",
			[]string{"/etc/", "/var/"},
			false,
		},

		{
			"NotInPathPartMatches",
			"/tmp/etc/test.log",
			[]string{"/etc/", "/var/"},
			false,
		},

		{
			"NoDirectoriesPassed",
			"/tmp/test.log",
			[]string{},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PathIsInDirectories(tt.pathArg, tt.directoryArg...); got != tt.want {
				t.Errorf("PathIsInDirectories() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPathIsInDirectories_Rel(t *testing.T) {
	err := os.Chdir("/tmp/")
	if err != nil {
		t.Fatal("chdir failed", err.Error())
	}

	inDir := PathIsInDirectories("log.txt", "/tmp")
	if !inDir {
		t.Error("PathIsInDirectories returned false when file is in relPath")
	}

	inDir = PathIsInDirectories("log.txt", "/etc")
	if inDir {
		t.Error("PathIsInDirectories returned true, when path of current dir was not passed")
	}
}
