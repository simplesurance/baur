package cfg

import (
	"io/ioutil"
	"os"
	"testing"
)

func Test_ExampleApp_IsValid(t *testing.T) {
	a := ExampleApp("shop")
	if err := a.Validate(); err != nil {
		t.Error("example app conf fails validation: ", err)
	}
}

func Test_ExampleApp_WrittenAndReadCfgIsValid(t *testing.T) {
	tmpfileFD, err := ioutil.TempFile("", "baur")
	if err != nil {
		t.Fatal("opening tmpfile failed: ", err)
	}

	tmpfileName := tmpfileFD.Name()
	tmpfileFD.Close()
	os.Remove(tmpfileName)

	a := ExampleApp("shop")
	if err := a.Validate(); err != nil {
		t.Error("example conf fails validation: ", err)
	}

	if err := a.ToFile(tmpfileName); err != nil {
		t.Fatal("writing conf to file failed: ", err)
	}

	rRead, err := AppFromFile(tmpfileName)
	if err != nil {
		t.Fatal("reading conf from file failed: ", err)
	}

	if err := rRead.Validate(); err != nil {
		t.Error("validating conf from file failed: ", err)
	}
}
