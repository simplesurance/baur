package baur

import (
	"fmt"
	"path/filepath"
	"runtime"
)

var testdataDir string

func init() {
	_, testfile, _, ok := runtime.Caller(0)
	if !ok {
		panic("could not get test filename")
	}

	absPath, err := filepath.Abs(testfile)
	if err != nil {
		panic(fmt.Sprintf(
			" could not get absolute path of testfile (%s): %s",
			testfile, err))
	}
	testdataDir = filepath.Join(filepath.Dir(absPath), "testdata")
}
