package baur

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/simplesurance/baur/cfg"
	"github.com/simplesurance/baur/version"
)

func Test_ensureRepositoryCFGHasVersion(t *testing.T) {
	version.Version = "0.0.0"
	sver, err := version.SemVerFromString(version.Version)
	if err != nil {
		t.Fatal("setting version failed")
	}
	version.CurSemVer = *sver

	tmpfileFD, err := ioutil.TempFile("", "example")
	if err != nil {
		t.Fatal("opening tmpfile failed: ", err)
	}

	tmpfileName := tmpfileFD.Name()
	tmpfileFD.Close()
	defer os.Remove(tmpfileName)

	r := cfg.ExampleRepository()
	r.BaurVersion = ""

	err = checkCfgVersion(r, tmpfileName)
	if err != nil {
		t.Fatal(err)
	}

	if r.BaurVersion != version.Version {
		t.Errorf("version in cfg object is %q expected %q",
			r.BaurVersion, version.Version)
	}

	rNew, err := cfg.RepositoryFromFile(tmpfileName)
	if err != nil {
		t.Error(err)
	}

	if rNew.BaurVersion != r.BaurVersion {
		t.Errorf("version in written config is %q expected %q",
			rNew.BaurVersion, r.BaurVersion)
	}
}
