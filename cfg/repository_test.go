package cfg

import (
	"io/ioutil"
	"os"
	"testing"
)

func Test_ExampleRepository_IsValid(t *testing.T) {
	r := ExampleRepository()
	if err := r.Validate(); err != nil {
		t.Error("example repository conf fails validation: ", err)
	}
}

func Test_ExampleRepository_WrittenAndReadCfgIsValid(t *testing.T) {
	tmpfileFD, err := ioutil.TempFile("", "baur")
	if err != nil {
		t.Fatal("opening tmpfile failed: ", err)
	}

	tmpfileName := tmpfileFD.Name()
	tmpfileFD.Close()
	defer os.Remove(tmpfileName)

	r := ExampleRepository()
	if err := r.Validate(); err != nil {
		t.Error("example conf fails validation: ", err)
	}

	if err := r.ToFile(tmpfileName, true); err != nil {
		t.Fatal("writing conf to file failed: ", err)
	}

	rRead, err := RepositoryFromFile(tmpfileName)
	if err != nil {
		t.Fatal("reading conf from file failed: ", err)
	}

	if err := rRead.Validate(); err != nil {
		t.Error("validating conf from file failed: ", err)
	}
}
